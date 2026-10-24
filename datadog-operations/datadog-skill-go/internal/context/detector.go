package context

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
)

// ServiceContext represents detected service context from git and environment
type ServiceContext struct {
	ServiceName      string
	Repository       string
	CurrentBranch    string
	LastCommitSHA    string
	LastCommitTime   time.Time
	Environment      string
	DetectionMethod  string
	Confidence       float64
}

// DetectContext detects service context from git repository and environment
// It uses multiple detection strategies and returns the highest-confidence result
func DetectContext(workingDir string) (*ServiceContext, error) {
	absDir, err := filepath.Abs(workingDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve working directory: %w", err)
	}

	// Try multiple detection strategies in order of confidence
	strategies := []func(string) (*ServiceContext, error){
		detectFromGitRemote,
		detectFromDirectoryName,
	}

	var bestContext *ServiceContext
	bestConfidence := 0.0

	for _, strategy := range strategies {
		ctx, err := strategy(absDir)
		if err == nil && ctx != nil && ctx.Confidence > bestConfidence {
			bestContext = ctx
			bestConfidence = ctx.Confidence
		}
	}

	if bestContext == nil {
		return nil, fmt.Errorf("failed to detect service context")
	}

	// Enhance with git metadata
	if err := addGitMetadata(absDir, bestContext); err != nil {
		// Log error but don't fail - we have at least basic context
		fmt.Fprintf(os.Stderr, "Warning: failed to add git metadata: %v\n", err)
	}

	// Infer environment from branch
	bestContext.Environment = inferEnvironment(bestContext.CurrentBranch)

	return bestContext, nil
}

// detectFromGitRemote detects service name from git remote URL (90% confidence)
func detectFromGitRemote(dir string) (*ServiceContext, error) {
	remoteURL, err := getGitRemote(dir)
	if err != nil {
		return nil, err
	}

	serviceName := parseServiceFromRemote(remoteURL)
	if serviceName == "" {
		return nil, fmt.Errorf("failed to parse service name from remote URL")
	}

	return &ServiceContext{
		ServiceName:     serviceName,
		Repository:      remoteURL,
		DetectionMethod: "git_remote",
		Confidence:      0.9,
	}, nil
}

// detectFromDirectoryName detects service name from directory name (50% confidence)
func detectFromDirectoryName(dir string) (*ServiceContext, error) {
	dirName := filepath.Base(dir)

	// Skip common non-service directory names
	skipNames := map[string]bool{
		"src":      true,
		"app":      true,
		"service":  true,
		"api":      true,
		"backend":  true,
		"frontend": true,
		"web":      true,
	}

	lowerName := strings.ToLower(dirName)
	if skipNames[lowerName] {
		return nil, fmt.Errorf("directory name is too generic")
	}

	return &ServiceContext{
		ServiceName:     lowerName,
		DetectionMethod: "directory_name",
		Confidence:      0.5,
	}, nil
}

// getGitRemote retrieves the git remote URL using go-git
func getGitRemote(dir string) (string, error) {
	repo, err := git.PlainOpenWithOptions(dir, &git.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		return "", fmt.Errorf("failed to open git repository: %w", err)
	}

	remote, err := repo.Remote("origin")
	if err != nil {
		return "", fmt.Errorf("failed to get origin remote: %w", err)
	}

	if len(remote.Config().URLs) == 0 {
		return "", fmt.Errorf("no remote URLs found")
	}

	return remote.Config().URLs[0], nil
}

// parseServiceFromRemote extracts service name from git remote URL
// Handles formats like:
//   - git@github.com:company/repo.git
//   - https://github.com/company/repo.git
//   - https://github.com/company/repo
func parseServiceFromRemote(remoteURL string) string {
	// Remove .git suffix if present
	remoteURL = strings.TrimSuffix(remoteURL, ".git")

	// Patterns to match different git URL formats
	patterns := []*regexp.Regexp{
		// SSH format: git@github.com:org/repo
		regexp.MustCompile(`[:/]([^/]+)/([^/\.]+)$`),
		// HTTPS format: https://github.com/org/repo
		regexp.MustCompile(`/([^/]+)/([^/\.]+)$`),
	}

	for _, pattern := range patterns {
		matches := pattern.FindStringSubmatch(remoteURL)
		if len(matches) >= 3 {
			// Return the repository name (last component)
			return strings.ToLower(matches[2])
		}
	}

	return ""
}

// getGitBranch retrieves the current git branch
func getGitBranch(dir string) (string, error) {
	repo, err := git.PlainOpenWithOptions(dir, &git.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		return "", fmt.Errorf("failed to open git repository: %w", err)
	}

	head, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD reference: %w", err)
	}

	// Extract branch name from reference
	// e.g., refs/heads/main -> main
	if head.Name().IsBranch() {
		return head.Name().Short(), nil
	}

	// If in detached HEAD state, return the short commit SHA
	return head.Hash().String()[:7], nil
}

// getLastCommit retrieves the last commit SHA and time
func getLastCommit(dir string) (sha string, commitTime time.Time, err error) {
	repo, err := git.PlainOpenWithOptions(dir, &git.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to open git repository: %w", err)
	}

	head, err := repo.Head()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to get HEAD reference: %w", err)
	}

	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to get commit object: %w", err)
	}

	// Return short SHA (7 characters) and commit time
	return head.Hash().String()[:7], commit.Committer.When, nil
}

// addGitMetadata enhances context with git branch, commit SHA, and commit time
func addGitMetadata(dir string, ctx *ServiceContext) error {
	// Get current branch
	branch, err := getGitBranch(dir)
	if err == nil {
		ctx.CurrentBranch = branch
	}

	// Get last commit SHA and time
	sha, commitTime, err := getLastCommit(dir)
	if err == nil {
		ctx.LastCommitSHA = sha
		ctx.LastCommitTime = commitTime
	}

	return nil
}

// inferEnvironment maps branch name to environment
func inferEnvironment(branch string) string {
	// Check environment variables first
	if env := os.Getenv("DD_ENV"); env != "" {
		return strings.ToLower(env)
	}
	if env := os.Getenv("ENVIRONMENT"); env != "" {
		return strings.ToLower(env)
	}

	// If no branch information, default to unknown
	if branch == "" {
		return "unknown"
	}

	branchLower := strings.ToLower(branch)

	// Map branch names to environments
	switch {
	case strings.Contains(branchLower, "main") || strings.Contains(branchLower, "master"):
		return "production"
	case strings.Contains(branchLower, "staging") || strings.Contains(branchLower, "stage"):
		return "staging"
	case strings.Contains(branchLower, "dev") || strings.Contains(branchLower, "develop"):
		return "development"
	default:
		return "development"
	}
}
