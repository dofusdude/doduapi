package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/dofusdude/doduapi/database"
	"github.com/dofusdude/doduapi/query"
	mapping "github.com/dofusdude/dodumap"
)

func TestAllTransportDecodersProduceSameTypedRequest(t *testing.T) {
	for _, endpoint := range query.Endpoints {
		if endpoint.Mapped {
			continue
		}
		t.Run(endpoint.Path, func(t *testing.T) {
			body := map[string]any{}
			values := url.Values{}
			for _, parameter := range endpoint.Parameters {
				var value any
				var raw string
				switch parameter.Type {
				case "integer":
					value, raw = 7, "7"
				case "boolean":
					value, raw = false, "false"
				case "strings":
					value, raw = []string{"hat", "-ring"}, "hat,-ring"
				default:
					value, raw = "example", "example"
				}
				if parameter.Key == "from" || parameter.Key == "to" {
					value, raw = "2026-11-05", "2026-11-05"
				}
				if parameter.Key == "timezone" {
					value, raw = "Europe/Paris", "Europe/Paris"
				}
				// This combination is deliberately forbidden by the common validator.
				if parameter.Name == "range[size]" {
					continue
				}
				values.Set(parameter.Name, raw)
				if parameter.Group == "" {
					body[parameter.Key] = value
				} else {
					group, ok := body[parameter.Group].(map[string]any)
					if !ok {
						group = map[string]any{}
						body[parameter.Group] = group
					}
					keys := strings.Split(parameter.Key, ".")
					for _, key := range keys[:len(keys)-1] {
						nested, ok := group[key].(map[string]any)
						if !ok {
							nested = map[string]any{}
							group[key] = nested
						}
						group = nested
					}
					group[keys[len(keys)-1]] = value
				}
			}
			if len(endpoint.Sort) > 0 {
				body["sort"] = []string{"-" + endpoint.Sort[0]}
				values.Set("sort["+endpoint.Sort[0]+"]", "desc")
			}
			if endpoint.Fields != "" {
				body["fields"] = map[string][]string{endpoint.Fields: {endpoint.FieldValues[0]}}
				values.Set("fields["+endpoint.Fields+"]", endpoint.FieldValues[0])
			}
			encoded, err := json.Marshal(body)
			if err != nil {
				t.Fatal(err)
			}
			get, err := decodeGETQuery(endpoint, values)
			if err != nil {
				t.Fatal(err)
			}
			if err := validateRequest(endpoint, get, false); err != nil {
				t.Fatal(err)
			}
			structured, err := decodeValidatedQuery(endpoint, encoded, nil)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(get, structured) {
				t.Fatalf("GET %#v != QUERY %#v", get, structured)
			}
		})
	}
}

func TestSharedTransportDoesNotRewriteRequestURL(t *testing.T) {
	for _, endpoint := range query.Endpoints {
		path := strings.ReplaceAll(strings.ReplaceAll(endpoint.Path, "{game}", "dofus3"), "{language}", "en")
		for _, method := range []string{"GET", "QUERY"} {
			t.Run(method+path, func(t *testing.T) {
				body := `{}`
				if endpoint.Mapped {
					body = `{"filter":{"ids":[1]},"page":{"number":1,"size":3}}`
				}
				request := httptest.NewRequest(method, path+"?page%5Bnumber%5D=2", strings.NewReader(body))
				if !endpoint.Mapped {
					// Search and /all endpoints do not accept pagination overrides.
					request = httptest.NewRequest(method, path, strings.NewReader(`{}`))
				}
				request.Header.Set("Content-Type", "application/json")
				original := request.URL.String()
				called := false
				shared := decodeQueryTransport(http.HandlerFunc(func(w http.ResponseWriter, decoded *http.Request) {
					called = true
					if decoded.URL.String() != original {
						t.Fatalf("URL rewritten: %s", decoded.URL)
					}
					if endpoint.Mapped {
						q := decoded.Context().Value(entityQueryContextKey{}).(*EntityQuery)
						if q.Page.Number != 2 {
							t.Fatal("typed pagination override lost")
						}
					} else if query.From(decoded) == nil {
						t.Fatal("missing typed request")
					}
				}))
				w := httptest.NewRecorder()
				shared.ServeHTTP(w, request)
				if !called {
					t.Fatalf("shared handler not reached: %d %s", w.Code, w.Body)
				}
				if request.URL.String() != original {
					t.Fatal("original URL mutated")
				}
			})
		}
	}
}

