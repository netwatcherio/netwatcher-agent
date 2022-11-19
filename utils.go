package main

import "strconv"

func convHandleStrInt(str string) int {
	atoi, err := strconv.Atoi(str)
	if err != nil {
		return 0
	}
	return atoi
}
