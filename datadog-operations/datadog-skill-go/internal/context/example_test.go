package context_test

import (
	"fmt"
	"log"

	"github.com/datadog/skill/internal/context"
)

// ExampleDetectContext demonstrates how to use the context detector
func ExampleDetectContext() {
	// Detect context from current working directory
	ctx, err := context.DetectContext(".")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Service Name: %s\n", ctx.ServiceName)
	fmt.Printf("Repository: %s\n", ctx.Repository)
	fmt.Printf("Branch: %s\n", ctx.CurrentBranch)
	fmt.Printf("Last Commit: %s\n", ctx.LastCommitSHA)
	fmt.Printf("Environment: %s\n", ctx.Environment)
	fmt.Printf("Detection Method: %s\n", ctx.DetectionMethod)
	fmt.Printf("Confidence: %.0f%%\n", ctx.Confidence*100)
}
