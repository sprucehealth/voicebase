package randutils

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"
)

const maxIntLength int64 = 18

var randRuneSet = []rune("abcdefghijklmnopqrstuvwxyz1234567890ABCDEFGHIJKLMNOPQRSTUVWXYZ`~!@#$%^&*()-_=+[{}];:<,'>.?/\\|")

// Int64LengthN returns a random number N digits in length
func Int64LengthN(n int64) int64 {
	if n > maxIntLength {
		n = maxIntLength
	}

	var base int64 = 1
	for i := base; i < n; i++ {
		base = base * 10
	}
	return base + rand.Int63n((base*10)-1)
}

// StringN generates a random string of length n
func StringN(n int) string {
	r := make([]rune, n)
	for i := range r {
		r[i] = randRuneSet[rand.Intn(len(randRuneSet))]
	}
	return string(r)
}

// String returns a random string of a random non empty length up to maxLength
func String(maxLength int) string {
	r := rand.Intn(maxLength)
	if r == 0 {
		r = 1
	}
	return StringN(r)
}

// PhoneNumber generates a random 10 digit number
func PhoneNumber() string {
	return "+" + strconv.FormatInt(rand.Int63n(9000000000)+1000000000, 10)
}

// UniqueEmail generates a random unique valid email address utilizing the current timestamp
func UniqueEmail() string {
	return fmt.Sprintf("e" + strconv.FormatInt(timestampPrefixedInt64(), 10) + "@example.com")
}

// Bool returns a random boolean value
func Bool() bool {
	return (rand.Int() % 2) == 0
}

func timestampPrefixedInt64() int64 {
	r := time.Now().Unix()
	// Utilize an increasing timestamp suffixed by a random 7 digit number
	r = (r * 10000000) + (rand.Int63n(9000000) + 1000000)
	return r
}
