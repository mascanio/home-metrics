package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/mascanio/home-metrics/metrics"
	govee "github.com/mascanio/home-metrics/providers/govee"
)

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

	temperatureHumidityChan := make(chan metrics.TemperatureHumidity)
	defer close(temperatureHumidityChan)
	go govee.ScanMetrics(mac, uint16(companyID), temperatureHumidityChan)
	for {
		metric := <-temperatureHumidityChan
		if metric.Device == govee.DEVICE_SALON {
			salon_temperature.Set(metric.Temperature)
			salon_humidity.Set(metric.Humidity)
		} else if metric.Device == govee.DEVICE_TALLER {
			sotano_temperature.Set(metric.Temperature)
			sotano_humidity.Set(metric.Humidity)
		}
		log.Printf("Temperature: %.2f, Humidity: %.2f, Device: %v\n", metric.Temperature, metric.Humidity, metric.Device)
	}
}
