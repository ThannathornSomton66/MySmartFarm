package main

import (
	"encoding/json"
	"log/slog"
	"machine"
	"net/netip"
	"strconv"
	"strings"
	"time"

	_ "embed"

	"IOTDEVICE/controller/common"

	"github.com/soypat/seqs"
	"github.com/soypat/seqs/httpx"
	"github.com/soypat/seqs/stacks"
	"tinygo.org/x/drivers/dht"
)

const (
	connTimeout   = 5 * time.Second
	tcpbufsize    = 2030
	serverAddrStr = "192.168.220.181:3000"
	ourHostname   = "tinygo-http-client"
	sendInterval  = 1 * time.Minute
)

func main() {
	machine.Serial.Configure(machine.UARTConfig{})
	machine.InitADC()

	logger := slog.New(slog.NewTextHandler(machine.Serial, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	dhtPin := machine.GPIO15
	dhtSensor := dht.New(dhtPin, dht.DHT22)

	soil := machine.ADC{Pin: machine.GPIO26}
	soil.Configure(machine.ADCConfig{})

	_, stack, _, err := common.SetupWithDHCP(common.SetupConfig{
		Hostname: ourHostname,
		Logger:   logger,
		TCPPorts: 1,
		UDPPorts: 1,
	})
	if err != nil {
		panic("setup DHCP:" + err.Error())
	}

	svAddr, err := netip.ParseAddrPort(serverAddrStr)
	if err != nil {
		panic("parsing server address:" + err.Error())
	}
	routerHW, err := common.ResolveHardwareAddr(stack, svAddr.Addr())
	if err != nil {
		panic("router hwaddr resolving:" + err.Error())
	}

	conn, err := stacks.NewTCPConn(stack, stacks.TCPConnConfig{
		TxBufSize: tcpbufsize,
		RxBufSize: tcpbufsize,
	})
	if err != nil {
		panic("conn create:" + err.Error())
	}

	closeConn := func(reason string) {
		slog.Info("closing TCP connection", slog.String("reason", reason))
		conn.Close()
		for !conn.State().IsClosed() {
			slog.Info("waiting for TCP close", slog.String("state", conn.State().String()))
			time.Sleep(time.Second)
		}
	}

	for {
		temp, hum, err := dhtSensor.Measurements()
		if err != nil {
			logger.Error("DHT22 read error", slog.String("err", err.Error()))
			time.Sleep(sendInterval)
			continue
		}

		raw := soil.Get()
		soilPercent := 100.0 - (float64(raw) * 100.0 / 65535.0)

		slog.Info("Sensor data",
			slog.Float64("temp", float64(temp)/10.0),
			slog.Float64("hum", float64(hum)/10.0),
			slog.Float64("soil", soilPercent),
			slog.Float64("raw", float64(raw)),
		)

		payload := []byte(`{
			"deviceID": "sensor-001",
			"temperature": ` + strconv.FormatFloat(float64(temp)/10.0, 'f', 1, 64) + `,
			"humidity": ` + strconv.FormatFloat(float64(hum)/10.0, 'f', 1, 64) + `,
			"soil": ` + strconv.FormatFloat(soilPercent, 'f', 1, 64) + `
		}`)

		var req httpx.RequestHeader
		req.SetRequestURI("/api/v1/data")
		req.SetMethod("POST")
		req.SetHost(svAddr.Addr().String())

		headerBytes := req.Header()
		headerWithoutCRLF := headerBytes[:len(headerBytes)-2]
		contentLen := strconv.Itoa(len(payload))
		extraHeaders := []byte("Content-Type: application/json\r\n" +
			"Content-Length: " + contentLen + "\r\n\r\n")

		postReq := make([]byte, 0, len(headerWithoutCRLF)+len(extraHeaders)+len(payload))
		postReq = append(postReq, headerWithoutCRLF...)
		postReq = append(postReq, extraHeaders...)
		postReq = append(postReq, payload...)

		slog.Info("dialing server", slog.String("addr", serverAddrStr))
		clientPort := uint16(time.Now().UnixNano()%60000 + 1024)
		clientAddr := netip.AddrPortFrom(stack.Addr(), clientPort)
		err = conn.OpenDialTCP(clientAddr.Port(), routerHW, svAddr, seqs.Value(time.Now().UnixNano()%65535))
		if err != nil {
			slog.Error("opening TCP", slog.String("err", err.Error()))
			closeConn("OpenDialTCP error")
			time.Sleep(sendInterval)
			continue
		}

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

		_, err = conn.Write(postReq)
		if err != nil {
			slog.Error("writing request", slog.String("err", err.Error()))
			closeConn("write-error")
			time.Sleep(sendInterval)
			continue
		}

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
		closeConn("end-of-loop")

		var jsonResponse struct {
			IntervalSeconds int `json:"intervalSeconds"`
		}
		respStr := string(rxBuf[:n])
		splitIdx := strings.Index(respStr, "\r\n\r\n")
		if splitIdx == -1 {
			slog.Warn("HTTP response malformed, no header/body split found")
			time.Sleep(5 * time.Minute)
			continue
		}
		body := respStr[splitIdx+4:]
		err = json.Unmarshal([]byte(body), &jsonResponse)
		if err != nil || jsonResponse.IntervalSeconds <= 0 {
			slog.Warn("Failed to parse intervalSeconds or missing, using default")
			time.Sleep(5 * time.Minute)
			continue
		}

		sleepDuration := time.Duration(jsonResponse.IntervalSeconds) * time.Second
		slog.Info("Sleeping until next send", slog.Duration("sleep", sleepDuration))
		time.Sleep(sleepDuration)
	}
}
