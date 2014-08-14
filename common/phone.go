package common

import (
	"database/sql"
	"fmt"
	"strconv"
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

	// lets make sure the phone number is atleast 10 digits long
	phoneNumberLength := len(phoneNumber)
	if phoneNumberLength < 10 {
		return fmt.Errorf("Phone number has to be atleast 10 digits long")
	}
	// lets make sure the phone number is no longer than the maximum length specified
	// this is a limit set forth by surescripts
	if phoneNumberLength > MaxPhoneNumberLength {
		return fmt.Errorf("Phone number cannot be longer than %d characters in length", MaxPhoneNumberLength)
	}

	var currentIndex int
	var separator rune
	normalizedPhoneNumber := make([]byte, 0, len(phoneNumber)+2)
	// get rid of any leading 1 (can do this because no US area code starts with a 1)
	if phoneNumber[0] == '1' {
		currentIndex++
		// remove any separator after the leading 1
		if phoneNumber[currentIndex] == '-' || phoneNumber[currentIndex] == ' ' || phoneNumber[currentIndex] == '.' {
			currentIndex++
		}
	}

	// take the first chunk of 3; this should be a valid area code
	if !isValidAreaCode(phoneNumber[currentIndex : currentIndex+3]) {
		return fmt.Errorf("Invalid area code in phone number")
	}
	normalizedPhoneNumber = append(normalizedPhoneNumber, phoneNumber[currentIndex:currentIndex+3]...)
	normalizedPhoneNumber = append(normalizedPhoneNumber, '-')
	currentIndex += 3

	// check for any valid separator
	if phoneNumber[currentIndex] == ' ' || phoneNumber[currentIndex] == '-' || phoneNumber[currentIndex] == '.' {
		separator = rune(phoneNumber[currentIndex])
		currentIndex++
	}

	// next chunk of 3 should only contain digits
	if _, err := strconv.Atoi(phoneNumber[currentIndex : currentIndex+3]); err != nil {
		return fmt.Errorf("Invalid phone number")
	}
	normalizedPhoneNumber = append(normalizedPhoneNumber, phoneNumber[currentIndex:currentIndex+3]...)
	normalizedPhoneNumber = append(normalizedPhoneNumber, '-')
	currentIndex += 3

	// check for any valid separator
	if rune(phoneNumber[currentIndex]) == separator {
		currentIndex++
	}

	// next chunk of 4 should contain only digits
	if currentIndex+4 > len(phoneNumber) {
		return fmt.Errorf("Invalid phone number")
	} else if _, err := strconv.Atoi(phoneNumber[currentIndex : currentIndex+4]); err != nil {
		return fmt.Errorf("Invalid phone number")
	}
	normalizedPhoneNumber = append(normalizedPhoneNumber, phoneNumber[currentIndex:currentIndex+4]...)
	currentIndex += 4

	// if there is still more to the phone number then we are dealing with an extension
	if currentIndex < len(phoneNumber) {
		if currentIndex+2 > len(phoneNumber) {
			return fmt.Errorf("Invalid phone number")
		} else if phoneNumber[currentIndex] != 'x' && phoneNumber[currentIndex] != 'X' {
			return fmt.Errorf("Invalid phone number")
		} else if _, err := strconv.Atoi(phoneNumber[currentIndex+1:]); err != nil {
			return fmt.Errorf("Invalid phone number")
		}
		normalizedPhoneNumber = append(normalizedPhoneNumber, phoneNumber[currentIndex:]...)
	}

	phoneStr := string(normalizedPhoneNumber)
	if isRepeatingDigits(phoneStr[0:12]) {
		return fmt.Errorf("Invalid phone number")
	}

	*p = Phone(phoneStr)
	return nil
}

func isRepeatingDigits(phoneNumber string) bool {
	if len(phoneNumber) == 0 {
		return false
	}

	firstRune := rune(phoneNumber[0])
	for _, r := range phoneNumber {
		if firstRune != r && r != '-' && r != ' ' {
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
