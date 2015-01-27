package handlers

import "unicode"

// resemblesCode returns true if query has the characteristics
// of a diagnosis code.
// The rules for the check are as follows:
// 1. Query has to be atleast 2 characters long
// 2. First character has to be a letter
// 3. Second and third characters have to be digits
// 4. Fourth character, if present, has to be a period
// 5. Fifth, sixth and seventh characters, if present, can either be digits or placeholders (identified by 'X')
// 6. Eighth character, if present, has to be a letter or a digit
func resemblesCode(query string) bool {

	if len(query) < 2 {
		return false
	} else if !unicode.IsLetter(rune(query[0])) {
		return false
	} else if !unicode.IsDigit(rune(query[1])) {
		return false
	} else if len(query) > 2 && !unicode.IsDigit(rune(query[2])) {
		return false
	} else if len(query) > 3 && query[3] != '.' {
		return false
	} else if len(query) > 4 && !isValidSubcategoryCharacter(query[4]) {
		return false
	} else if len(query) > 5 && !isValidSubcategoryCharacter(query[5]) {
		return false
	} else if len(query) > 6 && !isValidSubcategoryCharacter(query[6]) {
		return false
	} else if len(query) > 7 && !(unicode.IsLetter(rune(query[7])) || unicode.IsDigit(rune(query[7]))) {
		return false
	} else if len(query) > 8 {
		return false
	}

	return true
}

func isValidSubcategoryCharacter(u uint8) bool {
	return u == 'X' || u == 'x' || unicode.IsDigit(rune(u))
}
