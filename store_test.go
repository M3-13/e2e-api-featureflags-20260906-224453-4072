package main

import (
	"errors"
	"testing"
)

func TestStoreCreateAndGet(t *testing.T) {
	s := NewStore()
	f := Flag{Key: "alpha", Enabled: true, RolloutPercent: 50}
	if err := s.Create(f); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, ok := s.Get("alpha")
	if !ok {
		t.Fatal("expected flag to exist")
	}
	if got != f {
		t.Fatalf("expected %+v, got %+v", f, got)
	}
}

func TestStoreCreateDuplicate(t *testing.T) {
	s := NewStore()
	f := Flag{Key: "dup", Enabled: true}
	if err := s.Create(f); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := s.Create(f); !errors.Is(err, ErrFlagExists) {
		t.Fatalf("expected ErrFlagExists, got %v", err)
	}
}

func TestStoreListSorted(t *testing.T) {
	s := NewStore()
	for _, k := range []string{"charlie", "alpha", "bravo"} {
		if err := s.Create(Flag{Key: k, Enabled: true}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	flags := s.List()
	if len(flags) != 3 {
		t.Fatalf("expected 3 flags, got %d", len(flags))
	}
	want := []string{"alpha", "bravo", "charlie"}
	for i, k := range want {
		if flags[i].Key != k {
			t.Fatalf("expected key %q at index %d, got %q", k, i, flags[i].Key)
		}
	}
}

func TestStoreListEmpty(t *testing.T) {
	s := NewStore()
	flags := s.List()
	if flags == nil {
		t.Fatal("expected non-nil empty slice")
	}
	if len(flags) != 0 {
		t.Fatalf("expected 0 flags, got %d", len(flags))
	}
}

func TestStoreUpdatePartial(t *testing.T) {
	s := NewStore()
	if err := s.Create(Flag{Key: "k", Enabled: true, Description: "old", RolloutPercent: 10}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	enabled := false
	f, ok := s.Update("k", UpdatePatch{Enabled: &enabled})
	if !ok {
		t.Fatal("expected update to succeed")
	}
	if f.Enabled {
		t.Fatal("expected enabled to be false")
	}
	if f.Description != "old" {
		t.Fatalf("expected description unchanged, got %q", f.Description)
	}
	if f.RolloutPercent != 10 {
		t.Fatalf("expected rollout_percent unchanged, got %d", f.RolloutPercent)
	}
}

func TestStoreUpdateMissing(t *testing.T) {
	s := NewStore()
	if _, ok := s.Update("nope", UpdatePatch{}); ok {
		t.Fatal("expected update to fail for missing key")
	}
}

func TestStoreDelete(t *testing.T) {
	s := NewStore()
	if err := s.Create(Flag{Key: "k", Enabled: true}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !s.Delete("k") {
		t.Fatal("expected delete to succeed")
	}
	if _, ok := s.Get("k"); ok {
		t.Fatal("expected flag to be gone")
	}
	if s.Delete("k") {
		t.Fatal("expected second delete to fail")
	}
}
