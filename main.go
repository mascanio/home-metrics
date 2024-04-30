package main

import (
	"encoding/binary"
	"flag"
	"log"
	"net/http"
	"strings"

	"tinygo.org/x/bluetooth"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	address     string
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
				address:     device.Address.String(),
			}
		}
	})
	if err != nil {
		log.Fatalln("Failed to scan: ", err)
	}
	log.Println("Scan stopped")
}

var (
	salon_temperature = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "salon_temperature",
		Help: "Temperature of the salon",
	})
	salon_humidity = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "salon_humidity",
		Help: "Humidity of the salon",
	})
	sotano_temperature = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "sotano_temperature",
		Help: "Temperature of the sotano",
	})
	sotano_humidity = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "sotano_humidity",
		Help: "Humidity of the sotano",
	})
)

func main() {
	var (
		companyID      uint
		mac            string
		prometheusPort string
	)
	flag.UintVar(&companyID, "companyID", 60552, "Company ID of the device")
	flag.StringVar(&mac, "mac", "A4:C1:38", "Part of the MAC address start of the device")
	flag.StringVar(&prometheusPort, "prometheusPort", "2112", "Port for prometheus metrics")
	flag.Parse()

	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":"+prometheusPort, nil)

	if err := adapter.Enable(); err != nil {
		log.Fatalln("Failed to enable the bluetooth adapter: ", err)
	}

	metricsChan := make(chan metric)
	defer close(metricsChan)
	go scanMetrics(mac, uint16(companyID), metricsChan)
	for {
		metric := <-metricsChan
		if metric.address == "A4:C1:38:5F:A4:E6" {
			salon_temperature.Set(metric.temperature)
			salon_humidity.Set(metric.humidity)
		} else if metric.address == "A4:C1:38:B8:1A:4C" {
			sotano_temperature.Set(metric.temperature)
			sotano_humidity.Set(metric.humidity)
		}
		log.Printf("Temperature: %.2f, Humidity: %.2f, Address: %v\n", metric.temperature, metric.humidity, metric.address)
	}
}
