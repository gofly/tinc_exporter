package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/gofly/tinc_exporter/collector"
	"github.com/gofly/tinc_exporter/tinc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	var runPath, network, addr string
	flag.StringVar(&addr, "addr", ":9101", "the address to serve")
	flag.StringVar(&runPath, "d", "/var/run", "the run dir which contains pid and socket files")
	flag.StringVar(&network, "n", "", "network name")
	flag.Parse()

	collector := collector.NewTincController(network)
	temperature := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "home",
		Subsystem: "environment",
		Name:      "temperature",
		Help:      "Temperature in home",
	})
	humidity := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "home",
		Subsystem: "environment",
		Name:      "humidity",
		Help:      "Humidity in home",
	})
	prometheus.MustRegister(collector, temperature, humidity)

	controller := tinc.NewTincController(runPath, network)
	handler := promhttp.Handler()
	var n int64
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		collector.Reset()

		nodes, err := controller.QueryNodes()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else {
			for _, node := range nodes {
				if node.MySelf() {
					pidVal := -1
					p, err := controller.QueryPid()
					if err == nil {
						pidVal = p.Pid
					}
					collector.SetPid(node.NodeName, p.Port, float64(pidVal))
					continue
				}
				collector.SetReceiveBytes(node.NodeName, float64(node.InBytes))
				collector.SetReceivePackets(node.NodeName, float64(node.InPackets))
				collector.SetTransmitBytes(node.NodeName, float64(node.OutBytes))
				collector.SetTransmitPackets(node.NodeName, float64(node.OutPackets))
				collector.SetReachability(node.NodeName, node.ViaNode(), float64(node.Reachability()))
				rtt := node.RTT()
				if rtt > 0 {
					collector.SetUdpPingRtt(node.NodeName, rtt)
				}
			}
		}
		if atomic.AddInt64(&n, 1)%4 == 1 {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
			defer cancel()
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://192.168.3.204/dht", nil)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
			} else {
				result := &struct {
					T, H float64
				}{}
				err = json.NewDecoder(resp.Body).Decode(result)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
				} else {
					temperature.Set(result.T)
					humidity.Set(result.H)
				}
			}
			resp.Body.Close()
		}
		handler.ServeHTTP(w, r)
	})
	log.Fatal(http.ListenAndServe(addr, nil))
}
