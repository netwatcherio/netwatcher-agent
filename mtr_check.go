package main

import (
	pj "github.com/hokaccha/go-prettyjson"
	"github.com/tonobo/mtr/pkg/mtr"
	"log"
)

type mtrTarget struct {
	Address string `json:"address"`
	Result  string `json:"result"`
}

func CheckMTR(t *mtrTarget) {
	m, ch, err := mtr.NewMTR(t.Address, srcAddr, timeout, interval, hopSleep,
		maxHops, maxUnknownHops, ringBufferSize, ptrLookup)
	if err != nil {
		log.Fatal(err)
	}

	go func(ch chan struct{}) {
		for {
			<-ch
		}
	}(ch)
	m.Run(ch, count)
	s, err := pj.Marshal(m)
	if err != nil {
		log.Fatal(err)
	}
	//fmt.Println(string(s))
	t.Result = string(s)
}
