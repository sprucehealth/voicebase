package dronboard

import (
	"errors"
	"regexp"
	"strconv"
)

var ErrInvalidAmount = errors.New("invalid amount")

var re = regexp.MustCompile(`^[\s\$]*(\d+)?(\.\d*)?\s*$`)

func parseAmount(s string) (int, error) {
	m := re.FindStringSubmatch(s)
	if len(m) != 3 {
		return 0, ErrInvalidAmount
	}

	v1, v2 := -1, -1

	var err error
	if len(m[1]) != 0 {
		v1, err = strconv.Atoi(m[1])
		if err != nil {
			return 0, err
		}
	}
	if len(m[2]) > 1 {
		v2, err = strconv.Atoi(m[2][1:])
		if err != nil {
			return 0, err
		}
	}

	if m[2] == "" { // XX
		if v1 < 0 {
			return 0, ErrInvalidAmount
		}
		return v1, nil
	}
	if v1 < 0 {
		if v2 < 0 {
			return 0, ErrInvalidAmount
		}
		return v2, nil
	}
	cents := v1 * 100
	if v2 > 0 {
		cents += v2
	}
	return cents, nil
}
