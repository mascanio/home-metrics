package main

import (
	"encoding/binary"
	"flag"
	"log"
	"strings"

	"tinygo.org/x/bluetooth"
)

var adapter = bluetooth.DefaultAdapter

func decodeTemperature(in uint64) float64 {
	if in&0x800000 != 0 {
		in &= 0x7FFFFF
		return float64(uint64(in/1000)) / -10.0
	}
	return float64(uint64(in/1000)) / 10.0
}

func decodeHumid(in uint64) float64 {
	in &= 0x7FFFFF
	return float64(uint64(in%1000)) / 10.0
}

type metric struct {
	temperature float64
	humidity    float64
}

func scanMetrics(address string, CompanyID uint16, metricChan chan<- metric) {
	// Start scanning.
	log.Println("Scanning for devices...")
	err := adapter.Scan(func(adapter *bluetooth.Adapter, device bluetooth.ScanResult) {
		if strings.Contains(device.Address.String(), address) && device.ManufacturerData()[0].CompanyID == CompanyID {
			rawData := device.ManufacturerData()[0].Data
			data := append([]byte{0, 0, 0, 0}, rawData[:len(rawData)-2]...)
			n := binary.BigEndian.Uint64(data)
			metricChan <- metric{
				temperature: decodeTemperature(n),
				humidity:    decodeHumid(n),
			}
		}
	})
	if err != nil {
		log.Fatalln("Failed to scan: ", err)
	}
	log.Println("Scan stopped")
}

func main() {
	var (
		companyID uint
		mac       string
	)
	flag.UintVar(&companyID, "companyID", 60552, "Company ID of the device")
	flag.StringVar(&mac, "mac", "A4:C1:38", "Part of the MAC address start of the device")
	flag.Parse()

	if err := adapter.Enable(); err != nil {
		log.Fatalln("Failed to enable the bluetooth adapter: ", err)
	}

	metricsChan := make(chan metric)
	defer close(metricsChan)
	go scanMetrics(mac, uint16(companyID), metricsChan)
	for {
		metric := <-metricsChan
		log.Printf("Temperature: %.2f, Humidity: %.2f\n", metric.temperature, metric.humidity)
	}
}
