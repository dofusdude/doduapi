package query

import (
	"strings"
	"testing"
)

func TestMatchEveryEndpointAndRejectUnrelatedPaths(t *testing.T) {
	seen := map[string]bool{}
	for _, endpoint := range Endpoints {
		if seen[endpoint.Path] {
			t.Fatalf("duplicate contract %s", endpoint.Path)
		}
		seen[endpoint.Path] = true
		for _, game := range []string{"dofus3", "dofus3beta"} {
			for _, language := range []string{"en", "fr", "de", "es", "pt"} {
				path := strings.ReplaceAll(strings.ReplaceAll(endpoint.Path, "{game}", game), "{language}", language)
				if got, ok := Match(path + "/"); !ok || got.Path != endpoint.Path {
					t.Fatalf("no match: %s", path)
				}
			}
		}
	}
	for _, path := range []string{"/spells", "/other/v1/en/spells", "/dofus3/v2/en/spells", "/dofus3/v1/unknown/items/equipment", "/dofus3/v1/meta/version", "/dofus3/v1/en/spells/201", "/dofus3/v1/update/token", "/dofus3/v1/img/spell/1.png"} {
		if _, ok := Match(path); ok {
			t.Fatalf("unexpected QUERY endpoint %s", path)
		}
	}
}
