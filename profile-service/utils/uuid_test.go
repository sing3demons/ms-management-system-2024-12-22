package utils

import (
	"testing"

	"github.com/google/uuid"
	"github.com/oklog/ulid"
)

func TestNewUUID(t *testing.T) {
	uuidStr := NewUUID()
	_, err := uuid.Parse(uuidStr)
	if err != nil {
		t.Errorf("Expected valid UUID, got %s", uuidStr)
	}
}

func TestNewShortUUID(t *testing.T) {
	shortUUIDStr := NewShortUUID()
	if len(shortUUIDStr) == 0 {
		t.Errorf("Expected non-empty short UUID, got %s", shortUUIDStr)
	}

	if len(shortUUIDStr) != 22 {
		t.Errorf("Expected short UUID length to be 22, got %d", len(shortUUIDStr))
	}
}

func TestNewULID(t *testing.T) {
	ulidStr := NewULID()
	_, err := ulid.Parse(ulidStr)
	if err != nil {
		t.Errorf("Expected valid ULID, got %s", ulidStr)
	}
}
