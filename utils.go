package main

import (
	log "github.com/sirupsen/logrus"
	"strconv"
	"strings"
)

func ConvHandleStrInt(str string) int {
	atoi, err := strconv.Atoi(strings.ReplaceAll(str, " = ", ""))
	if err != nil {
		log.Error(err)
		return 0
	}
	return atoi
}
