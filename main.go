package main

import (
	"log"
	"net/http"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/mascanio/home-metrics/metrics"
	govee "github.com/mascanio/home-metrics/providers/govee"
	"github.com/mascanio/home-metrics/providers/tapo"
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
	sotano_power = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "sotano_power",
		Help: "Power consumption",
	})
)

type config struct {
	PrometheusPort string `yaml:"prometheus-port"`
	Providers      struct {
		Govee govee.Config
		Tapo  tapo.Config
	}
}

func main() {
	var config config
	configFile, err := os.ReadFile("config.yaml")
	if err != nil {
		panic(err)
	}
	err = yaml.Unmarshal(configFile, &config)
	if err != nil {
		panic(err)
	}

	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":"+config.PrometheusPort, nil)

	goveeProvider, err := govee.New(config.Providers.Govee)
	if err != nil {
		panic(err)
	}
	temperatureHumidityChan := make(chan metrics.TemperatureHumidity)
	defer close(temperatureHumidityChan)
	go goveeProvider.ScanMetrics(temperatureHumidityChan)

	tapoProvider := tapo.New(config.Providers.Tapo)
	powerChan := make(chan metrics.Power)
	defer close(powerChan)
	go tapoProvider.ScanMetrics(powerChan)

	for {
		log.Println("Waiting for metrics...")
		select {
		case metric := <-temperatureHumidityChan:
			if metric.Device == govee.DEVICE_SALON {
				salon_temperature.Set(metric.Temperature)
				salon_humidity.Set(metric.Humidity)
			} else if metric.Device == govee.DEVICE_TALLER {
				sotano_temperature.Set(metric.Temperature)
				sotano_humidity.Set(metric.Humidity)
			}
			log.Printf("Temperature: %.2f, Humidity: %.2f, Device: %v\n", metric.Temperature, metric.Humidity, metric.Device)
		case power := <-powerChan:
			sotano_power.Set(power.Value)
			log.Printf("Power from %v %v\n", power.Device, power.Value)
		}
	}
}
