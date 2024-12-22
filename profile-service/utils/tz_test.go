package utils

import (
	"testing"
	"time"
)

func TestLocBkk(t *testing.T) {
	loc := LocBkk()

	// Verify that the returned location is not nil
	if loc == nil {
		t.Fatal("Expected non-nil location, but got nil")
	}

	// Verify that the location name is "Asia/Bangkok"
	if loc.String() != "Asia/Bangkok" {
		t.Errorf("Expected location name to be 'Asia/Bangkok', but got '%s'", loc.String())
	}

	// Verify that the location works correctly by converting the current time
	now := time.Now()
	bkkTime := now.In(loc)

	// Compare the location of the time object
	if bkkTime.Location().String() != "Asia/Bangkok" {
		t.Errorf("Expected time location to be 'Asia/Bangkok', but got '%s'", bkkTime.Location().String())
	}
}
