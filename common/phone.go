package common

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

// original list from dosespot (they claim that it was updated in January 2011).
// Added new area codes and those scheduled to be introduced through 2016 from https://en.wikipedia.org/wiki/List_of_North_American_Numbering_Plan_area_codes
var (
	validAreaCodes = []string{"201", "202", "203", "204", "205", "206", "207", "208", "209", "210", "212", "213", "214", "215", "216",
		"217", "218", "219", "224", "225", "226", "228", "229", "231", "234", "239", "240", "242", "246", "248", "250", "251", "252", "253",
		"254", "256", "260", "262", "264", "267", "268", "269", "270", "276", "281", "284", "289", "301", "302", "303", "304", "305", "306",
		"307", "308", "309", "310", "312", "313", "314", "315", "316", "317", "318", "319", "320", "321", "323", "325", "330", "331", "334",
		"336", "337", "339", "340", "343", "345", "347", "351", "352", "360", "361", "385", "386", "401", "402", "403", "404", "405", "406",
		"407", "408", "409", "410", "412", "413", "414", "415", "416", "417", "418", "419", "423", "424", "425", "430", "432", "434", "435",
		"438", "440", "441", "442", "443", "450", "456", "458", "469", "470", "473", "475", "478", "479", "480", "484", "500", "501", "502",
		"503", "504", "505", "506", "507", "508", "509", "510", "512", "513", "514", "515", "516", "517", "518", "519", "520", "530", "533",
		"534", "540", "541", "551", "559", "561", "562", "563", "567", "570", "571", "573", "574", "575", "579", "580", "581", "585", "586",
		"587", "600", "601", "602", "603", "604", "605", "606", "607", "608", "609", "610", "612", "613", "614", "615", "616", "617", "618",
		"619", "620", "623", "626", "630", "631", "636", "641", "646", "647", "649", "650", "651", "657", "660", "661", "662", "664", "670",
		"671", "678", "681", "682", "684", "700", "701", "702", "703", "704", "705", "706", "707", "708", "709", "710", "712", "713", "714",
		"715", "716", "717", "718", "719", "720", "724", "727", "731", "732", "734", "740", "747", "754", "757", "758", "760", "762", "763",
		"765", "767", "769", "770", "772", "773", "774", "775", "778", "779", "780", "781", "784", "785", "786", "787", "800", "801", "802",
		"803", "804", "805", "806", "807", "808", "809", "810", "812", "813", "814", "815", "816", "817", "818", "819", "828", "829", "830",
		"831", "832", "843", "845", "847", "848", "849", "850", "855", "856", "857", "858", "859", "860", "862", "863", "864", "865", "866",
		"867", "868", "869", "870", "872", "876", "877", "878", "888", "900", "901", "902", "903", "904", "905", "906", "907", "908", "909",
		"910", "912", "913", "914", "915", "916", "917", "918", "919", "920", "925", "928", "931", "936", "937", "938", "939", "940", "941",
		"947", "949", "951", "952", "954", "956", "970", "971", "972", "973", "978", "979", "980", "985", "989", "249", "531", "539", "721",
		"929", "431", "566", "667", "669", "873", "984", "236", "272", "365", "437", "639", "737", "844", "346", "364", "577", "725", "782",
		"930", "959", "220", "548", "628", "629", "854", "825"}
)

const MaxPhoneNumberLength = 25

type Phone string

func (p Phone) String() string {
	return string(p)
}

func ParsePhone(phoneNumber string) (Phone, error) {
	p := Phone(phoneNumber)
	if err := p.Validate(); err != nil {
		return Phone(""), err
	}
	return p, nil
}

func (p *Phone) UnmarshalJSON(data []byte) error {
	strP := string(data)

	if len(strP) == 0 {
		return nil
	}

	if strP[0] == '"' && len(strP) > 2 {
		*p = Phone(strP[1 : len(strP)-1])
	} else {
		*p = Phone(strP)
	}
	if err := p.Validate(); err != nil {
		return err
	}

	return nil
}

func (p Phone) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, p)), nil
}

