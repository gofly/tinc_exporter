package tinc

import "fmt"

type TincTraffic struct {
	NodeName   string
	InPackets  int64
	InBytes    int64
	OutPackets int64
	OutBytes   int64
}

func (c *TincController) QueryTraffic() ([]TincTraffic, error) {
	traffic := make([]TincTraffic, 0)
	err := c.DoRequest(CONTROL, REQ_DUMP_TRAFFIC, func(line []byte) error {
		t := TincTraffic{}
		_, err := fmt.Sscanf(string(line), "%s %d %d %d %d", &t.NodeName, &t.InPackets, &t.InBytes, &t.OutPackets, &t.OutBytes)
		if err != nil {
			return err
		}
		traffic = append(traffic, t)
		return nil
	})
	return traffic, err
}
