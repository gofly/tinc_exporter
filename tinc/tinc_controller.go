package tinc

import (
	"fmt"
	"os"
	"path"
)

type TincPid struct {
	Pid    int
	Cookie string
	Host   string
	Port   string
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

func (c *TincController) QueryPid() (*TincPid, error) {
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
	_, err = fmt.Fscanf(f, "%d %s %s port %s", &p.Pid, &p.Cookie, &p.Host, &p.Port)
	return p, err
}
