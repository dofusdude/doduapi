package apitest

import "testing"

func TestMissingFieldsAndUnfriendlyValuesAreRejected(t *testing.T) {
	for key, fields := range requiredFields {
		valid := Object{}
		for _, field := range fields {
			valid[field] = true
		}
		if err := RequiredFields(key, valid); err != nil {
			t.Fatal(err)
		}
		for _, field := range fields {
			delete(valid, field)
			if RequiredFields(key, valid) == nil {
				t.Fatalf("missing %s.%s went undetected", key, field)
			}
			valid[field] = true
		}
	}
	for _, value := range []any{Object{"known": true}, Object{"nested": Object{"resolved": true}}, Object{"params": []int{1, 0}}, Object{"name": Object{"en": "Untranslated"}}, Object{"condition_expression": "CS>100"}} {
		if ValidatePublic(value) == nil {
			t.Fatalf("accepted bad API data: %#v", value)
		}
	}
}

func TestMappedComparisonRejectsNestedDataLoss(t *testing.T) {
	source := Object{"name": Object{"en": "Spell"}, "levels": []any{Object{"ap_cost": float64(3), "effects": []any{Object{"zone": Object{"shape": Object{"name": "circle"}}}}}}}
	valid := Object{"name": "Spell", "levels": []any{Object{"ap_cost": float64(3), "effects": []any{Object{"zone": Object{"shape": Object{"name": "circle"}}}}}}}
	if err := CompareMapped(source, valid, "en", "spells"); err != nil {
		t.Fatal(err)
	}
	Obj(List(valid["levels"])[0])["effects"] = []any{}
	if CompareMapped(source, valid, "en", "spells") == nil {
		t.Fatal("lost effect/zone data was accepted")
	}
}
