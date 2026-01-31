package models

import (
	"regexp"
	"strings"
	"unicode"
)

// CountryInfo holds extracted country information
type CountryInfo struct {
	Code string
	Name string
}

// Country code to name mapping (ISO 3166-1 alpha-2)
var countryNames = map[string]string{
	"AD": "Andorra", "AE": "UAE", "AF": "Afghanistan", "AG": "Antigua", "AL": "Albania",
	"AM": "Armenia", "AO": "Angola", "AR": "Argentina", "AT": "Austria", "AU": "Australia",
	"AZ": "Azerbaijan", "BA": "Bosnia", "BB": "Barbados", "BD": "Bangladesh", "BE": "Belgium",
	"BG": "Bulgaria", "BH": "Bahrain", "BI": "Burundi", "BJ": "Benin", "BN": "Brunei",
	"BO": "Bolivia", "BR": "Brazil", "BS": "Bahamas", "BT": "Bhutan", "BW": "Botswana",
	"BY": "Belarus", "BZ": "Belize", "CA": "Canada", "CD": "Congo", "CF": "CAR",
	"CG": "Congo", "CH": "Switzerland", "CI": "Ivory Coast", "CL": "Chile", "CM": "Cameroon",
	"CN": "China", "CO": "Colombia", "CR": "Costa Rica", "CU": "Cuba", "CV": "Cape Verde",
	"CY": "Cyprus", "CZ": "Czechia", "DE": "Germany", "DJ": "Djibouti", "DK": "Denmark",
	"DM": "Dominica", "DO": "Dominican Rep", "DZ": "Algeria", "EC": "Ecuador", "EE": "Estonia",
	"EG": "Egypt", "ER": "Eritrea", "ES": "Spain", "ET": "Ethiopia", "FI": "Finland",
	"FJ": "Fiji", "FR": "France", "GA": "Gabon", "GB": "UK", "GD": "Grenada",
	"GE": "Georgia", "GH": "Ghana", "GM": "Gambia", "GN": "Guinea", "GQ": "Eq Guinea",
	"GR": "Greece", "GT": "Guatemala", "GW": "Guinea-Bissau", "GY": "Guyana", "HK": "Hong Kong",
	"HN": "Honduras", "HR": "Croatia", "HT": "Haiti", "HU": "Hungary", "ID": "Indonesia",
	"IE": "Ireland", "IL": "Israel", "IN": "India", "IQ": "Iraq", "IR": "Iran",
	"IS": "Iceland", "IT": "Italy", "JM": "Jamaica", "JO": "Jordan", "JP": "Japan",
	"KE": "Kenya", "KG": "Kyrgyzstan", "KH": "Cambodia", "KI": "Kiribati", "KM": "Comoros",
	"KN": "St Kitts", "KP": "North Korea", "KR": "South Korea", "KW": "Kuwait", "KZ": "Kazakhstan",
	"LA": "Laos", "LB": "Lebanon", "LC": "St Lucia", "LI": "Liechtenstein", "LK": "Sri Lanka",
	"LR": "Liberia", "LS": "Lesotho", "LT": "Lithuania", "LU": "Luxembourg", "LV": "Latvia",
	"LY": "Libya", "MA": "Morocco", "MC": "Monaco", "MD": "Moldova", "ME": "Montenegro",
	"MG": "Madagascar", "MK": "N Macedonia", "ML": "Mali", "MM": "Myanmar", "MN": "Mongolia",
	"MO": "Macau", "MR": "Mauritania", "MT": "Malta", "MU": "Mauritius", "MV": "Maldives",
	"MW": "Malawi", "MX": "Mexico", "MY": "Malaysia", "MZ": "Mozambique", "NA": "Namibia",
	"NE": "Niger", "NG": "Nigeria", "NI": "Nicaragua", "NL": "Netherlands", "NO": "Norway",
	"NP": "Nepal", "NR": "Nauru", "NZ": "New Zealand", "OM": "Oman", "PA": "Panama",
	"PE": "Peru", "PG": "Papua New Guinea", "PH": "Philippines", "PK": "Pakistan", "PL": "Poland",
	"PR": "Puerto Rico", "PT": "Portugal", "PW": "Palau", "PY": "Paraguay", "QA": "Qatar",
	"RO": "Romania", "RS": "Serbia", "RU": "Russia", "RW": "Rwanda", "SA": "Saudi Arabia",
	"SB": "Solomon Islands", "SC": "Seychelles", "SD": "Sudan", "SE": "Sweden", "SG": "Singapore",
	"SI": "Slovenia", "SK": "Slovakia", "SL": "Sierra Leone", "SM": "San Marino", "SN": "Senegal",
	"SO": "Somalia", "SR": "Suriname", "SS": "South Sudan", "ST": "Sao Tome", "SV": "El Salvador",
	"SY": "Syria", "SZ": "Eswatini", "TD": "Chad", "TG": "Togo", "TH": "Thailand",
	"TJ": "Tajikistan", "TL": "Timor-Leste", "TM": "Turkmenistan", "TN": "Tunisia", "TO": "Tonga",
	"TR": "Turkey", "TT": "Trinidad", "TV": "Tuvalu", "TW": "Taiwan", "TZ": "Tanzania",
	"UA": "Ukraine", "UG": "Uganda", "UK": "UK", "US": "USA", "UY": "Uruguay",
	"UZ": "Uzbekistan", "VA": "Vatican", "VC": "St Vincent", "VE": "Venezuela", "VN": "Vietnam",
	"VU": "Vanuatu", "WS": "Samoa", "YE": "Yemen", "ZA": "South Africa", "ZM": "Zambia",
	"ZW": "Zimbabwe",
}

