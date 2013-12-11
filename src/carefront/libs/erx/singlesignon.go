package erx

import (
	"crypto/sha512"
	"encoding/base64"
	"io"
	"math/rand"
	"os"
	"strings"
	"time"
)

var (
	alphanum = []byte("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz")
)

type SingleSignOn struct {
	Code         string
	UserIdVerify string
}

func GenerateSingleSignOn() SingleSignOn {
	rand.Seed(time.Now().UnixNano())
	clinicKey := os.Getenv("DOSESPOT_CLINIC_KEY")
	userId := os.Getenv("DOSESPOT_USER_ID")

	singleSignOn := SingleSignOn{}

	// STEP 1: Create a random phrase 32 characters long in UTF8
	phrase := generateRandomAlphaNumString(32)
	singleSignOn.Code = createSingleSignOn(phrase, clinicKey)
	singleSignOn.UserIdVerify = createSingleSignOnUserIdVerify(phrase, clinicKey, userId)

	return singleSignOn
}

func generateRandomAlphaNumString(n int) string {
	randomBytes := make([]byte, n)
	for i := 0; i < n; i++ {
		randomBytes[i] = alphanum[rand.Intn(len(alphanum))]
	}
	return string(randomBytes)
}

// Steps to create the singleSignOnUserIdVerify is spelled out in the
func createSingleSignOnUserIdVerify(phrase, clinicKey, userId string) string {

	// STEP 1: first 22 characters from phrase
	first22Pharse := phrase[:22]

	// STEPS 2-5: Compute the hash of the userId + first 22 characters from phrase + key
	sha512Hash := sha512.New()
	io.WriteString(sha512Hash, userId)
	io.WriteString(sha512Hash, first22Pharse)
	io.WriteString(sha512Hash, clinicKey)
	hashedBytes := sha512Hash.Sum(nil)

	// STEP 6: Get a Base64String out of the hash that you created
	base64string := base64.StdEncoding.EncodeToString(hashedBytes)

	// STEP 7: if there are two = signs at the end, then remove them.
	base64string = removeTwoEqualSignsIfPresent(base64string)

	return base64string
}

func createSingleSignOn(phrase, clinicKey string) string {

	// STEPS 2 - 4: Compute the hash of the phrase concatenated with the key
	sha512Hash := sha512.New()
	io.WriteString(sha512Hash, phrase)
	io.WriteString(sha512Hash, clinicKey)
	hasedBytes := sha512Hash.Sum(nil)

	// STEP 5: Get a Base64String out of the hash that you created
	base64string := base64.StdEncoding.EncodeToString(hasedBytes)

	// STEP 6: If there are two = signs at the end, then remove them.
	base64string = removeTwoEqualSignsIfPresent(base64string)

	// STEP 7: Prepend the same random phrase from step 2 to your code.
	return phrase + base64string
}

func removeTwoEqualSignsIfPresent(str string) string {
	if strings.HasSuffix(str, "==") {
		return str[:len(str)-2]
	}
	return str
}
