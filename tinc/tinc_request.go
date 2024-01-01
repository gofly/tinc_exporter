package tinc

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"path"
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

func (c *TincController) DoRequest(reqType RequestType, req Request, fn func([]byte) error) error {
	p, err := c.QueryPid()
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
