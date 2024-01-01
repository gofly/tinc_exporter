package tinc

import (
	"fmt"
)

type Reachability int

func (r Reachability) String() string {
	switch r {
	case CAN_REACH_ITSELF:
		return "can reach itself"
	case UNREACHABLE:
		return "unreachable"
	case INDIRECT_VIA_OTHER_NODE:
		return "indirectly via other node"
	case UNKNOWN:
		return "unknown"
	case DIRECTLY_WITH_UDP:
		return "directly with UDP"
	case DIRECTLY_WITH_TCP:
		return "directly with TCP"
	case FORWARDED_VIA_OTHER_NODE:
		return "forwarded via other node"
	}
	return ""
}

const (
	CAN_REACH_ITSELF Reachability = iota + 1
	DIRECTLY_WITH_UDP
	DIRECTLY_WITH_TCP
	INDIRECT_VIA_OTHER_NODE
	FORWARDED_VIA_OTHER_NODE
	UNREACHABLE
	UNKNOWN
)

type TincNodeStatus uint32

type TincNode struct {
	NodeName        string
	NodeID          string
	Host            string
	Port            string
	Cipher          int
	Compression     int
	Options         uint32
	Status          TincNodeStatus
	Nexthop         string
	Via             string
	Distance        int
	Pmtu            int
	MinMtu          int
	MaxMtu          int
	LastStateChange int64
	UDPPingRTT      float64
	InPackets       int64
	InBytes         int64
	OutPackets      int64
	OutBytes        int64
}

func (s TincNodeStatus) Reachable() bool {
	return (s&0x10 > 0)
}

func (s TincNodeStatus) Indirect() bool {
	return (s&0x20 > 0)
}

func (s TincNodeStatus) ValidKey() bool {
	return (s&0x02 > 0)
}

func (n *TincNode) Reachability() Reachability {
	switch {
	case n.MySelf():
		return CAN_REACH_ITSELF
	case !n.Status.Reachable():
		return UNREACHABLE
	case n.Via != n.NodeName:
		return INDIRECT_VIA_OTHER_NODE
	case !n.Status.ValidKey():
		return UNKNOWN
	case n.MinMtu > 0:
		return DIRECTLY_WITH_UDP
	case n.Nexthop == n.NodeName:
		return DIRECTLY_WITH_TCP
	}
	return FORWARDED_VIA_OTHER_NODE
}

func (n *TincNode) MySelf() bool {
	return (n.Host == "MYSELF")
}

func (n *TincNode) ViaNode() string {
	switch n.Reachability() {
	case INDIRECT_VIA_OTHER_NODE:
		return n.Via
	case FORWARDED_VIA_OTHER_NODE:
		return n.Nexthop
	}
	return ""
}

func (n *TincNode) PMTU() int {
	if n.MinMtu > 0 {
		return n.Pmtu
	}
	return 0
}

func (n *TincNode) RTT() float64 {
	if n.MinMtu > 0 {
		return n.UDPPingRTT / 1000
	}
	return 0
}

func (c *TincController) QueryNodes() ([]TincNode, error) {
	nodes := make([]TincNode, 0)
	err := c.DoRequest(CONTROL, REQ_DUMP_NODES, func(line []byte) error {
		n := TincNode{}
		var ignoreInt int
		_, err := fmt.Sscanf(string(line), "%s %s %s port %s %d %d %d %d %x %x %s %s %d %d %d %d %d %f %d %d %d %d",
			&n.NodeName, &n.NodeID, &n.Host, &n.Port, &n.Cipher, &ignoreInt, &ignoreInt, &n.Compression,
			&n.Options, &n.Status, &n.Nexthop, &n.Via, &n.Distance, &n.Pmtu, &n.MinMtu, &n.MaxMtu, &n.LastStateChange,
			&n.UDPPingRTT, &n.InPackets, &n.InBytes, &n.OutPackets, &n.OutBytes)
		if err != nil {
			return err
		}
		nodes = append(nodes, n)
		return nil
	})
	return nodes, err
}
