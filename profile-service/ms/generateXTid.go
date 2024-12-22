package ms

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

func GenerateXTid(nodeName string) string {
	if len(nodeName) > 5 {
		nodeName = nodeName[:5]
	}

	now := time.Now()
	date := now.Format("060102") // Equivalent to "yymmdd" in Go's time format

	// Base xTid with node name and date
	xTid := fmt.Sprintf("%s-%s", nodeName, date)

	// Calculate remaining length
	remainingLength := 22 - len(xTid)

	// Generate a random string of the required length
	randomString := generateRandomString(remainingLength)

	// Append the random string to xTid
	xTid += randomString

	return xTid
}

// Helper function to generate a random string of a specific length
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	var sb strings.Builder
	for i := 0; i < length; i++ {
		sb.WriteByte(charset[seededRand.Intn(len(charset))])
	}
	return sb.String()
}

 
