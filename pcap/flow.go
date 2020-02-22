package pcap

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

type Socket struct {
	net.IP
	Port int
}

func (s Socket) String() string {
	return fmt.Sprintf("%s:%d", s.IP, s.Port)
}

type Flow struct {
	S0 Socket
	S1 Socket
}

func (f Flow) String() string {
	return f.S0.String() + "," + f.S1.String()
}

func ParseSocket(s string) (Socket, error) {
	pair := strings.Split(s, ":")
	if len(pair) == 2 {
		ip := net.ParseIP(pair[0])
		if ip != nil {
			port, err := strconv.Atoi(pair[1])
			if err == nil {
				return Socket{ip, port}, nil
			}
		}
	}
	return Socket{}, fmt.Errorf("address spec must have form ip:port (%s)", s)
}

func ParseFlow(h0, h1 string) (Flow, error) {
	s0, err := ParseSocket(h0)
	if err != nil {
		return Flow{}, err
	}
	s1, err := ParseSocket(h1)
	if err != nil {
		return Flow{}, err
	}
	return Flow{s0, s1}, nil
}
