package main

import (
	"bufio"
	"io"
	"log/slog"
	"machine"
	"net/netip"
	"time"

	_ "embed"

	cyw43439 "IOTDEVICE"
	"IOTDEVICE/controller/common"

	"github.com/soypat/seqs/httpx"
	"github.com/soypat/seqs/stacks"
)

const connTimeout = 3 * time.Second
const maxconns = 3
const tcpbufsize = 2030
const hostname = "http-pico"

var (
	dev          *cyw43439.Device
	lastLedState bool
)

func HTTPHandler(respWriter io.Writer, resp *httpx.ResponseHeader, req *httpx.RequestHeader) {
	uri := string(req.RequestURI())

	resp.SetConnectionClose()
	//resp.SetHeader("Access-Control-Allow-Origin", "*") // 👈 Allow browser CORS

	switch uri {
	case "/relay/on":
		println("Turning relay ON")
		lastLedState = true
		dev.GPIOSet(0, lastLedState)

		// Manually write response
		resp.SetStatusCode(200)
		resp.SetContentType("text/plain")
		respWriter.Write(resp.Header())
		respWriter.Write([]byte("OK"))
		return

	case "/relay/off":
		println("Turning relay OFF")
		lastLedState = false
		dev.GPIOSet(0, lastLedState)

		resp.SetStatusCode(200)
		resp.SetContentType("text/plain")
		respWriter.Write(resp.Header())
		respWriter.Write([]byte("OK"))
		return

	default:
		println("Path not found:", uri)
		resp.SetStatusCode(404)
		resp.SetContentLength(0)
		respWriter.Write(resp.Header())
	}
}

func main() {
	logger := slog.New(slog.NewTextHandler(machine.Serial, &slog.HandlerOptions{
		Level: slog.LevelDebug - 2,
	}))
	time.Sleep(time.Second)

	_, stack, devlocal, err := common.SetupWithDHCP(common.SetupConfig{
		Hostname: "TCP-pico",
		Logger:   logger,
		TCPPorts: 1,
	})
	dev = devlocal
	if err != nil {
		panic("setup DHCP: " + err.Error())
	}

	const listenPort = 80
	listenAddr := netip.AddrPortFrom(stack.Addr(), listenPort)
	listener, err := stacks.NewTCPListener(stack, stacks.TCPListenerConfig{
		MaxConnections: maxconns,
		ConnTxBufSize:  tcpbufsize,
		ConnRxBufSize:  tcpbufsize,
	})
	if err != nil {
		panic("listener create: " + err.Error())
	}
	err = listener.StartListening(listenPort)
	if err != nil {
		panic("listener start: " + err.Error())
	}

	var req httpx.RequestHeader
	var resp httpx.ResponseHeader
	buf := bufio.NewReaderSize(nil, 1024)

	logger.Info("listening", slog.String("addr", "http://"+listenAddr.String()))

	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Error("listener accept:", slog.String("err", err.Error()))
			time.Sleep(time.Second)
			continue
		}

		logger.Info("new connection", slog.String("remote", conn.RemoteAddr().String()))
		err = conn.SetDeadline(time.Now().Add(connTimeout))
		if err != nil {
			conn.Close()
			logger.Error("conn set deadline:", slog.String("err", err.Error()))
			continue
		}

		buf.Reset(conn)
		err = req.Read(buf)
		if err != nil {
			logger.Error("hdr read:", slog.String("err", err.Error()))
			conn.Close()
			continue
		}

		resp.Reset()
		HTTPHandler(conn, &resp, &req)
		conn.Close()
	}
}
