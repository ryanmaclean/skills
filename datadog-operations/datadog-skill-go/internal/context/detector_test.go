package context

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// setupGitRepo creates a temporary git repository for testing
func setupGitRepo(t *testing.T, remoteURL string, branchName string) string {
	t.Helper()

	tempDir := t.TempDir()

	// Initialize git repository
	repo, err := git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to initialize git repo: %v", err)
	}

	// Add a remote if specified
	if remoteURL != "" {
		_, err = repo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{remoteURL},
		})
		if err != nil {
			t.Fatalf("Failed to create remote: %v", err)
		}
	}

	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Stage the file
	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}

	if _, err := worktree.Add("test.txt"); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	// Create initial commit
	commitHash, err := worktree.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Date(2026, 1, 21, 12, 0, 0, 0, time.UTC),
		},
	})
	if err != nil {
		t.Fatalf("Failed to create commit: %v", err)
	}

	// Create branch if specified
	if branchName != "" && branchName != "master" {
		// Create new branch
		branchRef := plumbing.NewBranchReferenceName(branchName)
		ref := plumbing.NewHashReference(branchRef, commitHash)
		if err := repo.Storer.SetReference(ref); err != nil {
			t.Fatalf("Failed to create branch: %v", err)
		}

		// Checkout the branch
		if err := worktree.Checkout(&git.CheckoutOptions{
			Branch: branchRef,
			Force:  true,
		}); err != nil {
			t.Fatalf("Failed to checkout branch: %v", err)
		}
	}

	// Verify commit exists
	_, err = repo.CommitObject(commitHash)
	if err != nil {
		t.Fatalf("Failed to verify commit: %v", err)
	}

	return tempDir
}

func TestDetectContext_GitRepoWithRemote(t *testing.T) {
	remoteURL := "git@github.com:datadog/test-service.git"
	tempDir := setupGitRepo(t, remoteURL, "main")

	ctx, err := DetectContext(tempDir)
	if err != nil {
		t.Fatalf("DetectContext failed: %v", err)
	}

	// Verify service name was detected from git remote
	if ctx.ServiceName != "test-service" {
		t.Errorf("Expected service name 'test-service', got '%s'", ctx.ServiceName)
	}

	// Verify detection method
	if ctx.DetectionMethod != "git_remote" {
		t.Errorf("Expected detection method 'git_remote', got '%s'", ctx.DetectionMethod)
	}

	// Verify confidence
	if ctx.Confidence != 0.9 {
		t.Errorf("Expected confidence 0.9, got %.2f", ctx.Confidence)
	}

	// Verify repository URL
	if ctx.Repository != remoteURL {
		t.Errorf("Expected repository '%s', got '%s'", remoteURL, ctx.Repository)
	}

	// Verify branch
	if ctx.CurrentBranch != "main" {
		t.Errorf("Expected branch 'main', got '%s'", ctx.CurrentBranch)
	}

	// Verify commit SHA is present
	if ctx.LastCommitSHA == "" {
		t.Error("Expected commit SHA to be detected")
	}

	// Verify commit time is set
	if ctx.LastCommitTime.IsZero() {
		t.Error("Expected commit time to be set")
	}

	// Verify environment inference from branch
	if ctx.Environment != "production" {
		t.Errorf("Expected environment 'production' for main branch, got '%s'", ctx.Environment)
	}
}

func TestDetectContext_GitRepoWithoutRemote(t *testing.T) {
	tempDir := setupGitRepo(t, "", "develop")

	ctx, err := DetectContext(tempDir)
	if err != nil {
		t.Fatalf("DetectContext failed: %v", err)
	}

	// Should fall back to directory name
	if ctx.ServiceName == "" {
		t.Error("Expected service name to be detected from directory")
	}

	// Verify detection method
	if ctx.DetectionMethod != "directory_name" {
		t.Errorf("Expected detection method 'directory_name', got '%s'", ctx.DetectionMethod)
	}

	// Verify lower confidence
	if ctx.Confidence != 0.5 {
		t.Errorf("Expected confidence 0.5, got %.2f", ctx.Confidence)
	}

	// Verify branch
	if ctx.CurrentBranch != "develop" {
		t.Errorf("Expected branch 'develop', got '%s'", ctx.CurrentBranch)
	}

	// Verify environment inference
	if ctx.Environment != "development" {
		t.Errorf("Expected environment 'development' for develop branch, got '%s'", ctx.Environment)
	}
}

