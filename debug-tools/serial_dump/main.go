package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"time"

	"go.bug.st/serial"
)

func main() {
	portName := flag.String("port", "/dev/ttyACM0", "Serial port to open")
	baudRate := flag.Int("baud", 57600, "Baud rate")
	flag.Parse()

	mode := &serial.Mode{
		BaudRate: *baudRate,
	}

	log.Printf("Opening %s at %d baud...", *portName, *baudRate)
	port, err := serial.Open(*portName, mode)
	if err != nil {
		log.Fatalf("Failed to open port: %v", err)
	}
	defer port.Close()
	log.Println("Port opened successfully. Listening for data...")

	buf := make([]byte, 100)
	totalBytes := 0
	lastLog := time.Now()

	for {
		n, err := port.Read(buf)
		if err != nil {
			log.Fatalf("Read error: %v", err)
		}
		if n == 0 {
			continue
		}

		totalBytes += n

		// Print a hex dump of the received data
		fmt.Printf("Read %d bytes:\n%s\n", n, hex.Dump(buf[:n]))

		// Scan for MAVLink start bytes
		for _, b := range buf[:n] {
			if b == 0xFE {
				log.Println(">>> FOUND MAVLINK v1 HEADER (0xFE) <<<")
			}
			if b == 0xFD {
				log.Println(">>> FOUND MAVLINK v2 HEADER (0xFD) <<<")
			}
		}

		if time.Since(lastLog) > 5*time.Second {
			log.Printf("Total bytes received so far: %d", totalBytes)
			lastLog = time.Now()
		}
	}
}
