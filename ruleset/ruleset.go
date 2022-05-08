package ruleset

import "net"

type Ruleset interface {
	Match(addr *net.TCPAddr) bool
}

type RuleFunc func(addr *net.TCPAddr) bool

type Global struct{}

func (g *Global) Match(addr *net.TCPAddr) bool { return true }

type Direct struct{}

func (d *Direct) Match(addr *net.TCPAddr) bool { return false }
