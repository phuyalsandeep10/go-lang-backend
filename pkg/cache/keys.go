package cache

import (
	"fmt"
	"strings"
)

// cache key for the list of all properties.
func PropertyListKey() string {
	return "properties:list"
}

// cache key for a paginated list of properties.
func PropertyListPaginatedKey(offset, limit int) string {
	return fmt.Sprintf("properties:list:offset:%d:limit:%d", offset, limit)
}

// normalize address components by converting to lowercase and abbreviating common terms.
func NormalizeAddressComponent(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	replacements := map[string]string{
		"drive":     "dr",
		"street":    "st",
		"avenue":    "ave",
		"road":      "rd",
		"boulevard": "blvd",
		"lane":      "ln",
		"circle":    "cir",
		"court":     "ct",
		"terrace":   "ter",
		"place":     "pl",
		"highway":   "hwy",
	}
	for full, abbr := range replacements {
		s = strings.ReplaceAll(s, " "+full, " "+abbr)
	}
	return s
}

// cache key for a specific property search based on street and city.
func PropertySpecificSearchKey(street, city string) string {
	return fmt.Sprintf("properties:search-specific:street:%s:city:%s", street, city)
}

// cache key for a specific property.
func PropertyKey(id string) string {
	return fmt.Sprintf("property:%s", id)
}

// cache key for the set of cache keys associated with a property.
func PropertyKeysSetKey(propertyID string) string {
	return fmt.Sprintf("property:keys:%s", propertyID)
}

// cache key for a specific user.
func UserKey(id string) string {
	return fmt.Sprintf("user:%s", id)
}
