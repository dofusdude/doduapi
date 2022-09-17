package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPaginationSet(t *testing.T) {
	pagination := PageninationWithState("30,3")

	assert.Equal(t, 3, pagination.PageSize)
	assert.Equal(t, 30, pagination.PageNumber)
}

func TestPaginationValidation(t *testing.T) {
	pagination := PageninationWithState("31,3")
	listSize := 91

	valid := pagination.ValidatePagination(listSize)
	assert.Equal(t, 0, valid)

	startIdx, endIdx := pagination.CalculateStartEndIndex(listSize)

	assert.Equal(t, 90, startIdx)
	assert.Equal(t, 91, endIdx)
}

func TestPaginationValidationFail(t *testing.T) {
	pagination := PageninationWithState("32,3")
	listSize := 91

	valid := pagination.ValidatePagination(listSize)
	assert.NotEqual(t, 0, valid)
}

func TestPaginationValidation1(t *testing.T) {
	pagination := PageninationWithState("1,6")
	listSize := 6

	valid := pagination.ValidatePagination(listSize)
	assert.Equal(t, 0, valid)

	startIdx, endIdx := pagination.CalculateStartEndIndex(listSize)

	assert.Equal(t, 0, startIdx)
	assert.Equal(t, 6, endIdx)
}

func TestPaginationLastSite(t *testing.T) {
	pagination := PageninationWithState("30,3")
	listSize := 90
	valid := pagination.ValidatePagination(listSize)
	assert.Equal(t, 0, valid)

	startIdx, endIdx := pagination.CalculateStartEndIndex(listSize)

	assert.Equal(t, 87, startIdx)
	assert.Equal(t, 90, endIdx)
}

func TestPaginationFirstSite(t *testing.T) {
	pagination := PageninationWithState("1,3")
	listSize := 266
	valid := pagination.ValidatePagination(listSize)
	assert.Equal(t, 0, valid)

	startIdx, endIdx := pagination.CalculateStartEndIndex(listSize)

	assert.Equal(t, 0, startIdx)
	assert.Equal(t, 3, endIdx)
}
