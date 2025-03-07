package main

import (
	"net"
	"net/netip"
	"syscall"
	"time"

	cyw43439 "github.com/soypat/cyw43439"
)

func main() {
	var (
		fd     int
		err    error
		buffer [256]byte
		n      int
	)
	// Start with general device initialization. This is already implemented.
	spi, cs, WL_REG_ON, irq := cyw43439.PicoWSpi(0)
	dev := cyw43439.NewDev(spi, cs, WL_REG_ON, irq, irq)
	err = dev.Init(cyw43439.DefaultConfig(false))
	if err != nil {
		panic(err.Error())
	}
	// Everything beyond this point must be implemented!
	err = dev.ConnectWifi(cyw43439.DefaultWifiConfig())
	if err != nil {
		panic(err.Error())
	}

	// Create a file descriptor for the TCP socket.
	fd, err = dev.Socket(syscall.AF_INET, syscall.SOCK_STREAM, syscall.IPPROTO_TCP)
	if err != nil {
		panic(err)
	}

	// Define the TCP listener IP address and port. This would be your PC's
	// IP address and the port on which you are running a test server that will
	// receive the Pico W's TCP communications.
	listener := net.TCPAddrFromAddrPort(netip.MustParseAddrPort("192.168.1.1:8080"))
	// Performs TCP connection to server.
	err = dev.Connect(fd, "", listener.IP, listener.Port)
	if err != nil {
		dev.Close(fd)
		panic(err)
	}

	// We loop forever printing out what we receive from the server and
	// sending our own message. This should run forever as long as the server
	// remains up and keeps the connection open.
	for {
		deadline := time.Now().Add(time.Second)
		_, err = dev.Send(fd, []byte("Hello from the Pico W!\n"), 0, deadline)
		if err != nil {
			println("error:", err.Error())
		}
		n, err = dev.Recv(fd, buffer[:], 0, deadline)
		if err == nil {
			println("received message:", string(buffer[:n]))
		}
		time.Sleep(200 * time.Millisecond)
	}
}