func TestSharedValidationRejectsMalformedGETAndQUERY(t *testing.T) {
	for _, number := range []int{0, -1, 101} {
		// Use a search profile for limit validation and shared field types.
		search, _ := query.Match("/dofus3/v1/en/items/equipment/search")
		get, err := decodeGETQuery(search, url.Values{"limit": {strconv.Itoa(number)}})
		if err != nil {
			t.Fatal(err)
		}
		if err := validateRequest(search, get, false); (err != nil) != (number > 100) {
			t.Fatalf("legacy GET limit behavior changed for %d: %v", number, err)
		}
		if _, err := decodeValidatedQuery(search, []byte(fmt.Sprintf(`{"limit":%d}`, number)), nil); err == nil {
			t.Fatal("invalid QUERY limit accepted")
		}
	}
}

func TestQueryContractRoutesAndDiscovery(t *testing.T) {
	router := setupEntityQueryTest(t)
	for _, endpoint := range query.Endpoints {
		path := strings.ReplaceAll(strings.ReplaceAll(endpoint.Path, "{game}", "dofus3"), "{language}", "en")
		t.Run(path, func(t *testing.T) {
			w := queryTestRequest(router, "OPTIONS", path, "")
			if w.Code != 204 || w.Header().Get("Accept-Query") != "application/json" {
				t.Fatalf("route missing: %d %s", w.Code, w.Body)
			}
			w = queryTestRequest(router, "QUERY", path, `{"unsupported":true}`)
			if w.Code != 400 {
				t.Fatalf("closed request schema: %d %s", w.Code, w.Body)
			}
			r := httptest.NewRequest("OPTIONS", path, nil)
			r.Header.Set("Origin", "http://localhost:5173")
			r.Header.Set("Access-Control-Request-Method", "QUERY")
			r.Header.Set("Access-Control-Request-Headers", "content-type")
			w = httptest.NewRecorder()
			router.ServeHTTP(w, r)
			if w.Code != 204 {
				t.Fatalf("preflight: %d", w.Code)
			}
		})
	}
}

func TestQueryContractSnapshot(t *testing.T) {
	checkQueryContractSnapshot(t, "testdata/query-contract.json")
}

func TestAPIDocsQueryContractSnapshot(t *testing.T) {
	filename := os.Getenv("API_DOCS_QUERY_CONTRACT")
	if filename == "" {
		t.Skip("Set API_DOCS_QUERY_CONTRACT to check an exported api-docs contract")
	}
	checkQueryContractSnapshot(t, filename)
}

func checkQueryContractSnapshot(t *testing.T, filename string) {
	t.Helper()
	body, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	var published []query.Endpoint
	if err := json.Unmarshal(body, &published); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(published, query.Endpoints) {
		t.Fatalf("Query contract is stale: go run ./cmd/query-contract -output %q", filename)
	}
}

func decodeValidatedQuery(endpoint query.Endpoint, body []byte, overrides url.Values) (*query.Request, error) {
	request, err := decodeJSONQuery(endpoint, body, overrides)
	if err == nil {
		err = validateRequest(endpoint, request, true)
	}
	return request, err
}

