package utils

import (
	"fmt"
	"net/url"
)

func BuildPaginationURL(baseURL string, offset, limit int, params url.Values) string {
	u, _ := url.Parse(baseURL)
	q := url.Values{}
	q.Set("offset", fmt.Sprintf("%d", offset))
	q.Set("limit", fmt.Sprintf("%d", limit))
	for key, values := range params {
		if key != "offset" && key != "limit" {
			for _, value := range values {
				q.Add(key, value)
			}
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}