// Common alternative names to country codes
var countryAliases = map[string]string{
	"UNITED STATES":  "US",
	"AMERICA":        "US",
	"USA":            "US",
	"UNITED KINGDOM": "GB",
	"ENGLAND":        "GB",
	"BRITAIN":        "GB",
	"HONG KONG":      "HK",
	"SINGAPORE":      "SG",
	"JAPAN":          "JP",
	"KOREA":          "KR",
	"SOUTH KOREA":    "KR",
	"CHINA":          "CN",
	"TAIWAN":         "TW",
	"GERMANY":        "DE",
	"FRANCE":         "FR",
	"NETHERLANDS":    "NL",
	"HOLLAND":        "NL",
	"CANADA":         "CA",
	"AUSTRALIA":      "AU",
	"RUSSIA":         "RU",
	"INDIA":          "IN",
	"BRAZIL":         "BR",
	"MEXICO":         "MX",
	"ITALY":          "IT",
	"SPAIN":          "ES",
	"TURKEY":         "TR",
	"POLAND":         "PL",
	"SWEDEN":         "SE",
	"NORWAY":         "NO",
	"FINLAND":        "FI",
	"DENMARK":        "DK",
	"IRELAND":        "IE",
	"SWITZERLAND":    "CH",
	"AUSTRIA":        "AT",
	"BELGIUM":        "BE",
	"CZECH":          "CZ",
	"CZECHIA":        "CZ",
	"ROMANIA":        "RO",
	"UKRAINE":        "UA",
	"INDONESIA":      "ID",
	"MALAYSIA":       "MY",
	"THAILAND":       "TH",
	"VIETNAM":        "VN",
	"PHILIPPINES":    "PH",
	"ISRAEL":         "IL",
	"UAE":            "AE",
	"DUBAI":          "AE",
	"ARGENTINA":      "AR",
	"CHILE":          "CL",
	"COLOMBIA":       "CO",
	"PERU":           "PE",
	"PORTUGAL":       "PT",
	"GREECE":         "GR",
	"HUNGARY":        "HU",
	"BULGARIA":       "BG",
	"SERBIA":         "RS",
	"CROATIA":        "HR",
	"SLOVAKIA":       "SK",
	"SLOVENIA":       "SI",
	"LUXEMBOURG":     "LU",
	"ESTONIA":        "EE",
	"LATVIA":         "LV",
	"LITHUANIA":      "LT",
	"ICELAND":        "IS",
	"CYPRUS":         "CY",
	"MALTA":          "MT",
	"EGYPT":          "EG",
	"SOUTH AFRICA":   "ZA",
	"NIGERIA":        "NG",
	"KENYA":          "KE",
	"MOROCCO":        "MA",
	"PAKISTAN":       "PK",
	"BANGLADESH":     "BD",
	"NEW ZEALAND":    "NZ",
	"KAZAKHSTAN":     "KZ",
	"SAUDI ARABIA":   "SA",
	"QATAR":          "QA",
	"BAHRAIN":        "BH",
	"KUWAIT":         "KW",
	"IRAN":           "IR",
	"IRAQ":           "IQ",
}

