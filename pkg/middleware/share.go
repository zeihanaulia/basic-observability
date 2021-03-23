package http

import "strings"

func excludePath(path string) bool {
	return isContains(path, "metrics", "healthz")
}

func isContains(str string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(str, sub) {
			return true
		}
	}
	return false
}
