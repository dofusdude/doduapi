package main

import (
	"testing"
)

func TestPaginationSet(t *testing.T) {
	pagination := PageninationWithState("30,3")

	if pagination.PageSize != 3 {
		t.Error("Expected 3, got ", t)
	}

	if pagination.PageNumber != 30 {
		t.Error("Expected 30, got ", t)
	}
}

func TestPaginationValidation(t *testing.T) {
	pagination := PageninationWithState("31,3")
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
	pagination := PageninationWithState("32,3")
	listSize := 91

	valid := pagination.ValidatePagination(listSize)

	if valid == 0 {
		t.Error("Expected 0, got ", valid)
	}
}

func TestPaginationValidation1(t *testing.T) {
	pagination := PageninationWithState("1,6")
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

func TestPaginationLastSite(t *testing.T) {
	pagination := PageninationWithState("30,3")
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
	pagination := PageninationWithState("1,3")
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