func TestDetectContext_NonGitDirectory(t *testing.T) {
	tempDir := t.TempDir()

	// Create a directory with a specific name
	serviceDir := filepath.Join(tempDir, "my-service")
	if err := os.Mkdir(serviceDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	ctx, err := DetectContext(serviceDir)
	if err != nil {
		t.Fatalf("DetectContext failed: %v", err)
	}

	// Should use directory name
	if ctx.ServiceName != "my-service" {
		t.Errorf("Expected service name 'my-service', got '%s'", ctx.ServiceName)
	}

	// Verify detection method
	if ctx.DetectionMethod != "directory_name" {
		t.Errorf("Expected detection method 'directory_name', got '%s'", ctx.DetectionMethod)
	}

	// Verify confidence
	if ctx.Confidence != 0.5 {
		t.Errorf("Expected confidence 0.5, got %.2f", ctx.Confidence)
	}

	// No git metadata should be present
	if ctx.CurrentBranch != "" {
		t.Errorf("Expected no branch, got '%s'", ctx.CurrentBranch)
	}

	if ctx.LastCommitSHA != "" {
		t.Errorf("Expected no commit SHA, got '%s'", ctx.LastCommitSHA)
	}

	// Environment should be unknown without branch
	if ctx.Environment != "unknown" {
		t.Errorf("Expected environment 'unknown', got '%s'", ctx.Environment)
	}
}

func TestDetectContext_GenericDirectoryName(t *testing.T) {
	tempDir := t.TempDir()

	// Create directory with generic name
	genericDir := filepath.Join(tempDir, "src")
	if err := os.Mkdir(genericDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	_, err := DetectContext(genericDir)
	if err == nil {
		t.Error("Expected error for generic directory name, got nil")
	}

	if !strings.Contains(err.Error(), "failed to detect service context") {
		t.Errorf("Expected 'failed to detect service context' error, got: %v", err)
	}
}

func TestParseServiceFromRemote(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "GitHub SSH",
			url:      "git@github.com:datadog/my-service.git",
			expected: "my-service",
		},
		{
			name:     "GitHub SSH without .git",
			url:      "git@github.com:datadog/my-service",
			expected: "my-service",
		},
		{
			name:     "GitHub HTTPS",
			url:      "https://github.com/datadog/my-service.git",
			expected: "my-service",
		},
		{
			name:     "GitHub HTTPS without .git",
			url:      "https://github.com/datadog/my-service",
			expected: "my-service",
		},
		{
			name:     "GitLab SSH",
			url:      "git@gitlab.com:company/team/repo-name.git",
			expected: "repo-name",
		},
		{
			name:     "GitLab HTTPS",
			url:      "https://gitlab.com/company/team/repo-name.git",
			expected: "repo-name",
		},
		{
			name:     "Bitbucket SSH",
			url:      "git@bitbucket.org:company/service-name.git",
			expected: "service-name",
		},
		{
			name:     "Bitbucket HTTPS",
			url:      "https://bitbucket.org/company/service-name.git",
			expected: "service-name",
		},
		{
			name:     "Azure DevOps HTTPS",
			url:      "https://dev.azure.com/org/project/_git/repo-name",
			expected: "repo-name",
		},
		{
			name:     "Mixed case repository",
			url:      "git@github.com:datadog/My-Service.git",
			expected: "my-service",
		},
		{
			name:     "Invalid URL",
			url:      "invalid-url",
			expected: "",
		},
		{
			name:     "Empty URL",
			url:      "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseServiceFromRemote(tt.url)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestGetGitBranch(t *testing.T) {
	tests := []struct {
		name         string
		branchName   string
		expectedName string
	}{
		{
			name:         "main branch",
			branchName:   "main",
			expectedName: "main",
		},
		{
			name:         "develop branch",
			branchName:   "develop",
			expectedName: "develop",
		},
		{
			name:         "feature branch",
			branchName:   "feature/new-feature",
			expectedName: "feature/new-feature",
		},
		{
			name:         "staging branch",
			branchName:   "staging",
			expectedName: "staging",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := setupGitRepo(t, "", tt.branchName)

			branch, err := getGitBranch(tempDir)
			if err != nil {
				t.Fatalf("getGitBranch failed: %v", err)
			}

			if branch != tt.expectedName {
				t.Errorf("Expected branch '%s', got '%s'", tt.expectedName, branch)
			}
		})
	}
}

func TestGetGitBranch_NonGitDirectory(t *testing.T) {
	tempDir := t.TempDir()

	_, err := getGitBranch(tempDir)
	if err == nil {
		t.Error("Expected error for non-git directory, got nil")
	}

	if !strings.Contains(err.Error(), "failed to open git repository") {
		t.Errorf("Expected 'failed to open git repository' error, got: %v", err)
	}
}

func TestGetLastCommit(t *testing.T) {
	tempDir := setupGitRepo(t, "", "main")

	sha, commitTime, err := getLastCommit(tempDir)
	if err != nil {
		t.Fatalf("getLastCommit failed: %v", err)
	}

	// Verify SHA format (7 characters)
	if len(sha) != 7 {
		t.Errorf("Expected SHA length 7, got %d", len(sha))
	}

	// Verify commit time is set
	if commitTime.IsZero() {
		t.Error("Expected commit time to be set")
	}

	// Verify commit time matches our test commit
	expectedTime := time.Date(2026, 1, 21, 12, 0, 0, 0, time.UTC)
	if !commitTime.Equal(expectedTime) {
		t.Errorf("Expected commit time %v, got %v", expectedTime, commitTime)
	}
}

func TestGetLastCommit_NonGitDirectory(t *testing.T) {
	tempDir := t.TempDir()

	_, _, err := getLastCommit(tempDir)
	if err == nil {
		t.Error("Expected error for non-git directory, got nil")
	}

	if !strings.Contains(err.Error(), "failed to open git repository") {
		t.Errorf("Expected 'failed to open git repository' error, got: %v", err)
	}
}

func TestInferEnvironment_FromBranch(t *testing.T) {
	tests := []struct {
		name        string
		branch      string
		environment string
	}{
		{
			name:        "main branch",
			branch:      "main",
			environment: "production",
		},
		{
			name:        "master branch",
			branch:      "master",
			environment: "production",
		},
		{
			name:        "Main with capital",
			branch:      "Main",
			environment: "production",
		},
		{
			name:        "staging branch",
			branch:      "staging",
			environment: "staging",
		},
		{
			name:        "stage branch",
			branch:      "stage",
			environment: "staging",
		},
		{
			name:        "develop branch",
			branch:      "develop",
			environment: "development",
		},
		{
			name:        "dev branch",
			branch:      "dev",
			environment: "development",
		},
		{
			name:        "development branch",
			branch:      "development",
			environment: "development",
		},
		{
			name:        "feature branch",
			branch:      "feature/new-feature",
			environment: "development",
		},
		{
			name:        "bugfix branch",
			branch:      "bugfix/fix-123",
			environment: "development",
		},
		{
			name:        "hotfix branch",
			branch:      "hotfix/urgent-fix",
			environment: "development",
		},
		{
			name:        "empty branch",
			branch:      "",
			environment: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Ensure no environment variables are set
			os.Unsetenv("DD_ENV")
			os.Unsetenv("ENVIRONMENT")

			result := inferEnvironment(tt.branch)
			if result != tt.environment {
				t.Errorf("Expected environment '%s', got '%s'", tt.environment, result)
			}
		})
	}
}

