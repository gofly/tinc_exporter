package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const TINC_CTL_VERSION_CURRENT = 0

type RequestType int

const (
	ID RequestType = iota
	METAKEY
	CHALLENGE
	CHAL_REPLY
	ACK
	STATUS
	ERROR
	TERMREQ
	PING
	PONG
	ADD_SUBNET
	DEL_SUBNET
	ADD_EDGE
	DEL_EDGE
	KEY_CHANGED
	REQ_KEY
	ANS_KEY
	PACKET
	/* Tinc 1.1 requests */
	CONTROL
	REQ_PUBKEY
	ANS_PUBKEY
	SPTPS_PACKET
	UDP_INFO
	MTU_INFO
	LAST
)

type Request int

const (
	REQ_STOP Request = iota
	REQ_RELOAD
	REQ_RESTART
	REQ_DUMP_NODES
	REQ_DUMP_EDGES
	REQ_DUMP_SUBNETS
	REQ_DUMP_CONNECTIONS
	REQ_DUMP_GRAPH
	REQ_PURGE
	REQ_SET_DEBUG
	REQ_RETRY
	REQ_CONNECT
	REQ_DISCONNECT
	REQ_DUMP_TRAFFIC
	REQ_PCAP
	REQ_LOG
)

type TincTraffic struct {
	NodeName   string
	InPackets  int64
	InBytes    int64
	OutPackets int64
	OutBytes   int64
}

type TincPid struct {
	Pid    int
	Cookie string
	Host   string
	Port   int
}

type TincController struct {
	network string
	runPath string
}

func NewTincController(runPath, network string) *TincController {
	return &TincController{
		runPath: runPath,
		network: network,
	}
}

func (c *TincController) LoadPidFile() (*TincPid, error) {
	pidFile := "tinc.pid"
	if c.network != "" {
		pidFile = fmt.Sprintf("tinc.%s.pid", c.network)
	}
	f, err := os.Open(path.Join(c.runPath, pidFile))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	p := &TincPid{}
	_, err = fmt.Fscanf(f, "%d %s %s port %d", &p.Pid, &p.Cookie, &p.Host, &p.Port)
	return p, err
}

func (c *TincController) Do(reqType RequestType, req Request, fn func([]byte) error) error {
	p, err := c.LoadPidFile()
	if err != nil {
		return err
	}
	socketFile := "tinc.pid"
	if c.network != "" {
		socketFile = fmt.Sprintf("tinc.%s.socket", c.network)
	}
	conn, err := net.Dial("unix", path.Join(c.runPath, socketFile))
	if err != nil {
		return err
	}
	defer conn.Close()

	command := fmt.Sprintf("%d %d", reqType, req)
	_, err = fmt.Fprintf(conn, "%d ^%s %d\n%s\n", ID, p.Cookie, TINC_CTL_VERSION_CURRENT, command)
	if err != nil {
		return err
	}
	r := bufio.NewReader(conn)
	for {
		data, err := r.ReadBytes('\n')
		if err != nil {
			return err
		}
		line := bytes.TrimSpace(data)
		if bytes.Equal(line, []byte(command)) {
			break
		}
		if !bytes.HasPrefix(line, []byte(command)) {
			continue
		}
		err = fn(bytes.TrimPrefix(line, []byte(command+" ")))
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}
	return nil
}

func (c *TincController) DumpTraffic() ([]TincTraffic, error) {
	traffic := make([]TincTraffic, 0)
	err := c.Do(CONTROL, REQ_DUMP_TRAFFIC, func(line []byte) error {
		t := TincTraffic{}
		n, err := fmt.Sscanf(string(line), "%s %d %d %d %d", &t.NodeName, &t.InPackets, &t.InBytes, &t.OutPackets, &t.OutBytes)
		if err != nil {
			return err
		}
		if n == 2 {
			return io.EOF
		}
		traffic = append(traffic, t)
		return nil
	})
	return traffic, err
}

func main() {
	var runPath, network string
	flag.StringVar(&runPath, "d", "/var/run", "the run dir which contains pid and socket files")
	flag.StringVar(&network, "n", "", "network name")
	flag.Parse()

	receivePackets := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tinc",
		Subsystem: "network",
		Name:      "receive_packets_total",
		Help:      "Tinc network statistic receive_packets",
	}, []string{"network", "name"})
	receiveBytes := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tinc",
		Subsystem: "network",
		Name:      "receive_bytes_total",
		Help:      "Tinc network statistic receive_bytes",
	}, []string{"network", "name"})
	transmitPackets := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tinc",
		Subsystem: "network",
		Name:      "transmit_packets_total",
		Help:      "Tinc network statistic transmit_packets",
	}, []string{"network", "name"})
	transmitBytes := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tinc",
		Subsystem: "network",
		Name:      "transmit_bytes_total",
		Help:      "Tinc network statistic transmit_bytes",
	}, []string{"network", "name"})
	pid := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tinc",
		Subsystem: "runtime",
		Name:      "pid",
		Help:      "Tinc daemon pid",
	}, []string{"network", "port"})
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
	prometheus.MustRegister(receivePackets, receiveBytes,
		transmitPackets, transmitBytes, pid, temperature, humidity)

	c := NewTincController(runPath, network)
	handler := promhttp.Handler()
	var n int64
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		receivePackets.Reset()
		receiveBytes.Reset()
		transmitPackets.Reset()
		transmitBytes.Reset()
		pidVal := -500
		p, err := c.LoadPidFile()
		if err == nil {
			pidVal = p.Pid
		}
		pid.WithLabelValues(network, strconv.Itoa(p.Port)).Set(float64(pidVal))
		traffic, err := c.DumpTraffic()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else {
			for _, t := range traffic {
				receivePackets.WithLabelValues(network, t.NodeName).Set(float64(t.InPackets))
				receiveBytes.WithLabelValues(network, t.NodeName).Set(float64(t.InBytes))
				transmitPackets.WithLabelValues(network, t.NodeName).Set(float64(t.OutPackets))
				transmitBytes.WithLabelValues(network, t.NodeName).Set(float64(t.OutBytes))
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
	log.Fatal(http.ListenAndServe(":9101", nil))
}
