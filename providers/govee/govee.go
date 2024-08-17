package govee

import (
	"encoding/binary"
	"log"
	"strings"
	"time"

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

func getDeviceName(addr string) string {
	switch addr {
	case "A4:C1:38:5F:A4:E6":
		return DEVICE_SALON
	case "A4:C1:38:B8:1A:4C":
		return DEVICE_TALLER
	default:
		return "unknown"
	}
}

func (g *Govee) shouldProcessDevice(device bluetooth.ScanResult) bool {
	return strings.Contains(device.Address.String(), g.config.Mac) &&
		device.ManufacturerData()[0].CompanyID == g.config.CompanyID
}

func parseTempHum(rawData []byte, deviceAddr string) metrics.TemperatureHumidity {
	data := append([]byte{0, 0, 0, 0}, rawData[:len(rawData)-2]...)
	n := binary.BigEndian.Uint64(data)
	return metrics.TemperatureHumidity{
		Temperature: decodeTemperature(n),
		Humidity:    decodeHumid(n),
		Device:      getDeviceName(deviceAddr),
	}
}

func sc(g *Govee, metricChan chan<- metrics.TemperatureHumidity,
	adapter *bluetooth.Adapter, scanRes bluetooth.ScanResult) {
	log.Println(scanRes.LocalName(), " ", scanRes.Address)
	if g.shouldProcessDevice(scanRes) {
		_ = adapter.StopScan()
		addrStr := scanRes.Address.String()
		rawData := scanRes.ManufacturerData()[0].Data
		metric := parseTempHum(rawData, addrStr)
		log.Printf("Writting to channel %v %v %v\n", metric.Device,
			metric.Temperature, metric.Humidity)
		metricChan <- metric
	}
}

func (g *Govee) ScanMetrics(metricChan chan<- metrics.TemperatureHumidity) {
	doneChan := make(chan struct{})
	defer close(doneChan)
	for {
		go func() {
			defer func() {
				_ = g.adapter.StopScan()
				log.Println("Scan stopped")
				doneChan <- struct{}{}
				if recover() != nil {
					log.Println("Recovered from panic")
				}
			}()
			log.Println("Scanning for devices...")
			err := g.adapter.Scan(func(a *bluetooth.Adapter, sr bluetooth.ScanResult) {
				sc(g, metricChan, a, sr)
			})
			if err != nil {
				log.Println("Failed to scan: ", err)
			} else {
				log.Println("Scanned")
			}
		}()
		<-doneChan
		time.Sleep(time.Second * 15)
	}
}
