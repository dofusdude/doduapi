package main

import (
	"net/url"
	"testing"

	"github.com/dofusdude/doduapi/utils"
)

func TestPaginationLinksPreserveFiltersAndEncodeSeparateParameters(t *testing.T) {
	u, err := url.Parse("/dofus3/v1/en/items/equipment?filter[min_level]=100&sort[level]=desc&fields[item]=effects")
	if err != nil {
		t.Fatal(err)
	}
	p := utils.Pagination{PageNumber: 1, PageSize: 10}
	links, _ := p.BuildLinks(*u, 30, "http", "localhost:3000")
	if links.Next == nil {
		t.Fatal("missing next page")
	}
	next, err := url.Parse(*links.Next)
	if err != nil {
		t.Fatal(err)
	}
	if next.Query().Get("page[number]") != "2" || next.Query().Get("filter[min_level]") != "100" || next.Query().Get("fields[item]") != "effects" || next.Query().Get("sort[level]") != "desc" {
		t.Fatalf("broken links: %s", *links.Next)
	}
}

func TestPaginationSet(t *testing.T) {
	pagination := utils.PageninationWithState("30,3")

	if pagination.PageSize != 3 {
		t.Error("Expected 3, got ", t)
	}

	if pagination.PageNumber != 30 {
		t.Error("Expected 30, got ", t)
	}
}

func TestPaginationValidation(t *testing.T) {
	pagination := utils.PageninationWithState("31,3")
	listSize := 91

	valid := pagination.ValidatePagination(listSize)
	if valid != 0 {
		t.Error("Expected 0, got ", valid)
	}

	startIdx, endIdx := pagination.CalculateStartEndIndex(listSize)

	if startIdx != 90 {
		t.Error("Expected 90, got ", startIdx)
	}

	if endIdx != 91 {
		t.Error("Expected 91, got ", endIdx)
	}
}

func TestPaginationValidationFail(t *testing.T) {
	pagination := utils.PageninationWithState("32,3")
	listSize := 91

	valid := pagination.ValidatePagination(listSize)

	if valid == 0 {
		t.Error("Expected 0, got ", valid)
	}
}

func TestPaginationValidation1(t *testing.T) {
	pagination := utils.PageninationWithState("1,6")
	listSize := 6

	valid := pagination.ValidatePagination(listSize)
	if valid != 0 {
		t.Error("Expected 0, got ", valid)
	}

	startIdx, endIdx := pagination.CalculateStartEndIndex(listSize)

	if startIdx != 0 {
		t.Error("Expected 0, got ", startIdx)
	}

	if endIdx != 6 {
		t.Error("Expected 6, got ", endIdx)
	}
}

func TestPaginationAllowsPageSizeLargerThanCollection(t *testing.T) {
	pagination := utils.PageninationWithState("1,30")
	listSize := 9

	if valid := pagination.ValidatePagination(listSize); valid != 0 {
		t.Fatalf("expected page to be valid, got %d", valid)
	}

	startIdx, endIdx := pagination.CalculateStartEndIndex(listSize)
	if startIdx != 0 || endIdx != listSize {
		t.Fatalf("expected clamped range 0:%d, got %d:%d", listSize, startIdx, endIdx)
	}
}

func TestPaginationLastSite(t *testing.T) {
	pagination := utils.PageninationWithState("30,3")
	listSize := 90
	valid := pagination.ValidatePagination(listSize)
	if valid != 0 {
		t.Error("Expected 0, got ", valid)
	}

	startIdx, endIdx := pagination.CalculateStartEndIndex(listSize)

	if startIdx != 87 {
		t.Error("Expected 87, got ", startIdx)
	}
	if endIdx != 90 {
		t.Error("Expected 90, got ", endIdx)
	}
}

func TestPaginationFirstSite(t *testing.T) {
	pagination := utils.PageninationWithState("1,3")
	listSize := 266
	valid := pagination.ValidatePagination(listSize)
	if valid != 0 {
		t.Error("Expected 0, got ", valid)
	}

	startIdx, endIdx := pagination.CalculateStartEndIndex(listSize)

	if startIdx != 0 {
		t.Error("Expected 0, got ", startIdx)
	}

	if endIdx != 3 {
		t.Error("Expected 3, got ", endIdx)
	}
}
