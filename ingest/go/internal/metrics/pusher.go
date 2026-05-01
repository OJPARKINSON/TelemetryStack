package metrics

import (
	"log"
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

func StartPusher(pushgatewayURL string) {
	instance := runtime.GOOS + "/" + runtime.GOARCH
	pusher := push.New(pushgatewayURL+":9091", "ingest-service").
		Grouping("instance", instance).
		Gatherer(prometheus.DefaultGatherer)

	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for range ticker.C {
			if err := pusher.Push(); err != nil {
				log.Printf("metrics push failed: %v", err)
			}
		}
	}()
}
