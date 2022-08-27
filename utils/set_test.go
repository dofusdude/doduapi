package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewSet(t *testing.T) {
	s := NewSet()
	assert.Equal(t, 0, s.Size())
}

func TestSetHas(t *testing.T) {
	s := NewSet()
	s.Add("foo")
	assert.True(t, s.Has("foo"))
	assert.False(t, s.Has("bar"))
}

func TestSetAdd(t *testing.T) {
	s := NewSet()
	s.Add("foo")
	assert.True(t, s.Has("foo"))
}

func TestSetRemove(t *testing.T) {
	s := NewSet()
	s.Add("foo")
	s.Remove("foo")
	assert.False(t, s.Has("foo"))
}

func TestSetClear(t *testing.T) {
	s := NewSet()
	s.Add("foo")
	s.Clear()
	assert.False(t, s.Has("foo"))
}

func TestSetSize(t *testing.T) {
	s := NewSet()
	s.Add("foo")
	assert.Equal(t, 1, s.Size())
}

func TestSetEmptyWithoutInit(t *testing.T) {
	var s Set
	assert.Equal(t, 0, s.Size())
}
