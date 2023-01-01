package checks

import "strconv"

func ConvHandleStrInt(str string) int {
	atoi, err := strconv.Atoi(str)
	if err != nil {
		return 0
	}
	return atoi
}
