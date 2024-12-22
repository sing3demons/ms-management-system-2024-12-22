package utils

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
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
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	var sb strings.Builder
	for i := 0; i < length; i++ {
		sb.WriteByte(alphabet[seededRand.Intn(len(alphabet))])
	}
	return sb.String()
}

func IsEmail(email string) bool {
	// Define a regex pattern for email validation
	regex := `^[a-zA-Z0-9._%+-]+@([a-zA-Z0-9-]+\.)+[a-zA-Z]{1,}$`

	// Compile the regex
	re := regexp.MustCompile(regex)

	// Match the email against the regex
	return re.MatchString(email)
}

func IsStructEmpty(s interface{}) bool {
	v := reflect.ValueOf(s)
	if v.Kind() == reflect.Struct {
		for i := 0; i < v.NumField(); i++ {
			if !isZeroValue(v.Field(i)) {
				return false
			}
		}
	}
	return true
}

func isZeroValue(value reflect.Value) bool {
	return value.Interface() == reflect.Zero(value.Type()).Interface()
}

// Helper function to check if a string starts with a prefix
func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// Helper function to check if the string contains the date part
func containsDate(xTid, date string) bool {
	return len(xTid) >= len(date) && xTid[6:12] == date
}

func ProjectName() string {
	if projectName := getEnvProjectName(); projectName != "" {
		return projectName
	}

	projectName, err := getModuleNameFromGoMod()
	if err != nil {
		// fmt.Printf("Error reading go.mod: %v\n", err)
		return ""
	}

	return projectName
}

func getEnvProjectName() string {
	if serviceName := os.Getenv("SERVICE_NAME"); serviceName != "" {
		return serviceName
	}
	if projectName := os.Getenv("PROJECT_NAME"); projectName != "" {
		return projectName
	}
	return ""
}

func getModuleNameFromGoMod() (string, error) {
	file, err := os.Open("go.mod")
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "module") {
			moduleName := strings.TrimSpace(strings.TrimPrefix(line, "module"))
			return filepath.Base(moduleName), nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", nil
}
