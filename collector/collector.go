package collector

import "github.com/prometheus/client_golang/prometheus"

type TincCollector struct {
	network         string
	collectors      []prometheus.Collector
	receivePackets  *prometheus.GaugeVec
	receiveBytes    *prometheus.GaugeVec
	transmitPackets *prometheus.GaugeVec
	transmitBytes   *prometheus.GaugeVec
	udpPingRtt      *prometheus.GaugeVec
	reachability    *prometheus.GaugeVec
	pid             *prometheus.GaugeVec
}

func NewTincController(network string) *TincCollector {
	c := &TincCollector{network: network}
	c.receivePackets = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tinc",
		Subsystem: "network",
		Name:      "receive_packets_total",
		Help:      "Tinc network statistic receive_packets",
	}, []string{"network", "name"})
	c.collectors = append(c.collectors, c.receivePackets)

	c.receiveBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tinc",
		Subsystem: "network",
		Name:      "receive_bytes_total",
		Help:      "Tinc network statistic receive_bytes",
	}, []string{"network", "name"})
	c.collectors = append(c.collectors, c.receiveBytes)

	c.transmitPackets = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tinc",
		Subsystem: "network",
		Name:      "transmit_packets_total",
		Help:      "Tinc network statistic transmit_packets",
	}, []string{"network", "name"})
	c.collectors = append(c.collectors, c.transmitPackets)

	c.transmitBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tinc",
		Subsystem: "network",
		Name:      "transmit_bytes_total",
		Help:      "Tinc network statistic transmit_bytes",
	}, []string{"network", "name"})
	c.collectors = append(c.collectors, c.transmitBytes)

	c.udpPingRtt = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tinc",
		Subsystem: "network",
		Name:      "udp_ping_rtt",
		Help:      "Tinc network udp ping rtt",
	}, []string{"network", "name"})
	c.collectors = append(c.collectors, c.udpPingRtt)

	c.reachability = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tinc",
		Subsystem: "network",
		Name:      "reachability",
		Help:      "Tinc network reachability with other node",
	}, []string{"network", "name", "via"})
	c.collectors = append(c.collectors, c.reachability)

	c.pid = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tinc",
		Subsystem: "runtime",
		Name:      "pid",
		Help:      "Tinc daemon pid",
	}, []string{"network", "name", "port"})
	c.collectors = append(c.collectors, c.pid)

	return c
}

func (c *TincCollector) SetReceivePackets(nodeName string, value float64) {
	c.receivePackets.WithLabelValues(c.network, nodeName).Set(value)
}

func (c *TincCollector) SetReceiveBytes(nodeName string, value float64) {
	c.receiveBytes.WithLabelValues(c.network, nodeName).Set(value)
}

func (c *TincCollector) SetTransmitPackets(nodeName string, value float64) {
	c.transmitPackets.WithLabelValues(c.network, nodeName).Set(value)
}

func (c *TincCollector) SetTransmitBytes(nodeName string, value float64) {
	c.transmitBytes.WithLabelValues(c.network, nodeName).Set(value)
}

func (c *TincCollector) SetUdpPingRtt(nodeName string, value float64) {
	c.udpPingRtt.WithLabelValues(c.network, nodeName).Set(value)
}

func (c *TincCollector) SetReachability(nodeName, via string, value float64) {
	c.reachability.WithLabelValues(c.network, nodeName, via).Set(value)
}

func (c *TincCollector) SetPid(nodeName, port string, value float64) {
	c.pid.WithLabelValues(c.network, nodeName, port).Set(value)
}

func (c *TincCollector) Reset() {
	for _, collector := range c.collectors {
		if c, ok := collector.(*prometheus.GaugeVec); ok {
			c.Reset()
		}
	}
}
func (c *TincCollector) Collect(ch chan<- prometheus.Metric) {
	for _, collector := range c.collectors {
		collector.Collect(ch)
	}
}
func (c *TincCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, collector := range c.collectors {
		collector.Describe(ch)
	}
}
