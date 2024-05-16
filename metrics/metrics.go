package metrics

type TemperatureHumidity struct {
	Temperature float64
	Humidity    float64
	Device      string
}

type Power struct {
	Value  float64
	Device string
}