// Regex patterns for country code detection
var (
	// Match country codes at the start: "US-Server", "US_Server", "US Server", "[US]"
	countryCodePrefixPattern = regexp.MustCompile(`^[\[\(]?([A-Z]{2})[\]\)]?[-_\s]`)
	// Match country codes at the end: "Server-US", "Server_US", "Server US"
	countryCodeSuffixPattern = regexp.MustCompile(`[-_\s]([A-Z]{2})$`)
	// Match flag emoji (regional indicator symbols)
	flagEmojiPattern = regexp.MustCompile(`[\x{1F1E6}-\x{1F1FF}]{2}`)
)

// ExtractCountryInfo extracts country information from a proxy name
func ExtractCountryInfo(name string) CountryInfo {
	if name == "" {
		return CountryInfo{}
	}

	// Try to extract from flag emoji first
	if code := extractFromFlagEmoji(name); code != "" {
		return CountryInfo{Code: code, Name: countryNames[code]}
	}

	// Try to extract from country code prefix (e.g., "US-Server")
	upperName := strings.ToUpper(name)
	if matches := countryCodePrefixPattern.FindStringSubmatch(upperName); len(matches) > 1 {
		code := matches[1]
		if _, valid := countryNames[code]; valid {
			return CountryInfo{Code: code, Name: countryNames[code]}
		}
	}

	// Try to extract from country code suffix (e.g., "Server-US")
	if matches := countryCodeSuffixPattern.FindStringSubmatch(upperName); len(matches) > 1 {
		code := matches[1]
		if _, valid := countryNames[code]; valid {
			return CountryInfo{Code: code, Name: countryNames[code]}
		}
	}

	// Try to find country name in the proxy name
	for alias, code := range countryAliases {
		if strings.Contains(upperName, alias) {
			return CountryInfo{Code: code, Name: countryNames[code]}
		}
	}

	// Try to match standalone country codes anywhere in the name
	words := strings.FieldsFunc(upperName, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	for _, word := range words {
		if len(word) == 2 {
			if _, valid := countryNames[word]; valid {
				return CountryInfo{Code: word, Name: countryNames[word]}
			}
		}
	}

	return CountryInfo{}
}

// extractFromFlagEmoji extracts country code from flag emoji
func extractFromFlagEmoji(name string) string {
	matches := flagEmojiPattern.FindString(name)
	if matches == "" {
		return ""
	}

	// Convert regional indicator symbols to country code
	runes := []rune(matches)
	if len(runes) >= 2 {
		// Regional indicator symbols start at U+1F1E6 (A) and go to U+1F1FF (Z)
		// Subtract the base to get 0-25, then add 'A' to get the letter
		first := runes[0] - 0x1F1E6 + 'A'
		second := runes[1] - 0x1F1E6 + 'A'
		code := string([]rune{first, second})
		if _, valid := countryNames[code]; valid {
			return code
		}
	}
	return ""
}

// CountryCodeToFlag converts a 2-letter country code to a flag emoji
func CountryCodeToFlag(code string) string {
	if len(code) != 2 {
		return ""
	}
	code = strings.ToUpper(code)
	// Convert each letter to a regional indicator symbol
	// A = U+1F1E6, B = U+1F1E7, etc.
	first := rune(code[0]) - 'A' + 0x1F1E6
	second := rune(code[1]) - 'A' + 0x1F1E6
	return string([]rune{first, second})
}

// GetCountryName returns the country name for a code
func GetCountryName(code string) string {
	return countryNames[strings.ToUpper(code)]
}
