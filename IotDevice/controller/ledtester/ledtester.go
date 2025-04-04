package main

import (
	"machine"
	"time"
)

func main() {
	led := machine.LED
	led.Configure(machine.PinConfig{Mode: machine.PinOutput})

	for {
		led.High()
		time.Sleep(500 * time.Millisecond)
		println("12")

		led.Low()
		time.Sleep(500 * time.Millisecond)
	}
}
