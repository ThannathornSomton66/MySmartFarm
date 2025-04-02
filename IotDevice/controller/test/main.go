package main

import (
	"encoding/json"
	"log/slog"
	"machine"
	"math/rand"
	"net/netip"
	"strconv"
	"time"

	_ "embed"

	"IOTDEVICE/controller/common"

	"github.com/soypat/seqs"
	"github.com/soypat/seqs/httpx"
	"github.com/soypat/seqs/stacks"
)

const (
	connTimeout   = 5 * time.Second
	tcpbufsize    = 2030 // MTU - ethhdr - iphdr - tcphdr
	serverAddrStr = "192.168.220.181:3000"
	ourHostname   = "tinygo-http-client"

	// Default interval (1 minute for testing, change to 1 hour as needed)
	sendInterval = 1 * time.Minute
)

func main() {
	logger := slog.New(slog.NewTextHandler(machine.Serial, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// 1) Bring up network with DHCP.
	_, stack, _, err := common.SetupWithDHCP(common.SetupConfig{
		Hostname: ourHostname,
		Logger:   logger,
		TCPPorts: 1,
		UDPPorts: 1,
	})
	if err != nil {
		panic("setup DHCP:" + err.Error())
	}

	// 2) Parse the server address and resolve hardware (MAC).
	svAddr, err := netip.ParseAddrPort(serverAddrStr)
	if err != nil {
		panic("parsing server address:" + err.Error())
	}
	routerHW, err := common.ResolveHardwareAddr(stack, svAddr.Addr())
	if err != nil {
		panic("router hwaddr resolving:" + err.Error())
	}

	// 3) Prepare a TCPConn for re-use. We'll open/close it each time in the loop.
	conn, err := stacks.NewTCPConn(stack, stacks.TCPConnConfig{
		TxBufSize: tcpbufsize,
		RxBufSize: tcpbufsize,
	})
	if err != nil {
		panic("conn create:" + err.Error())
	}

	// Utility function to close the connection with logs:
	closeConn := func(reason string) {
		slog.Info("closing TCP connection", slog.String("reason", reason))
		conn.Close()
		for !conn.State().IsClosed() {
			slog.Info("waiting for TCP close", slog.String("state", conn.State().String()))
			time.Sleep(time.Second)
		}
	}

	// 4) Initialize random generator for synthetic sensor data.
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Request setup
	var req httpx.RequestHeader
	req.SetRequestURI("/api/v1/data")
	req.SetMethod("POST")
	req.SetHost(svAddr.Addr().String())

	headerBytes := req.Header()
	if len(headerBytes) < 4 {
		panic("unexpected short header from httpx.RequestHeader")
	}
	headerWithoutCRLF := headerBytes[:len(headerBytes)-2]

	// 5) Main loop: send data, then sleep with time correction
	for {
		//startTime := time.Now() // Record when the request starts

		// Generate fake sensor values
		temperature := 20.0 + rng.Float64()*10.0
		humidity := 50.0 + rng.Float64()*30.0

		// Build JSON payload
		payload := []byte(`{
  "deviceID": "sensor-001",
  "temperature": ` + strconv.FormatFloat(temperature, 'f', 2, 64) + `,
  "humidity": ` + strconv.FormatFloat(humidity, 'f', 2, 64) + `
}`)

		contentLen := strconv.Itoa(len(payload))
		extraHeaders := []byte("Content-Type: application/json\r\n" +
			"Content-Length: " + contentLen + "\r\n\r\n")

		// Combine [headerWithoutCRLF + extraHeaders + payload]
		postReq := make([]byte, 0, len(headerWithoutCRLF)+len(extraHeaders)+len(payload))
		postReq = append(postReq, headerWithoutCRLF...)
		postReq = append(postReq, extraHeaders...)
		postReq = append(postReq, payload...)

		// 6) Dial and send the request.
		slog.Info("dialing server", slog.String("addr", serverAddrStr))
		clientPort := uint16(rng.Intn(65535-1024) + 1024)
		clientAddr := netip.AddrPortFrom(stack.Addr(), clientPort)
		err = conn.OpenDialTCP(clientAddr.Port(), routerHW, svAddr, seqs.Value(rng.Intn(65535-1024)+1024))
		if err != nil {
			slog.Error("opening TCP", slog.String("err", err.Error()))
			closeConn("OpenDialTCP error")
			time.Sleep(sendInterval)
			continue
		}

		// Wait for established state (or give up).
		retries := 50
		for conn.State() != seqs.StateEstablished && retries > 0 {
			time.Sleep(100 * time.Millisecond)
			retries--
		}
		if retries == 0 {
			slog.Error("tcp establish", "err", "retry limit exceeded")
			closeConn("establish-timeout")
			time.Sleep(sendInterval)
			continue
		}

		// Write request.
		_, err = conn.Write(postReq)
		if err != nil {
			slog.Error("writing request", slog.String("err", err.Error()))
			closeConn("write-error")
			time.Sleep(sendInterval)
			continue
		}

		// 7) Read the response.
		rxBuf := make([]byte, 4096)
		conn.SetDeadline(time.Now().Add(connTimeout))
		n, err := conn.Read(rxBuf)
		if n == 0 && err != nil {
			slog.Error("reading response", slog.String("err", err.Error()))
			closeConn("read-error")
			time.Sleep(sendInterval)
			continue
		} else if n == 0 {
			slog.Error("no response from server")
			closeConn("no response")
			time.Sleep(sendInterval)
			continue
		}

		println("Raw response from server:")
		println(string(rxBuf[:n]))

		// Parse server response
		serverTime := parseServerTime(rxBuf[:n])

		// Close the connection gracefully
		closeConn("done")

		// Align to next 5-minute mark based on server time
		if !serverTime.IsZero() {
			// You can adjust this to 1 * time.Hour if needed
			const alignInterval = 5 * time.Minute

			// Find next aligned time
			truncated := serverTime.Truncate(alignInterval)
			nextTime := truncated.Add(alignInterval)
			sleepDuration := nextTime.Sub(serverTime)

			if sleepDuration < 0 {
				sleepDuration = alignInterval // fallback in rare case of wrap
			}

			slog.Info("Aligning to next 5-min mark",
				slog.String("now", serverTime.Format(time.RFC3339)),
				slog.String("next", nextTime.Format(time.RFC3339)),
				slog.Duration("sleep", sleepDuration),
			)

			time.Sleep(sleepDuration)
		} else {
			slog.Warn("No server time available, using default fallback interval")
			time.Sleep(5 * time.Minute)
		}

	}
}

// Function to parse server time from response JSON
func parseServerTime(response []byte) time.Time {
	// Find where the headers end: look for "\r\n\r\n"
	headerEnd := []byte("\r\n\r\n")
	bodyStart := 0
	if i := indexOf(response, headerEnd); i != -1 {
		bodyStart = i + len(headerEnd)
	}

	body := response[bodyStart:]

	var jsonResponse struct {
		ServerTime string `json:"serverTime"`
	}
	err := json.Unmarshal(body, &jsonResponse)
	if err != nil {
		slog.Error("Failed to parse server time", slog.String("err", err.Error()))
		println("RAW BODY:")
		println(string(body))
		return time.Time{}
	}

	serverTime, err := time.Parse(time.RFC3339, jsonResponse.ServerTime)
	if err != nil {
		slog.Error("Invalid server time format", slog.String("err", err.Error()))
		return time.Time{}
	}

	slog.Info("Parsed server time", slog.String("server_time", serverTime.String()))
	return serverTime
}

func indexOf(buf []byte, sep []byte) int {
	for i := 0; i <= len(buf)-len(sep); i++ {
		match := true
		for j := range sep {
			if buf[i+j] != sep[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
