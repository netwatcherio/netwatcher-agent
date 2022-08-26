package main

import "time"

var (
	timeout        = 800 * time.Millisecond
	interval       = 100 * time.Millisecond
	hopSleep       = time.Nanosecond
	maxHops        = 64
	maxUnknownHops = 10
	ringBufferSize = 50
	ptrLookup      = false
	srcAddr        = ""
	ttl            = 60
)
