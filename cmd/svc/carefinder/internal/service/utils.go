package service

// cleanupZipcode returns the first 5 digits of the zipcode
func cleanupZipcode(zipcode string) string {
	if len(zipcode) > 5 {
		return zipcode[:5]
	}

	return zipcode
}
