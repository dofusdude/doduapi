package query

import "strings"

func pathMatches(pattern, path string) bool {
	want, got := strings.Split(strings.Trim(pattern, "/"), "/"), strings.Split(strings.Trim(path, "/"), "/")
	if len(want) != len(got) {
		return false
	}
	for i, part := range want {
		switch part {
		case "{game}":
			if got[i] != "dofus3" && got[i] != "dofus3beta" {
				return false
			}
		case "{language}":
			switch got[i] {
			case "en", "fr", "de", "es", "pt":
			default:
				return false
			}
		default:
			if part != got[i] {
				return false
			}
		}
	}
	return true
}