func TestInferEnvironment_FromEnvVariable(t *testing.T) {
	tests := []struct {
		name        string
		envVar      string
		envValue    string
		branch      string
		expected    string
	}{
		{
			name:     "DD_ENV takes precedence",
			envVar:   "DD_ENV",
			envValue: "custom-env",
			branch:   "main",
			expected: "custom-env",
		},
		{
			name:     "ENVIRONMENT variable",
			envVar:   "ENVIRONMENT",
			envValue: "qa",
			branch:   "main",
			expected: "qa",
		},
		{
			name:     "DD_ENV with uppercase",
			envVar:   "DD_ENV",
			envValue: "PRODUCTION",
			branch:   "develop",
			expected: "production",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment variables first
			os.Unsetenv("DD_ENV")
			os.Unsetenv("ENVIRONMENT")

			// Set the test environment variable
			os.Setenv(tt.envVar, tt.envValue)
			defer os.Unsetenv(tt.envVar)

			result := inferEnvironment(tt.branch)
			if result != tt.expected {
				t.Errorf("Expected environment '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestDetectFromGitRemote(t *testing.T) {
	remoteURL := "git@github.com:datadog/my-api.git"
	tempDir := setupGitRepo(t, remoteURL, "main")

	ctx, err := detectFromGitRemote(tempDir)
	if err != nil {
		t.Fatalf("detectFromGitRemote failed: %v", err)
	}

	if ctx.ServiceName != "my-api" {
		t.Errorf("Expected service name 'my-api', got '%s'", ctx.ServiceName)
	}

	if ctx.Repository != remoteURL {
		t.Errorf("Expected repository '%s', got '%s'", remoteURL, ctx.Repository)
	}

	if ctx.DetectionMethod != "git_remote" {
		t.Errorf("Expected detection method 'git_remote', got '%s'", ctx.DetectionMethod)
	}

	if ctx.Confidence != 0.9 {
		t.Errorf("Expected confidence 0.9, got %.2f", ctx.Confidence)
	}
}

func TestDetectFromGitRemote_NoRemote(t *testing.T) {
	tempDir := setupGitRepo(t, "", "main")

	_, err := detectFromGitRemote(tempDir)
	if err == nil {
		t.Error("Expected error when no remote is configured")
	}

	if !strings.Contains(err.Error(), "failed to get origin remote") {
		t.Errorf("Expected 'failed to get origin remote' error, got: %v", err)
	}
}

func TestDetectFromGitRemote_NonGitDirectory(t *testing.T) {
	tempDir := t.TempDir()

	_, err := detectFromGitRemote(tempDir)
	if err == nil {
		t.Error("Expected error for non-git directory")
	}

	if !strings.Contains(err.Error(), "failed to open git repository") {
		t.Errorf("Expected 'failed to open git repository' error, got: %v", err)
	}
}

func TestDetectFromDirectoryName(t *testing.T) {
	tests := []struct {
		name        string
		dirName     string
		shouldFail  bool
		expected    string
		confidence  float64
	}{
		{
			name:       "valid service name",
			dirName:    "my-service",
			shouldFail: false,
			expected:   "my-service",
			confidence: 0.5,
		},
		{
			name:       "valid service with uppercase",
			dirName:    "My-Service",
			shouldFail: false,
			expected:   "my-service",
			confidence: 0.5,
		},
		{
			name:       "generic name: src",
			dirName:    "src",
			shouldFail: true,
		},
		{
			name:       "generic name: app",
			dirName:    "app",
			shouldFail: true,
		},
		{
			name:       "generic name: service",
			dirName:    "service",
			shouldFail: true,
		},
		{
			name:       "generic name: api",
			dirName:    "api",
			shouldFail: true,
		},
		{
			name:       "generic name: backend",
			dirName:    "backend",
			shouldFail: true,
		},
		{
			name:       "generic name: frontend",
			dirName:    "frontend",
			shouldFail: true,
		},
		{
			name:       "generic name: web",
			dirName:    "web",
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			testDir := filepath.Join(tempDir, tt.dirName)
			if err := os.Mkdir(testDir, 0755); err != nil {
				t.Fatalf("Failed to create directory: %v", err)
			}

			ctx, err := detectFromDirectoryName(testDir)

			if tt.shouldFail {
				if err == nil {
					t.Error("Expected error for generic directory name, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("detectFromDirectoryName failed: %v", err)
			}

			if ctx.ServiceName != tt.expected {
				t.Errorf("Expected service name '%s', got '%s'", tt.expected, ctx.ServiceName)
			}

			if ctx.DetectionMethod != "directory_name" {
				t.Errorf("Expected detection method 'directory_name', got '%s'", ctx.DetectionMethod)
			}

			if ctx.Confidence != tt.confidence {
				t.Errorf("Expected confidence %.2f, got %.2f", tt.confidence, ctx.Confidence)
			}
		})
	}
}

func TestAddGitMetadata(t *testing.T) {
	tempDir := setupGitRepo(t, "git@github.com:datadog/test.git", "staging")

	ctx := &ServiceContext{
		ServiceName:     "test",
		DetectionMethod: "git_remote",
		Confidence:      0.9,
	}

	err := addGitMetadata(tempDir, ctx)
	if err != nil {
		t.Fatalf("addGitMetadata failed: %v", err)
	}

	// Verify branch was added
	if ctx.CurrentBranch != "staging" {
		t.Errorf("Expected branch 'staging', got '%s'", ctx.CurrentBranch)
	}

	// Verify commit SHA was added
	if ctx.LastCommitSHA == "" {
		t.Error("Expected commit SHA to be set")
	}

	if len(ctx.LastCommitSHA) != 7 {
		t.Errorf("Expected commit SHA length 7, got %d", len(ctx.LastCommitSHA))
	}

	// Verify commit time was added
	if ctx.LastCommitTime.IsZero() {
		t.Error("Expected commit time to be set")
	}
}

func TestAddGitMetadata_NonGitDirectory(t *testing.T) {
	tempDir := t.TempDir()

	ctx := &ServiceContext{
		ServiceName:     "test",
		DetectionMethod: "directory_name",
		Confidence:      0.5,
	}

	// Should not fail, but metadata won't be added
	err := addGitMetadata(tempDir, ctx)

	// The function returns nil even for non-git directories
	// It's designed to be lenient and not fail the overall detection
	if err != nil {
		t.Fatalf("addGitMetadata should not fail for non-git directory: %v", err)
	}

	// Verify no metadata was added
	if ctx.CurrentBranch != "" {
		t.Errorf("Expected no branch for non-git directory, got '%s'", ctx.CurrentBranch)
	}

	if ctx.LastCommitSHA != "" {
		t.Errorf("Expected no commit SHA for non-git directory, got '%s'", ctx.LastCommitSHA)
	}
}

func TestConfidenceScoring(t *testing.T) {
	tests := []struct {
		name               string
		remoteURL          string
		expectedConfidence float64
		expectedMethod     string
	}{
		{
			name:               "git remote detection",
			remoteURL:          "git@github.com:datadog/service.git",
			expectedConfidence: 0.9,
			expectedMethod:     "git_remote",
		},
		{
			name:               "directory name fallback",
			remoteURL:          "",
			expectedConfidence: 0.5,
			expectedMethod:     "directory_name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := setupGitRepo(t, tt.remoteURL, "main")

			ctx, err := DetectContext(tempDir)
			if err != nil {
				t.Fatalf("DetectContext failed: %v", err)
			}

			if ctx.Confidence != tt.expectedConfidence {
				t.Errorf("Expected confidence %.2f, got %.2f", tt.expectedConfidence, ctx.Confidence)
			}

			if ctx.DetectionMethod != tt.expectedMethod {
				t.Errorf("Expected detection method '%s', got '%s'", tt.expectedMethod, ctx.DetectionMethod)
			}
		})
	}
}

func TestDetectContext_InvalidPath(t *testing.T) {
	// Test with a generic directory name that should be rejected
	ctx, err := DetectContext("/nonexistent/path/src")
	if err == nil {
		t.Error("Expected error for generic directory name, got nil")
	}
	if ctx != nil {
		t.Error("Expected nil context for generic directory name")
	}
}

func TestDetectContext_MultipleStrategies(t *testing.T) {
	// Create a git repo with remote to test strategy priority
	tempDir := setupGitRepo(t, "git@github.com:datadog/priority-test.git", "main")

	ctx, err := DetectContext(tempDir)
	if err != nil {
		t.Fatalf("DetectContext failed: %v", err)
	}

	// Should use git_remote (higher confidence) over directory_name
	if ctx.DetectionMethod != "git_remote" {
		t.Errorf("Expected 'git_remote' to be chosen over 'directory_name', got '%s'", ctx.DetectionMethod)
	}

	if ctx.Confidence != 0.9 {
		t.Errorf("Expected highest confidence strategy (0.9), got %.2f", ctx.Confidence)
	}

	if ctx.ServiceName != "priority-test" {
		t.Errorf("Expected service name from git remote 'priority-test', got '%s'", ctx.ServiceName)
	}
}

func TestGetGitRemote(t *testing.T) {
	remoteURL := "https://github.com/datadog/test-repo.git"
	tempDir := setupGitRepo(t, remoteURL, "main")

	result, err := getGitRemote(tempDir)
	if err != nil {
		t.Fatalf("getGitRemote failed: %v", err)
	}

	if result != remoteURL {
		t.Errorf("Expected remote URL '%s', got '%s'", remoteURL, result)
	}
}

func TestGetGitRemote_NoRemote(t *testing.T) {
	tempDir := setupGitRepo(t, "", "main")

	_, err := getGitRemote(tempDir)
	if err == nil {
		t.Error("Expected error when no remote is configured")
	}

	if !strings.Contains(err.Error(), "failed to get origin remote") {
		t.Errorf("Expected 'failed to get origin remote' error, got: %v", err)
	}
}

func TestGetGitRemote_NonGitDirectory(t *testing.T) {
	tempDir := t.TempDir()

	_, err := getGitRemote(tempDir)
	if err == nil {
		t.Error("Expected error for non-git directory")
	}

	if !strings.Contains(err.Error(), "failed to open git repository") {
		t.Errorf("Expected 'failed to open git repository' error, got: %v", err)
	}
}
