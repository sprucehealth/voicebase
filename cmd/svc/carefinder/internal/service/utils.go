package service

import (
	"math/rand"
	"net/http"
	"strings"

	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/models"
)

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

func isMobile(r *http.Request) bool {
	return strings.Contains(r.UserAgent(), "iPhone") || strings.Contains(r.UserAgent(), "iPod") || strings.Contains(strings.ToLower(r.UserAgent()), "android")
}

type byReviewCount []*models.Doctor

func (c byReviewCount) Len() int      { return len(c) }
func (c byReviewCount) Swap(i, j int) { c[i], c[j] = c[j], c[i] }
func (c byReviewCount) Less(i, j int) bool {
	return c[i].ReviewCount < c[j].ReviewCount
}
