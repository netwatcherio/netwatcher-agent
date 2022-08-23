package main

import (
	"github.com/tonobo/mtr/pkg/icmp"
	"math"
	"math/rand"
	"net"
)

type icmpTarget struct {
	Address string `json:"address"`
	Result  struct {
		ElapsedMilliseconds int64 `json:"elapsed_ms"`
	} `json:"result"`
}

func CheckICMP(t *icmpTarget) {
	ipAddr := net.IPAddr{IP: net.ParseIP(t.Address)}

	seq := rand.Intn(math.MaxUint16)
	id := rand.Intn(math.MaxUint16) & 0xffff
	hop, _ := icmp.SendICMP(srcAddr, &ipAddr, t.Address, ttl, id, timeout, seq)
	t.Result.ElapsedMilliseconds = hop.Elapsed.Milliseconds()
}
