package govee

import (
	"encoding/binary"
	"log"
	"strings"

	"github.com/mascanio/home-metrics/metrics"
	"tinygo.org/x/bluetooth"
)

type Config struct {
	CompanyID uint16 `yaml:"company-id"`
	Mac       string
}

type Govee struct {
	adapter *bluetooth.Adapter
	config  Config
}

func New(config Config) (Govee, error) {
	rv := Govee{config: config, adapter: bluetooth.DefaultAdapter}
	if err := rv.adapter.Enable(); err != nil {
		return rv, err
	}
	return rv, nil
}

var (
	DEVICE_SALON  = "salon"
	DEVICE_TALLER = "taller"
)

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

func (g *Govee) ScanMetrics(metricChan chan<- metrics.TemperatureHumidity) {
	// Start scanning.
	for {
		log.Println("Scanning for devices...")
		err := g.adapter.Scan(func(adapter *bluetooth.Adapter, device bluetooth.ScanResult) {
			if strings.Contains(device.Address.String(), g.config.Mac) && device.ManufacturerData()[0].CompanyID == g.config.CompanyID {
				adapter.StopScan()
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
		g.adapter.StopScan()
		log.Println("Scan stopped")
	}
}
