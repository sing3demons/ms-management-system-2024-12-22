package utils

import "time"

func LocBkk() *time.Location {
	loc, _ := time.LoadLocation("Asia/Bangkok")

	// Get the current time in Bangkok
	// now := time.Now().In(loc)
	return loc
}
