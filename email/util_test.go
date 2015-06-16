package email

import "testing"

var validEmails = []string{
	"a@example.com",
	"postmaster@example.com",
	"president@kremlin.gov.ru",
	"example@example.co.uk",
}

var invalidEmails = []string{
	"",
	"example",
	"example.com",
	".com",
	"адрес@пример.рф",
	" space_before@example.com",
	"space between@example.com",
	"\nnewlinebefore@example.com",
	"newline\nbetween@example.com",
	"test@example.com.",
	"asyouallcanseethisemailaddressexceedsthemaximumnumberofcharactersallowedtobeintheemailaddresswhichisnomorethatn254accordingtovariousrfcokaycanistopnowornotyetnoineedmorecharacterstoadd@i.really.cannot.thinkof.what.else.to.put.into.this.invalid.address.net",
	"someone@somewhere",
	"someone@gmail.con", // invalid TLD
}

func TestIsValidEmail(t *testing.T) {
	for i, v := range validEmails {
		if !IsValidEmail(v) {
			t.Errorf("%d: didn't accept valid email: %s", i, v)
		}
	}
	for i, v := range invalidEmails {
		if IsValidEmail(v) {
			t.Errorf("%d: accepted invalid email: %s", i, v)
		}
	}
}
