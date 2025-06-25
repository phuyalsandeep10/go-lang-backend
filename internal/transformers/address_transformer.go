package transformers

import (
	"regexp"
	"strings"
)

type addressTransformer struct{}

func NewAddressTransformer() AddressTransformer {
	return &addressTransformer{}
}

func (t *addressTransformer) NormalizeAddressComponent(input string) string {
	return strings.ToUpper(strings.TrimSpace(input))
}

func (t *addressTransformer) ParseAddress(search string) (street, city, state, zip string) {
	search = t.NormalizeAddressComponent(search)
	if search == "" {
		return "", "", "", ""
	}

	re := regexp.MustCompile(`^(.*?),\s*([^,]+),\s*([A-Z]{2})\s*(\d{5})$`)
	matches := re.FindStringSubmatch(search)
	if len(matches) == 5 {
		return t.NormalizeAddressComponent(matches[1]), t.NormalizeAddressComponent(matches[2]),
			t.NormalizeAddressComponent(matches[3]), t.NormalizeAddressComponent(matches[4])
	}

	parts := strings.Split(search, ",")
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}
	if len(parts) >= 2 {
		street := t.NormalizeAddressComponent(parts[0])
		city := t.NormalizeAddressComponent(parts[1])
		var state, zip string
		if len(parts) > 2 {
			stateZip := strings.Fields(parts[2])
			if len(stateZip) >= 2 {
				state = t.NormalizeAddressComponent(stateZip[0])
				zip = t.NormalizeAddressComponent(stateZip[1])
			} else if len(stateZip) == 1 {
				if regexp.MustCompile(`^[A-Z]{2}$`).MatchString(stateZip[0]) {
					state = t.NormalizeAddressComponent(stateZip[0])
				} else if regexp.MustCompile(`^\d{5}$`).MatchString(stateZip[0]) {
					zip = t.NormalizeAddressComponent(stateZip[0])
				}
			}
		}
		return street, city, state, zip
	}

	return t.NormalizeAddressComponent(search), "", "", ""
}
