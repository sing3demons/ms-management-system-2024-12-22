package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func init() {
	if os.Getenv("SERVICE_NAME") == "" {
		file, err := os.Open("go.mod")
		if err != nil {
			fmt.Printf("Error opening go.mod: %v\n", err)
			return
		}
		defer file.Close()

		var projectName string
		// Scan the file line by line
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			// Check for the "module" directive
			if strings.HasPrefix(line, "module") {
				// Extract the module name
				moduleName := strings.TrimSpace(strings.TrimPrefix(line, "module"))
				// Get the last part of the module name
				projectName = filepath.Base(moduleName)
				fmt.Println("Project Name:", projectName)
			}
		}

		// Handle any scanning errors
		if err := scanner.Err(); err != nil {
			fmt.Printf("Error reading go.mod: %v\n", err)
		}

		os.Setenv("SERVICE_NAME", projectName)
	}
}

func main() {
	servers := "localhost:29092"
	groupID := "example-group"
	ms := NewMicroservice(servers, groupID)
	err := ms.Consume("service.register", func(ctx IContext) error {

		data := map[string]any{}
		json.Unmarshal([]byte(ctx.ReadInput()), &data)

		ctx.Log("Processing", data)
		return nil
	})
	if err != nil {
		fmt.Println("Error:", err)
	}

	ms.Start()
}
