package main

import (
	"time"

	cyw43439 "IOTDEVICE"
	"IOTDEVICE/controller/common"
)

var (
	dev          *cyw43439.Device
	lastLedState bool
)

func main() {
	// Set up the CYW43439 device (includes firmware upload, Wi-Fi init, etc.)
	_, _, devptr, err := common.SetupWithDHCP(common.SetupConfig{
		Hostname: "debug-pico",
	})
	if err != nil {
		panic("setup failed: " + err.Error())
	}
	dev = devptr

	// Blink LED on GPIO 0
	lastLedState = true
	dev.GPIOSet(0, lastLedState)
	println("Initial LED state:", lastLedState)

	for {
		time.Sleep(500 * time.Millisecond)
		lastLedState = !lastLedState
		dev.GPIOSet(0, lastLedState)
		println("LED toggled to:", lastLedState)
	}
}
