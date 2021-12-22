package web

import (
	"net/http"
)

func FindSessionCookie(cookies []*http.Cookie) *http.Cookie {
	for _, c := range cookies {
		if c.Name == "clsession" {
			return c
		}
	}

	return nil
}