func (p *Phone) Scan(src interface{}) error {
	var nullString sql.NullString
	err := nullString.Scan(src)
	if err != nil {
		return err
	}

	*p = Phone(nullString.String)
	return nil
}

func (p *Phone) Validate() error {
	phoneNumber := p.String()
	// phone number has to be atleast 10 digits long
	if len(phoneNumber) < 10 {
		return fmt.Errorf("Invalid phone number")
	}

	if len(phoneNumber) > MaxPhoneNumberLength {
		return fmt.Errorf("Phone numbers cannot be longer than %d digits", MaxPhoneNumberLength)
	}

	// ensure that there are no repeating digits in the number
	if isRepeatingDigits(phoneNumber) {
		return fmt.Errorf("Phone number cannot have repeating digits: %s", phoneNumber)
	}

	// attempt to break the string based on "-" to identify if phone number is formatted
	components := strings.Split(phoneNumber, "-")

	if len(components) == 1 {

		// remove the leading 1 if present
		if len(phoneNumber) == 11 && phoneNumber[0] == '1' {
			phoneNumber = phoneNumber[1:]
		}

		// if there is no "-" in the number, then the only possible format that we accept is all digits for phone number
		// if first 10 characteres are not digits, phone number is not valid
		_, err := strconv.Atoi(phoneNumber[:10])
		if err != nil {
			return fmt.Errorf("Invalid phone number")
		}

		if !isValidAreaCode(phoneNumber[:3]) {
			return fmt.Errorf("Invalid area code")
		}

		if len(phoneNumber) > 10 {
			// only acceptable character for extension is x
			if phoneNumber[10] != 'x' && phoneNumber[10] != 'X' {
				return fmt.Errorf("Invalid extension for phone number. Extension must to start with an 'x'")
			}

			if len(phoneNumber) == 11 {
				return fmt.Errorf("Invalid extension for phone number. 'x' must follow the extension")
			}

			_, err := strconv.Atoi(phoneNumber[11:])
			if err != nil {
				return fmt.Errorf("Invalid extension for phone number. Extension can only be digits")
			}
		}
	} else {
		if len(components) != 3 {
			return fmt.Errorf("Invalid phone number")
		}

		// check if the first component has a leading 1, and remove if so
		if len(components[0]) == 4 && components[0][0] == '1' {
			components[0] = components[0][1:]
		}

		// area code should have 3 digits in it
		if !isValidAreaCode(components[0]) {
			return fmt.Errorf("Invalid area code")
		}

		// second component should also have 3 digits in it
		if len(components[1]) != 3 {
			return fmt.Errorf("Invalid area code")
		}
		_, err := strconv.Atoi(components[1])
		if err != nil {
			return fmt.Errorf("Invalid phone number")
		}

		// third component should definitely have 4 digits but can have more if there is an extension involved
		if len(components[2]) < 4 {
			return fmt.Errorf("Invalid phone number")
		}

		// first 4 can only be digits in the last component
		_, err = strconv.Atoi(components[2][:4])
		if err != nil {
			return fmt.Errorf("Invalid phone number")
		}

		if len(components[2]) > 4 {
			if components[2][4] != 'x' && components[2][4] != 'X' {
				return fmt.Errorf("Invalid extension for phone number. Extension must to start with an 'x'")
			}

			if len(components[2]) == 5 {
				return fmt.Errorf("Invalid extension for phone number. 'x' must follow the extension")
			}

			_, err := strconv.Atoi(components[2][5:])
			if err != nil {
				return fmt.Errorf("Invalid extension for phone number. Extension can only be digits")
			}
		}
	}

	return nil
}

func isRepeatingDigits(phoneNumber string) bool {
	firstRune, _ := utf8.DecodeRuneInString(phoneNumber)
	for _, r := range phoneNumber {
		if firstRune != r && r != '-' {
			return false
		}
	}
	return true
}

func isValidAreaCode(areaCode string) bool {
	for _, validAreaCode := range validAreaCodes {
		if validAreaCode == areaCode {
			return true
		}
	}
	return false
}
