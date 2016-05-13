package auth

// ObfuscateAccount obfuscates customer identifiable information from an account
func ObfuscateAccount(a *Account) *Account {
	if len(a.FirstName) > 1 {
		a.FirstName = a.FirstName[:1]
	}
	if len(a.LastName) > 1 {
		a.LastName = a.LastName[:1]
	}
	return a
}
