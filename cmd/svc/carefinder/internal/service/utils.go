package service

import "math/rand"

// cleanupZipcode returns the first 5 digits of the zipcode
func cleanupZipcode(zipcode string) string {
	if len(zipcode) > 5 {
		return zipcode[:5]
	}

	return zipcode
}

func shuffle(ids []string) {
	for i := len(ids) - 1; i > 0; i-- {
		j := rand.Intn(i)
		ids[i], ids[j] = ids[j], ids[i]
	}
}
