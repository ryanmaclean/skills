package commands

import (
	"encoding/json"
	"flag"
	"fmt"

	"github.com/datadog/skill/internal/context"
	"github.com/datadog/skill/internal/observability"
)

type ContextCommand struct {
	flags  *flag.FlagSet
	json   bool
	wd     string
}

func NewContextCommand() *ContextCommand {
	cmd := &ContextCommand{
		flags: flag.NewFlagSet("context", flag.ExitOnError),
	}

	cmd.flags.BoolVar(&cmd.json, "json", false, "Output as JSON")
	cmd.flags.StringVar(&cmd.wd, "working-dir", ".", "Working directory")

	return cmd
}

func (c *ContextCommand) Name() string {
	return "context"
}

func (c *ContextCommand) Description() string {
	return "Detect service context from project"
}

func (c *ContextCommand) Run(args []string) error {
	// Initialize observability
	obs, err := observability.Init("dd-cli", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	// Parse flags
	if err := c.flags.Parse(args); err != nil {
		return err
	}

	// Start tracing
	span := obs.StartSpan("context.detect")
	defer obs.FinishSpan(span)

	obs.LogInfo("Detecting service context")

	// Detect context
	ctx, err := context.DetectContext(c.wd)
	if err != nil {
		obs.LogError("Failed to detect context: " + err.Error())
		return fmt.Errorf("failed to detect context: %w", err)
	}

	obs.GetTracer().SetTag(span, "service.name", ctx.ServiceName)
	obs.GetTracer().SetTag(span, "detection.method", ctx.DetectionMethod)
	obs.GetTracer().SetTag(span, "detection.confidence", fmt.Sprintf("%.0f", ctx.Confidence*100))

	// Record metrics
	obs.GetMetrics().Gauge("context.detection.confidence", ctx.Confidence,
		"method:"+ctx.DetectionMethod,
	)

	// Output
	if c.json {
		jsonData, err := json.MarshalIndent(ctx, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		c.printFormatted(ctx)
	}

	obs.LogInfo("Context detection complete")
	return nil
}

func (c *ContextCommand) printFormatted(ctx *context.ServiceContext) {
	fmt.Printf("✓ Detected service: **%s**\n", ctx.ServiceName)
	fmt.Printf("  Method: %s\n", ctx.DetectionMethod)
	fmt.Printf("  Confidence: %.0f%%\n", ctx.Confidence*100)

	if ctx.Repository != "" {
		fmt.Printf("  Repository: %s\n", ctx.Repository)
	}

	if ctx.CurrentBranch != "" {
		fmt.Printf("  Branch: %s\n", ctx.CurrentBranch)
	}

	if ctx.LastCommitSHA != "" {
		fmt.Printf("  Last commit: %s\n", ctx.LastCommitSHA)
	}

	if ctx.Environment != "" {
		fmt.Printf("  Environment: %s\n", ctx.Environment)
	}
}

func (c *ContextCommand) Help() {
	fmt.Println("Usage: dd context [options]")
	fmt.Println()
	fmt.Println("Detect service context from project (git remote, directory name, etc.)")
	fmt.Println()
	fmt.Println("Options:")
	c.flags.PrintDefaults()
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  dd context")
	fmt.Println("  dd context --json")
	fmt.Println("  dd context --working-dir /path/to/project")
}
