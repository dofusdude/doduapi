package apitest

import "testing"

func TestBaselineRejectsSharedDataLoss(t *testing.T) {
	data := &Data{Collections: []Collection{{Path: "spells", Records: []Record{{ID: 1}, {ID: 2}, {ID: 3}}}}, Almanax: []Object{{}}, Recipes: []Object{{}}}
	baseline := Snapshot(data)
	if err := baseline.Check(data); err != nil {
		t.Fatal(err)
	}
	data.Collections[0].Records = data.Collections[0].Records[:2]
	if baseline.Check(data) == nil {
		t.Fatal("mapper and API could silently lose the same record")
	}
	data.Collections[0].Records = append(data.Collections[0].Records, Record{ID: 4})
	if baseline.Check(data) == nil {
		t.Fatal("replacement of required ID went undetected")
	}
}
