package utils

import "testing"

func TestGenerateNanoIdDefaultSize(t *testing.T) {
	id := GenerateNanoId()

	// Verify the length of the generated ID matches the default size
	if len(id) != defaultSize {
		t.Errorf("Expected length %d, got %d", defaultSize, len(id))
	}
}

func TestGenerateNanoIdCustomSize(t *testing.T) {
	customSize := 10
	id := GenerateNanoId(customSize)

	// Verify the length of the generated ID matches the custom size
	if len(id) != customSize {
		t.Errorf("Expected length %d, got %d", customSize, len(id))
	}
}

func TestGenerateNanoIdInvalidSize(t *testing.T) {
	invalidSize := -5
	id := GenerateNanoId(invalidSize)

	// Verify the length of the generated ID falls back to the default size
	if len(id) != defaultSize {
		t.Errorf("Expected length %d for invalid size, got %d", defaultSize, len(id))
	}
}

func TestGenerateNanoIdUniqueIds(t *testing.T) {
	count := 1000
	ids := make(map[string]bool)

	for i := 0; i < count; i++ {
		id := GenerateNanoId()

		// Check if the ID is already in the map
		if ids[id] {
			t.Errorf("Duplicate ID found: %s", id)
		}
		ids[id] = true
	}

	// Ensure the number of unique IDs matches the count
	if len(ids) != count {
		t.Errorf("Expected %d unique IDs, but got %d", count, len(ids))
	}
}
