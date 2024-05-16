package tapo

import (
	"encoding/json"
	"log"
	"time"

	goklap "github.com/mascanio/go-klap"
	"github.com/mascanio/home-metrics/metrics"
)

type response struct {
	Result struct {
		Current_power int
	}
	Error_code int
}

type Config struct {
	User, Password, Ip string
}

type Tapo struct {
	config Config
	klap   goklap.Klap
}

func New(config Config) Tapo {
	return Tapo{klap: goklap.New(config.Ip, "80", config.User, config.Password), config: config}
}

func (t *Tapo) ScanMetrics(metricChan chan<- metrics.Power) {
	for {
		time.Sleep(time.Second * 15)
		log.Println("Reading power")
		data := "{\"method\": \"get_energy_usage\", \"params\": null}"
		r, err := t.klap.Request("request", data, nil)
		if err != nil {
			log.Println(err)
			continue
		}
		var responseParsed response
		err = json.Unmarshal(r, &responseParsed)
		if err != nil {
			log.Println(err)
		}
		powerValue := float64(responseParsed.Result.Current_power) / 1000.0
		metricChan <- metrics.Power{Device: t.config.Ip, Value: powerValue}
	}
}
