package govee

import (
	"encoding/binary"
	"log"
	"strings"

	"github.com/mascanio/home-metrics/metrics"
	"tinygo.org/x/bluetooth"
)

var (
	DEVICE_SALON  = "salon"
	DEVICE_TALLER = "taller"
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

func getDeviceName(device bluetooth.ScanResult) string {
	switch device.Address.String() {
	case "A4:C1:38:5F:A4:E6":
		return DEVICE_SALON
	case "A4:C1:38:B8:1A:4C":
		return DEVICE_TALLER
	default:
		return "unknown"
	}
}

func ScanMetrics(address string, CompanyID uint16, metricChan chan<- metrics.TemperatureHumidity) {
	if err := adapter.Enable(); err != nil {
		log.Fatalln("Failed to enable the bluetooth adapter: ", err)
	}
	// Start scanning.
	log.Println("Scanning for devices...")
	err := adapter.Scan(func(adapter *bluetooth.Adapter, device bluetooth.ScanResult) {
		if strings.Contains(device.Address.String(), address) && device.ManufacturerData()[0].CompanyID == CompanyID {
			rawData := device.ManufacturerData()[0].Data
			data := append([]byte{0, 0, 0, 0}, rawData[:len(rawData)-2]...)
			n := binary.BigEndian.Uint64(data)
			log.Printf("Writting to channel %v %v %v\n", getDeviceName(device), decodeTemperature(n), decodeHumid(n))
			metricChan <- metrics.TemperatureHumidity{
				Temperature: decodeTemperature(n),
				Humidity:    decodeHumid(n),
				Device:      getDeviceName(device),
			}
		}
	})
	if err != nil {
		log.Fatalln("Failed to scan: ", err)
	}
	log.Println("Scan stopped")
}
