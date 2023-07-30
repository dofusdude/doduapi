package utils

import (
	"testing"
)

func TestNewSet(t *testing.T) {
	s := NewSet()
	if s.Size() != 0 {
		t.Error("Expected 0, got ", s.Size())
	}
}

func TestSetHas(t *testing.T) {
	s := NewSet()
	s.Add("foo")
	if !s.Has("foo") {
		t.Error("Expected true, got ", s.Has("foo"))
	}
	if s.Has("bar") {
		t.Error("Expected false, got ", s.Has("bar"))
	}
}

func TestSetAdd(t *testing.T) {
	s := NewSet()
	s.Add("foo")
	if !s.Has("foo") {
		t.Error("Expected true, got ", s.Has("foo"))
	}
}

func TestSetRemove(t *testing.T) {
	s := NewSet()
	s.Add("foo")
	s.Remove("foo")
	if s.Has("foo") {
		t.Error("Expected false, got ", s.Has("foo"))
	}
}

func TestSetClear(t *testing.T) {
	s := NewSet()
	s.Add("foo")
	s.Clear()
	if s.Has("foo") {
		t.Error("Expected false, got ", s.Has("foo"))
	}
}

func TestSetSize(t *testing.T) {
	s := NewSet()
	s.Add("foo")
	if s.Size() != 1 {
		t.Error("Expected 1, got ", s.Size())
	}
}

func TestSetEmptyWithoutInit(t *testing.T) {
	var s Set
	if s.Size() != 0 {
		t.Error("Expected 0, got ", s.Size())
	}
}
