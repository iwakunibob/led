// This program accesses GPIO of RPI4 or RP400
// References:
// https://github.com/warthog618/gpiod/blob/master/example/blinker/blinker.go
//

// A simple example that toggles an output pin.
package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/warthog618/gpiod"
)

const totOutpLines = 11 // Total number of output lines used
const cycles = 64       // Number of cycles for led changes
const cycsec = 4        // Number of cycles per second

func getMode() string {
	fmt.Print(
		"\nWelcome to the LED toggle program.\n",
		"Select the mode from list below:",
		"\n    R = Right shift bit",
		"\n    L = Left shift bit",
		"\n    C = Count in binary",
		"\n    K = Knocker lights",
		"\n    T = Tri-color light",
		"\n    Q = Quit\n>")
	for {
		reader := bufio.NewReader(os.Stdin)
		mode, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		mode = strings.ToUpper(mode[0:1])
		if mode == "R" || mode == "L" || mode == "C" || mode == "K" || mode == "T" || mode == "Q" {
			return mode
		} else {
			fmt.Print("Error you must type one of letters listed above\n>")
		}
	}
}

func setOutputs(outBits int, outLns [totOutpLines]*gpiod.Line) {
	m := 1
	for i := 0; i < totOutpLines; i++ {
		v := m & outBits
		outLns[i].SetValue(v)
		m = m << 1
	}
}

func main() {
	piPortsOut := [totOutpLines]int{27, 26, 25, 24, 23, 22, 17, 16, 12, 20, 21}
	var outpLines [totOutpLines]*gpiod.Line
	var outp int

	// Capture exit signals to ensure pin is reverted to input on exit.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	// Setup GPIO Ports as Outputs
	for i, port := range piPortsOut {
		line, err := gpiod.RequestLine("gpiochip0", port, gpiod.AsOutput(0))
		if err != nil {
			panic(err)
		}
		outpLines[i] = line
		defer func() {
			line.Reconfigure(gpiod.AsInput)
			line.Close()
		}()
	}

	for {
		setOutputs(0, outpLines)
		mode := getMode()
		if mode == "Q" {
			break
		} else if mode == "R" {
			outp = 0x80 // initial outputs
			for j := 0; j <= cycles; j++ {
				select {
				case <-time.After(time.Second / cycsec):
					if outp == 0 && j < cycles {
						outp = 0x80
					}
					fmt.Printf("%03d: Output Byte = 0x%02X = %08b\n", j, outp, outp)
					setOutputs(outp, outpLines)
					outp = outp >> 1
				case <-quit:
					return
				}
			}
		} else if mode == "L" {
			outp = 0x01 // initial outputs
			for j := 0; j <= cycles; j++ {
				select {
				case <-time.After(time.Second / cycsec):
					if outp > 0x80 {
						if j == cycles {
							outp = 0
						} else {
							outp = 0x01
						}
					}
					fmt.Printf("%03d: Output Byte = 0x%02X = %08b\n", j, outp, outp)
					setOutputs(outp, outpLines)
					outp = outp << 1
				case <-quit:
					return
				}
			}
		} else if mode == "C" {
			outp = 0 // initial outputs
			for j := 0; j <= 256; j++ {
				select {
				case <-time.After(time.Second / cycsec):
					if outp > 0xFF {
						outp = 0
					}
					fmt.Printf("%03d: Output Byte = 0x%02X = %08b\n", j, outp, outp)
					setOutputs(outp, outpLines)
					outp++
				case <-quit:
					return
				}
			}
		} else if mode == "K" {
			outseq := [...]int{0x81, 0x42, 0x24, 0x18, 0x24, 0x42} // sequencer
			i := 0

			for j := 0; j < cycles; j++ {
				select {
				case <-time.After(time.Second / cycsec):
					outp = outseq[i]
					fmt.Printf("%3d: Output Byte = %02X = %08b\n", j, outp, outp)
					setOutputs(outp, outpLines)
					i++
					if i == len(outseq) {
						i = 0
					}
				case <-quit:
					return
				}
			}
		} else if mode == "T" {
			outseq := [...]int{0x180, 0x220, 0x3A0, 0x410, 0x590, 0x630, 0x7B0, 0} // sequencer
			i := 0
			setOutputs(0x700, outpLines)
			for j := 0; j < cycles; j++ {
				select {
				case <-time.After(time.Second * cycsec):
					outp = outseq[i]
					fmt.Printf("%3d: Output Byte = %04X = %016b\n", j, outp, outp)
					setOutputs(outp, outpLines)
					i++
					if i == len(outseq) {
						i = 0
					}
				case <-quit:
					return
				}
			}
		}
	}
}