func TestTypedQueryMatchesGET(t *testing.T) {
	endpoint, _ := query.Match("/dofus3/v1/en/items/equipment")
	values, err := decodeValidatedQuery(endpoint, []byte(`{"filter":{"min_level":100,"max_level":200,"type":{"name_id":["hat","-ring"]}},"sort":["-level"],"fields":{"item":["effects","pods"]},"page":{"number":1,"size":10}}`), url.Values{"page[number]": {"2"}})
	if err != nil {
		t.Fatal(err)
	}
	get := url.Values{"filter[min_level]": {"100"}, "filter[max_level]": {"200"}, "filter[type.name_id]": {"hat,-ring"}, "sort[level]": {"desc"}, "fields[item]": {"effects,pods"}, "page[number]": {"2"}, "page[size]": {"10"}}
	want, err := decodeGETQuery(endpoint, get)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateRequest(endpoint, want, false); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(values, want) {
		t.Fatalf("%v != %v", values, want)
	}
	for _, body := range []string{`{"filter":{"min_level":"100"}}`, `{"filter":{"min_level":1.5}}`, `{"filter":{"unknown":1}}`, `{"filter":{"type.name_id":"hat"}}`, `{"sort":["name"]}`, `{"fields":{"mount":["effects"]}}`, `{"page":{"size":101}}`, `{"include":["spells"]}`} {
		if _, err := decodeValidatedQuery(endpoint, []byte(body), nil); err == nil {
			t.Errorf("accepted invalid body %s", body)
		}
	}
}

func TestQueryRequiresNestedFilterObjects(t *testing.T) {
	for _, tc := range []struct {
		path, valid string
		invalid     []string
	}{
		{"/dofus3/v1/en/items/equipment", `{"filter":{"type":{"name_id":["hat","-ring"]}}}`, []string{
			`{"filter":{"type.name_id":["hat"]}}`,
			`{"filter":{"type":{"name_id":"hat"}}}`,
			`{"filter":{"type":{"unknown":["hat"]}}}`,
			`{"filter":{"type":{"name_id":["hat"],"name":"Hat"}}}`,
			`{"filter":{"type":["hat"]}}`,
			`{"filter":{"type.name_id":["ring"],"type":{"name_id":["hat"]}}}`,
		}},
		{"/dofus3/v1/en/mounts", `{"filter":{"family":{"id":242,"name":"Dragoturkey"}}}`, []string{
			`{"filter":{"family.id":242}}`,
			`{"filter":{"family.name":"Dragoturkey"}}`,
			`{"filter":{"family":{"id":"242"}}}`,
			`{"filter":{"family":{"id":242,"unknown":1}}}`,
			`{"filter":{"family":242}}`,
		}},
	} {
		t.Run(tc.path, func(t *testing.T) {
			endpoint, _ := query.Match(tc.path)
			if _, err := decodeValidatedQuery(endpoint, []byte(tc.valid), nil); err != nil {
				t.Fatal(err)
			}
			for _, body := range tc.invalid {
				if _, err := decodeValidatedQuery(endpoint, []byte(body), nil); err == nil {
					t.Errorf("accepted %s", body)
				}
			}
		})
	}
}

