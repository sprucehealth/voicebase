package stripe

// This list of currencies represents what Stripe supports:
// https://support.stripe.com/questions/which-currencies-does-stripe-support

import (
	"fmt"
	"strings"
)

type Currency struct {
	ISO  string
	Name string
}

var (
	AED = Currency{ISO: "AED", Name: "United Arab Emirates Dirham"}
	AFN = Currency{ISO: "AFN", Name: "Afghan Afghani*"}
	ALL = Currency{ISO: "ALL", Name: "Albanian Lek"}
	AMD = Currency{ISO: "AMD", Name: "Armenian Dram"}
	ANG = Currency{ISO: "ANG", Name: "Netherlands Antillean Gulden"}
	AOA = Currency{ISO: "AOA", Name: "Angolan Kwanza*"}
	ARS = Currency{ISO: "ARS", Name: "Argentine Peso*"}
	AUD = Currency{ISO: "AUD", Name: "Australian Dollar*"}
	AWG = Currency{ISO: "AWG", Name: "Aruban Florin"}
	AZN = Currency{ISO: "AZN", Name: "Azerbaijani Manat"}
	BAM = Currency{ISO: "BAM", Name: "Bosnia & Herzegovina Convertible Mark"}
	BBD = Currency{ISO: "BBD", Name: "Barbadian Dollar"}
	BDT = Currency{ISO: "BDT", Name: "Bangladeshi Taka"}
	BGN = Currency{ISO: "BGN", Name: "Bulgarian Lev"}
	BIF = Currency{ISO: "BIF", Name: "Burundian Franc"}
	BMD = Currency{ISO: "BMD", Name: "Bermudian Dollar"}
	BND = Currency{ISO: "BND", Name: "Brunei Dollar"}
	BOB = Currency{ISO: "BOB", Name: "Bolivian Boliviano*"}
	BRL = Currency{ISO: "BRL", Name: "Brazilian Real*"}
	BSD = Currency{ISO: "BSD", Name: "Bahamian Dollar"}
	BWP = Currency{ISO: "BWP", Name: "Botswana Pula"}
	BZD = Currency{ISO: "BZD", Name: "Belize Dollar"}
	CAD = Currency{ISO: "CAD", Name: "Canadian Dollar"}
	CDF = Currency{ISO: "CDF", Name: "Congolese Franc"}
	CHF = Currency{ISO: "CHF", Name: "Swiss Franc"}
	CLP = Currency{ISO: "CLP", Name: "Chilean Peso*"}
	CNY = Currency{ISO: "CNY", Name: "Chinese Renminbi Yuan"}
	COP = Currency{ISO: "COP", Name: "Colombian Peso*"}
	CRC = Currency{ISO: "CRC", Name: "Costa Rican Colón*"}
	CVE = Currency{ISO: "CVE", Name: "Cape Verdean Escudo*"}
	CZK = Currency{ISO: "CZK", Name: "Czech Koruna*"}
	DJF = Currency{ISO: "DJF", Name: "Djiboutian Franc*"}
	DKK = Currency{ISO: "DKK", Name: "Danish Krone"}
	DOP = Currency{ISO: "DOP", Name: "Dominican Peso"}
	DZD = Currency{ISO: "DZD", Name: "Algerian Dinar"}
	EEK = Currency{ISO: "EEK", Name: "Estonian Kroon*"}
	EGP = Currency{ISO: "EGP", Name: "Egyptian Pound"}
	ETB = Currency{ISO: "ETB", Name: "Ethiopian Birr"}
	EUR = Currency{ISO: "EUR", Name: "Euro"}
	FJD = Currency{ISO: "FJD", Name: "Fijian Dollar"}
	FKP = Currency{ISO: "FKP", Name: "Falkland Islands Pound*"}
	GBP = Currency{ISO: "GBP", Name: "British Pound"}
	GEL = Currency{ISO: "GEL", Name: "Georgian Lari"}
	GIP = Currency{ISO: "GIP", Name: "Gibraltar Pound"}
	GMD = Currency{ISO: "GMD", Name: "Gambian Dalasi"}
	GNF = Currency{ISO: "GNF", Name: "Guinean Franc*"}
	GTQ = Currency{ISO: "GTQ", Name: "Guatemalan Quetzal*"}
	GYD = Currency{ISO: "GYD", Name: "Guyanese Dollar"}
	HKD = Currency{ISO: "HKD", Name: "Hong Kong Dollar"}
	HNL = Currency{ISO: "HNL", Name: "Honduran Lempira*"}
	HRK = Currency{ISO: "HRK", Name: "Croatian Kuna"}
	HTG = Currency{ISO: "HTG", Name: "Haitian Gourde"}
	HUF = Currency{ISO: "HUF", Name: "Hungarian Forint"}
	IDR = Currency{ISO: "IDR", Name: "Indonesian Rupiah"}
	ILS = Currency{ISO: "ILS", Name: "Israeli New Sheqel"}
	INR = Currency{ISO: "INR", Name: "Indian Rupee*"}
	ISK = Currency{ISO: "ISK", Name: "Icelandic Króna"}
	JMD = Currency{ISO: "JMD", Name: "Jamaican Dollar"}
	JPY = Currency{ISO: "JPY", Name: "Japanese Yen"}
	KES = Currency{ISO: "KES", Name: "Kenyan Shilling"}
	KGS = Currency{ISO: "KGS", Name: "Kyrgyzstani Som"}
	KHR = Currency{ISO: "KHR", Name: "Cambodian Riel"}
	KMF = Currency{ISO: "KMF", Name: "Comorian Franc"}
	KRW = Currency{ISO: "KRW", Name: "South Korean Won"}
	KYD = Currency{ISO: "KYD", Name: "Cayman Islands Dollar"}
	KZT = Currency{ISO: "KZT", Name: "Kazakhstani Tenge"}
	LAK = Currency{ISO: "LAK", Name: "Lao Kip*"}
	LBP = Currency{ISO: "LBP", Name: "Lebanese Pound"}
	LKR = Currency{ISO: "LKR", Name: "Sri Lankan Rupee"}
	LRD = Currency{ISO: "LRD", Name: "Liberian Dollar"}
	LSL = Currency{ISO: "LSL", Name: "Lesotho Loti"}
	LTL = Currency{ISO: "LTL", Name: "Lithuanian Litas"}
	LVL = Currency{ISO: "LVL", Name: "Latvian Lats"}
	MAD = Currency{ISO: "MAD", Name: "Moroccan Dirham"}
	MDL = Currency{ISO: "MDL", Name: "Moldovan Leu"}
	MGA = Currency{ISO: "MGA", Name: "Malagasy Ariary"}
	MKD = Currency{ISO: "MKD", Name: "Macedonian Denar"}
	MNT = Currency{ISO: "MNT", Name: "Mongolian Tögrög"}
	MOP = Currency{ISO: "MOP", Name: "Macanese Pataca"}
	MRO = Currency{ISO: "MRO", Name: "Mauritanian Ouguiya"}
	MUR = Currency{ISO: "MUR", Name: "Mauritian Rupee*"}
	MVR = Currency{ISO: "MVR", Name: "Maldivian Rufiyaa"}
	MWK = Currency{ISO: "MWK", Name: "Malawian Kwacha"}
	MXN = Currency{ISO: "MXN", Name: "Mexican Peso*"}
	MYR = Currency{ISO: "MYR", Name: "Malaysian Ringgit"}
	MZN = Currency{ISO: "MZN", Name: "Mozambican Metical"}
	NAD = Currency{ISO: "NAD", Name: "Namibian Dollar"}
	NGN = Currency{ISO: "NGN", Name: "Nigerian Naira"}
	NIO = Currency{ISO: "NIO", Name: "Nicaraguan Córdoba*"}
	NOK = Currency{ISO: "NOK", Name: "Norwegian Krone"}
	NPR = Currency{ISO: "NPR", Name: "Nepalese Rupee"}
	NZD = Currency{ISO: "NZD", Name: "New Zealand Dollar"}
	PAB = Currency{ISO: "PAB", Name: "Panamanian Balboa*"}
	PEN = Currency{ISO: "PEN", Name: "Peruvian Nuevo Sol*"}
	PGK = Currency{ISO: "PGK", Name: "Papua New Guinean Kina"}
	PHP = Currency{ISO: "PHP", Name: "Philippine Peso"}
	PKR = Currency{ISO: "PKR", Name: "Pakistani Rupee"}
	PLN = Currency{ISO: "PLN", Name: "Polish Złoty"}
	PYG = Currency{ISO: "PYG", Name: "Paraguayan Guaraní*"}
	QAR = Currency{ISO: "QAR", Name: "Qatari Riyal"}
	RON = Currency{ISO: "RON", Name: "Romanian Leu"}
	RSD = Currency{ISO: "RSD", Name: "Serbian Dinar"}
	RUB = Currency{ISO: "RUB", Name: "Russian Ruble"}
	RWF = Currency{ISO: "RWF", Name: "Rwandan Franc"}
	SAR = Currency{ISO: "SAR", Name: "Saudi Riyal"}
	SBD = Currency{ISO: "SBD", Name: "Solomon Islands Dollar"}
	SCR = Currency{ISO: "SCR", Name: "Seychellois Rupee"}
	SEK = Currency{ISO: "SEK", Name: "Swedish Krona"}
	SGD = Currency{ISO: "SGD", Name: "Singapore Dollar"}
	SHP = Currency{ISO: "SHP", Name: "Saint Helenian Pound*"}
	SLL = Currency{ISO: "SLL", Name: "Sierra Leonean Leone"}
	SOS = Currency{ISO: "SOS", Name: "Somali Shilling"}
	SRD = Currency{ISO: "SRD", Name: "Surinamese Dollar*"}
	STD = Currency{ISO: "STD", Name: "São Tomé and Príncipe Dobra"}
	SVC = Currency{ISO: "SVC", Name: "Salvadoran Colón*"}
	SZL = Currency{ISO: "SZL", Name: "Swazi Lilangeni"}
	THB = Currency{ISO: "THB", Name: "Thai Baht"}
	TJS = Currency{ISO: "TJS", Name: "Tajikistani Somoni"}
	TOP = Currency{ISO: "TOP", Name: "Tongan Paʻanga"}
	TRY = Currency{ISO: "TRY", Name: "Turkish Lira"}
	TTD = Currency{ISO: "TTD", Name: "Trinidad and Tobago Dollar"}
	TWD = Currency{ISO: "TWD", Name: "New Taiwan Dollar"}
	TZS = Currency{ISO: "TZS", Name: "Tanzanian Shilling"}
	UAH = Currency{ISO: "UAH", Name: "Ukrainian Hryvnia"}
	UGX = Currency{ISO: "UGX", Name: "Ugandan Shilling"}
	USD = Currency{ISO: "USD", Name: "United States Dollar"}
	UYU = Currency{ISO: "UYU", Name: "Uruguayan Peso*"}
	UZS = Currency{ISO: "UZS", Name: "Uzbekistani Som"}
	VEF = Currency{ISO: "VEF", Name: "Venezuelan Bolívar*"}
	VND = Currency{ISO: "VND", Name: "Vietnamese Đồng"}
	VUV = Currency{ISO: "VUV", Name: "Vanuatu Vatu"}
	WST = Currency{ISO: "WST", Name: "Samoan Tala"}
	XAF = Currency{ISO: "XAF", Name: "Central African Cfa Franc"}
	XCD = Currency{ISO: "XCD", Name: "East Caribbean Dollar"}
	XOF = Currency{ISO: "XOF", Name: "West African Cfa Franc*"}
	XPF = Currency{ISO: "XPF", Name: "Cfp Franc*"}
	YER = Currency{ISO: "YER", Name: "Yemeni Rial"}
	ZAR = Currency{ISO: "ZAR", Name: "South African Rand"}
	ZMW = Currency{ISO: "ZMW", Name: "Zambian Kwacha"}

	isoToCurrencyMapping = map[string]*Currency{
		"AED": &AED,
		"AFN": &AFN,
		"ALL": &ALL,
		"AMD": &AMD,
		"ANG": &ANG,
		"AOA": &AOA,
		"ARS": &ARS,
		"AUD": &AUD,
		"AWG": &AWG,
		"AZN": &AZN,
		"BAM": &BAM,
		"BBD": &BBD,
		"BDT": &BDT,
		"BGN": &BGN,
		"BIF": &BIF,
		"BMD": &BMD,
		"BND": &BND,
		"BOB": &BOB,
		"BRL": &BRL,
		"BSD": &BSD,
		"BWP": &BWP,
		"BZD": &BZD,
		"CAD": &CAD,
		"CDF": &CDF,
		"CHF": &CHF,
		"CLP": &CLP,
		"CNY": &CNY,
		"COP": &COP,
		"CRC": &CRC,
		"CVE": &CVE,
		"CZK": &CZK,
		"DJF": &DJF,
		"DKK": &DKK,
		"DOP": &DOP,
		"DZD": &DZD,
		"EEK": &EEK,
		"EGP": &EGP,
		"ETB": &ETB,
		"EUR": &EUR,
		"FJD": &FJD,
		"FKP": &FKP,
		"GBP": &GBP,
		"GEL": &GEL,
		"GIP": &GIP,
		"GMD": &GMD,
		"GNF": &GNF,
		"GTQ": &GTQ,
		"GYD": &GYD,
		"HKD": &HKD,
		"HNL": &HNL,
		"HRK": &HRK,
		"HTG": &HTG,
		"HUF": &HUF,
		"IDR": &IDR,
		"ILS": &ILS,
		"INR": &INR,
		"ISK": &ISK,
		"JMD": &JMD,
		"JPY": &JPY,
		"KES": &KES,
		"KGS": &KGS,
		"KHR": &KHR,
		"KMF": &KMF,
		"KRW": &KRW,
		"KYD": &KYD,
		"KZT": &KZT,
		"LAK": &LAK,
		"LBP": &LBP,
		"LKR": &LKR,
		"LRD": &LRD,
		"LSL": &LSL,
		"LTL": &LTL,
		"LVL": &LVL,
		"MAD": &MAD,
		"MDL": &MDL,
		"MGA": &MGA,
		"MKD": &MKD,
		"MNT": &MNT,
		"MOP": &MOP,
		"MRO": &MRO,
		"MUR": &MUR,
		"MVR": &MVR,
		"MWK": &MWK,
		"MXN": &MXN,
		"MYR": &MYR,
		"MZN": &MZN,
		"NAD": &NAD,
		"NGN": &NGN,
		"NIO": &NIO,
		"NOK": &NOK,
		"NPR": &NPR,
		"NZD": &NZD,
		"PAB": &PAB,
		"PEN": &PEN,
		"PGK": &PGK,
		"PHP": &PHP,
		"PKR": &PKR,
		"PLN": &PLN,
		"PYG": &PYG,
		"QAR": &QAR,
		"RON": &RON,
		"RSD": &RSD,
		"RUB": &RUB,
		"RWF": &RWF,
		"SAR": &SAR,
		"SBD": &SBD,
		"SCR": &SCR,
		"SEK": &SEK,
		"SGD": &SGD,
		"SHP": &SHP,
		"SLL": &SLL,
		"SOS": &SOS,
		"SRD": &SRD,
		"STD": &STD,
		"SVC": &SVC,
		"SZL": &SZL,
		"THB": &THB,
		"TJS": &TJS,
		"TOP": &TOP,
		"TRY": &TRY,
		"TTD": &TTD,
		"TWD": &TWD,
		"TZS": &TZS,
		"UAH": &UAH,
		"UGX": &UGX,
		"USD": &USD,
		"UYU": &UYU,
		"UZS": &UZS,
		"VEF": &VEF,
		"VND": &VND,
		"VUV": &VUV,
		"WST": &WST,
		"XAF": &XAF,
		"XCD": &XCD,
		"XOF": &XOF,
		"XPF": &XPF,
		"YER": &YER,
		"ZAR": &ZAR,
		"ZMW": &ZMW,
	}
)

func (c Currency) String() string {
	return fmt.Sprintf("%s (%s)", c.Name, c.ISO)
}

func getCurrency(isoCode string) (*Currency, error) {
	currency, ok := isoToCurrencyMapping[strings.ToUpper(isoCode)]
	if !ok {
		return nil, fmt.Errorf("Unknown iso code for currency: %s", isoCode)
	}
	return currency, nil
}

func (c *Currency) UnmarshalJSON(b []byte) error {
	s := string(b)

	if s == "null" {
		*c = Currency{}
		return nil
	}

	var err error
	var currency *Currency
	if len(s) > 2 && s[0] == '"' {
		currency, err = getCurrency(s[1 : len(s)-1])
	} else {
		currency, err = getCurrency(s)
	}

	if err != nil {
		return err
	}

	*c = *currency
	return nil
}
