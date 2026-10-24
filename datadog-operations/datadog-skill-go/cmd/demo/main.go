package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/datadog/skill/internal/context"
)

func main() {
	// Get working directory from args or use current directory
	workingDir := "."
	if len(os.Args) > 1 {
		workingDir = os.Args[1]
	}

	fmt.Printf("Detecting context for: %s\n\n", workingDir)

	// Detect context
	ctx, err := context.DetectContext(workingDir)
	if err != nil {
		log.Fatalf("Failed to detect context: %v", err)
	}

	// Display results
	fmt.Printf("Service Context Detection Results:\n")
	fmt.Printf("===================================\n\n")
	fmt.Printf("Service Name:      %s\n", ctx.ServiceName)
	fmt.Printf("Repository:        %s\n", ctx.Repository)
	fmt.Printf("Current Branch:    %s\n", ctx.CurrentBranch)
	fmt.Printf("Last Commit SHA:   %s\n", ctx.LastCommitSHA)
	fmt.Printf("Last Commit Time:  %s\n", ctx.LastCommitTime.Format("2006-01-02 15:04:05 MST"))
	fmt.Printf("Environment:       %s\n", ctx.Environment)
	fmt.Printf("Detection Method:  %s\n", ctx.DetectionMethod)
	fmt.Printf("Confidence:        %.0f%%\n\n", ctx.Confidence*100)

	// Also output as JSON
	fmt.Printf("JSON Output:\n")
	fmt.Printf("============\n")
	jsonData, err := json.MarshalIndent(ctx, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}
	fmt.Println(string(jsonData))
}