func TestAlmanaxQueryUsesExistingRangeParametersAndValidation(t *testing.T) {
	endpoint, _ := query.Match("/dofus3/v1/en/almanax")
	values, err := decodeValidatedQuery(endpoint, []byte(`{"range":{"from":"2026-11-05","size":7},"filter":{"bonus_type":"Offering"},"level":200,"timezone":"Europe/Paris"}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	if values.Date("range[from]").Format("2006-01-02") != "2026-11-05" || values.Int("range[size]") != 7 || values.Int("level") != 200 || values.String("filter[bonus_type]") != "Offering" || values.Location.String() != "Europe/Paris" {
		t.Fatalf("range parameters lost: %v", values)
	}
	router := setupEntityQueryTest(t)
	get := queryTestRequest(router, "GET", "/dofus3/v1/en/almanax?level=0", "")
	post := queryTestRequest(router, "QUERY", "/dofus3/v1/en/almanax", `{"level":0}`)
	if get.Code != 400 || post.Code != 400 || get.Body.String() != post.Body.String() {
		t.Fatalf("Almanax validation differs: %d/%d", get.Code, post.Code)
	}
}

func TestLegacyQueryGETParity(t *testing.T) {
	router := setupEntityQueryTest(t)
	txn := database.Db.Txn(true)
	for _, category := range []string{"equipment", "consumables", "resources", "quest_items", "cosmetics"} {
		for id, level := range []int{50, 120, 180} {
			item := &mapping.MappedMultilangItemUnity{AnkamaId: id + 1, Level: level, Name: map[string]string{"en": "Fixture"}, Description: map[string]string{"en": "Description"}, Type: mapping.MappedMultilangItemType{ItemTypeId: 242, Name: map[string]string{"en": "Mount"}}}
			if err := txn.Insert("red-"+category, item); err != nil {
				txn.Abort()
				t.Fatal(err)
			}
		}
	}
	for id, level := range []int{50, 120, 180} {
		set := &mapping.MappedMultilangSetUnity{AnkamaId: id + 1, Level: level, Name: map[string]string{"en": "Set"}, ContainsCosmetics: id == 2, ItemIds: []int{1}}
		if err := txn.Insert("red-sets", set); err != nil {
			txn.Abort()
			t.Fatal(err)
		}
	}
	txn.Commit()
	for _, category := range []string{"equipment", "consumables", "resources", "quest", "cosmetics"} {
		t.Run(category, func(t *testing.T) {
			base := "/dofus3/v1/en/items/" + category
			body := `{"filter":{"min_level":100,"max_level":200},"sort":["-level"],"page":{"size":1},"fields":{"item":["pods"]}}`
			get := queryTestRequest(router, "GET", base+"?filter[min_level]=100&filter[max_level]=200&sort[level]=desc&page[size]=1&fields[item]=pods", "")
			w := queryTestRequest(router, "QUERY", base, body)
			if get.Code != 200 || w.Code != 200 {
				t.Fatalf("GET/QUERY %d/%d: %s", get.Code, w.Code, w.Body)
			}
			var a, b APIPageItem
			json.Unmarshal(get.Body.Bytes(), &a)
			json.Unmarshal(w.Body.Bytes(), &b)
			if !reflect.DeepEqual(a.Items, b.Items) {
				t.Fatalf("GET and QUERY differ")
			}
			if b.Items[0].Level != 180 || b.Links.Next == nil {
				t.Fatal("filters/sorting/pagination missing")
			}
			next := queryTestRequest(router, "QUERY", *b.Links.Next, body)
			json.Unmarshal(next.Body.Bytes(), &b)
			if next.Code != 200 || b.Items[0].Level != 120 {
				t.Fatalf("next page lost body: %d %s", next.Code, next.Body)
			}
		})
	}
	for _, entry := range []struct{ path, body, get string }{
		{"sets", `{"filter":{"min_highest_equipment_level":100,"contains_cosmetics":true},"sort":["-level"]}`, "filter[min_highest_equipment_level]=100&filter[contains_cosmetics]=true&sort[level]=desc"},
		{"mounts", `{"filter":{"family":{"id":242}},"fields":{"mount":["effects"]}}`, "filter[family.id]=242&fields[mount]=effects"},
	} {
		get := queryTestRequest(router, "GET", "/dofus3/v1/en/"+entry.path+"?"+entry.get, "")
		w := queryTestRequest(router, "QUERY", "/dofus3/v1/en/"+entry.path, entry.body)
		if get.Code != 200 || w.Code != 200 {
			t.Fatalf("%s: %d/%d %s", entry.path, get.Code, w.Code, w.Body)
		}
		var a, b map[string]any
		json.Unmarshal(get.Body.Bytes(), &a)
		json.Unmarshal(w.Body.Bytes(), &b)
		delete(a, "_links")
		delete(b, "_links")
		if !reflect.DeepEqual(a, b) {
			t.Fatalf("%s GET/QUERY mismatch", entry.path)
		}
	}
}
