GOOS=linux GOARCH=arm64 go build .
scp home-metrics iot.internal:home-metrics-new
