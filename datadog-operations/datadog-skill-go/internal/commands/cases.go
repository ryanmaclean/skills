package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/datadog/skill/internal/client"
)

// CasesCommand manages Datadog Case Management for issue tracking and resolution
type CasesCommand struct {
	flags      *flag.FlagSet
	action     string
	projectKey string
	projectID  string
	caseID     string
	title      string
	typeID     string
	priority   string
	status     string
	assignee   string
	description string
	comment    string
	query      string
	jsonOut    bool
}

// NewCasesCommand creates a new cases command
func NewCasesCommand() Command {
	cmd := &CasesCommand{
		flags: flag.NewFlagSet("cases", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.action, "action", "", "Action: projects-list, projects-create, projects-get, projects-delete, list, create, get, assign, unassign, archive, unarchive, update-status, update-priority, comment")
	cmd.flags.StringVar(&cmd.projectKey, "project-key", "", "Project key (unique identifier)")
	cmd.flags.StringVar(&cmd.projectID, "project-id", "", "Project ID")
	cmd.flags.StringVar(&cmd.caseID, "case-id", "", "Case ID")
	cmd.flags.StringVar(&cmd.title, "title", "", "Case title")
	cmd.flags.StringVar(&cmd.typeID, "type-id", "", "Case type ID")
	cmd.flags.StringVar(&cmd.priority, "priority", "P3", "Case priority (P1, P2, P3, P4, P5)")
	cmd.flags.StringVar(&cmd.status, "status", "", "Case status")
	cmd.flags.StringVar(&cmd.assignee, "assignee", "", "Assignee user handle")
	cmd.flags.StringVar(&cmd.description, "description", "", "Case description")
	cmd.flags.StringVar(&cmd.comment, "comment", "", "Comment text")
	cmd.flags.StringVar(&cmd.query, "query", "", "Search query for cases")
	cmd.flags.BoolVar(&cmd.jsonOut, "json", false, "Output in JSON format")

	return cmd
}

// Run executes the cases command
func (c *CasesCommand) Run(args []string) error {
	if err := c.flags.Parse(args); err != nil {
		return err
	}

	if c.action == "" {
		return fmt.Errorf("action is required. Use --help to see available actions")
	}

	ddClient, err := client.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Datadog client: %w", err)
	}

	switch c.action {
	case "projects-list":
		return c.listProjects(ddClient)
	case "projects-create":
		return c.createProject(ddClient)
	case "projects-get":
		return c.getProject(ddClient)
	case "projects-delete":
		return c.deleteProject(ddClient)
	case "list":
		return c.listCases(ddClient)
	case "create":
		return c.createCase(ddClient)
	case "get":
		return c.getCase(ddClient)
	case "assign":
		return c.assignCase(ddClient)
	case "unassign":
		return c.unassignCase(ddClient)
	case "archive":
		return c.archiveCase(ddClient)
	case "unarchive":
		return c.unarchiveCase(ddClient)
	case "update-status":
		return c.updateStatus(ddClient)
	case "update-priority":
		return c.updatePriority(ddClient)
	case "comment":
		return c.addComment(ddClient)
	default:
		return fmt.Errorf("unknown action: %s", c.action)
	}
}

// listProjects retrieves all Case Management projects
func (c *CasesCommand) listProjects(ddClient *client.Client) error {
	resp, err := ddClient.ListCaseProjects()
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	if c.jsonOut {
		fmt.Println(string(resp))
		return nil
	}

	var result struct {
		Data []struct {
			ID         string `json:"id"`
			Attributes struct {
				Key  string `json:"key"`
				Name string `json:"name"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Println("Case Management Projects:")
	fmt.Println("========================")
	for _, project := range result.Data {
		fmt.Printf("ID: %s\n", project.ID)
		fmt.Printf("  Key: %s\n", project.Attributes.Key)
		fmt.Printf("  Name: %s\n", project.Attributes.Name)
		fmt.Println()
	}

	return nil
}

// createProject creates a new Case Management project
func (c *CasesCommand) createProject(ddClient *client.Client) error {
	if c.projectKey == "" || c.title == "" {
		return fmt.Errorf("both --project-key and --title are required")
	}

	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "project",
			"attributes": map[string]interface{}{
				"key":  c.projectKey,
				"name": c.title,
			},
		},
	}

	resp, err := ddClient.CreateCaseProject(payload)
	if err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	if c.jsonOut {
		fmt.Println(string(resp))
		return nil
	}

	var result struct {
		Data struct {
			ID         string `json:"id"`
			Attributes struct {
				Key  string `json:"key"`
				Name string `json:"name"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Println("✅ Project Created Successfully")
	fmt.Printf("ID: %s\n", result.Data.ID)
	fmt.Printf("Key: %s\n", result.Data.Attributes.Key)
	fmt.Printf("Name: %s\n", result.Data.Attributes.Name)

	return nil
}

// getProject retrieves a specific project
func (c *CasesCommand) getProject(ddClient *client.Client) error {
	if c.projectID == "" {
		return fmt.Errorf("--project-id is required")
	}

	resp, err := ddClient.GetCaseProject(c.projectID)
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	if c.jsonOut {
		fmt.Println(string(resp))
		return nil
	}

	var result struct {
		Data struct {
			ID         string `json:"id"`
			Attributes struct {
				Key  string `json:"key"`
				Name string `json:"name"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Println("Project Details:")
	fmt.Println("===============")
	fmt.Printf("ID: %s\n", result.Data.ID)
	fmt.Printf("Key: %s\n", result.Data.Attributes.Key)
	fmt.Printf("Name: %s\n", result.Data.Attributes.Name)

	return nil
}

// deleteProject removes a project
func (c *CasesCommand) deleteProject(ddClient *client.Client) error {
	if c.projectID == "" {
		return fmt.Errorf("--project-id is required")
	}

	if err := ddClient.DeleteCaseProject(c.projectID); err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	fmt.Printf("✅ Project %s deleted successfully\n", c.projectID)
	return nil
}

// listCases searches and lists cases
func (c *CasesCommand) listCases(ddClient *client.Client) error {
	payload := map[string]interface{}{}

	if c.query != "" {
		payload["filter[query]"] = c.query
	}

	resp, err := ddClient.SearchCases(payload)
	if err != nil {
		return fmt.Errorf("failed to list cases: %w", err)
	}

	if c.jsonOut {
		fmt.Println(string(resp))
		return nil
	}

	var result struct {
		Data []struct {
			ID         string `json:"id"`
			Attributes struct {
				Title       string    `json:"title"`
				Status      string    `json:"status"`
				Priority    string    `json:"priority"`
				Description string    `json:"description"`
				CreatedAt   time.Time `json:"created_at"`
				ModifiedAt  time.Time `json:"modified_at"`
			} `json:"attributes"`
			Relationships struct {
				Assignee struct {
					Data struct {
						ID string `json:"id"`
					} `json:"data"`
				} `json:"assignee"`
				Project struct {
					Data struct {
						ID string `json:"id"`
					} `json:"data"`
				} `json:"project"`
			} `json:"relationships"`
		} `json:"data"`
		Meta struct {
			Page struct {
				TotalCount int `json:"total_count"`
			} `json:"page"`
		} `json:"meta"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Printf("Cases (Total: %d):\n", result.Meta.Page.TotalCount)
	fmt.Println("==================")

	for _, caseItem := range result.Data {
		fmt.Printf("\nCase ID: %s\n", caseItem.ID)
		fmt.Printf("  Title: %s\n", caseItem.Attributes.Title)
		fmt.Printf("  Status: %s\n", caseItem.Attributes.Status)
		fmt.Printf("  Priority: %s\n", caseItem.Attributes.Priority)
		if caseItem.Relationships.Assignee.Data.ID != "" {
			fmt.Printf("  Assignee: %s\n", caseItem.Relationships.Assignee.Data.ID)
		}
		if caseItem.Relationships.Project.Data.ID != "" {
			fmt.Printf("  Project: %s\n", caseItem.Relationships.Project.Data.ID)
		}
		fmt.Printf("  Created: %s\n", caseItem.Attributes.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Modified: %s\n", caseItem.Attributes.ModifiedAt.Format("2006-01-02 15:04:05"))
		if caseItem.Attributes.Description != "" && len(caseItem.Attributes.Description) > 100 {
			fmt.Printf("  Description: %s...\n", caseItem.Attributes.Description[:100])
		} else if caseItem.Attributes.Description != "" {
			fmt.Printf("  Description: %s\n", caseItem.Attributes.Description)
		}
	}

	return nil
}

// createCase creates a new case
func (c *CasesCommand) createCase(ddClient *client.Client) error {
	if c.title == "" || c.typeID == "" || c.projectID == "" {
		return fmt.Errorf("--title, --type-id, and --project-id are required")
	}

	attributes := map[string]interface{}{
		"title":    c.title,
		"priority": c.priority,
	}

	if c.description != "" {
		attributes["description"] = c.description
	}

	if c.status != "" {
		attributes["status_name"] = c.status
	}

	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type":       "case",
			"attributes": attributes,
			"relationships": map[string]interface{}{
				"type": map[string]interface{}{
					"data": map[string]interface{}{
						"type": "case_type",
						"id":   c.typeID,
					},
				},
				"project": map[string]interface{}{
					"data": map[string]interface{}{
						"type": "project",
						"id":   c.projectID,
					},
				},
			},
		},
	}

	if c.assignee != "" {
		payload["data"].(map[string]interface{})["relationships"].(map[string]interface{})["assignee"] = map[string]interface{}{
			"data": map[string]interface{}{
				"type": "user",
				"id":   c.assignee,
			},
		}
	}

	resp, err := ddClient.CreateCase(payload)
	if err != nil {
		return fmt.Errorf("failed to create case: %w", err)
	}

	if c.jsonOut {
		fmt.Println(string(resp))
		return nil
	}

	var result struct {
		Data struct {
			ID         string `json:"id"`
			Attributes struct {
				Title    string `json:"title"`
				Status   string `json:"status"`
				Priority string `json:"priority"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Println("✅ Case Created Successfully")
	fmt.Printf("Case ID: %s\n", result.Data.ID)
	fmt.Printf("Title: %s\n", result.Data.Attributes.Title)
	fmt.Printf("Status: %s\n", result.Data.Attributes.Status)
	fmt.Printf("Priority: %s\n", result.Data.Attributes.Priority)

	return nil
}

// getCase retrieves case details
func (c *CasesCommand) getCase(ddClient *client.Client) error {
	if c.caseID == "" {
		return fmt.Errorf("--case-id is required")
	}

	resp, err := ddClient.GetCase(c.caseID)
	if err != nil {
		return fmt.Errorf("failed to get case: %w", err)
	}

	if c.jsonOut {
		fmt.Println(string(resp))
		return nil
	}

	var result struct {
		Data struct {
			ID         string `json:"id"`
			Attributes struct {
				Title       string    `json:"title"`
				Status      string    `json:"status"`
				Priority    string    `json:"priority"`
				Description string    `json:"description"`
				CreatedAt   time.Time `json:"created_at"`
				ModifiedAt  time.Time `json:"modified_at"`
			} `json:"attributes"`
			Relationships struct {
				Assignee struct {
					Data struct {
						ID string `json:"id"`
					} `json:"data"`
				} `json:"assignee"`
				Project struct {
					Data struct {
						ID string `json:"id"`
					} `json:"data"`
				} `json:"project"`
			} `json:"relationships"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Println("Case Details:")
	fmt.Println("=============")
	fmt.Printf("Case ID: %s\n", result.Data.ID)
	fmt.Printf("Title: %s\n", result.Data.Attributes.Title)
	fmt.Printf("Status: %s\n", result.Data.Attributes.Status)
	fmt.Printf("Priority: %s\n", result.Data.Attributes.Priority)
	if result.Data.Relationships.Assignee.Data.ID != "" {
		fmt.Printf("Assignee: %s\n", result.Data.Relationships.Assignee.Data.ID)
	}
	if result.Data.Relationships.Project.Data.ID != "" {
		fmt.Printf("Project: %s\n", result.Data.Relationships.Project.Data.ID)
	}
	fmt.Printf("Created: %s\n", result.Data.Attributes.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Modified: %s\n", result.Data.Attributes.ModifiedAt.Format("2006-01-02 15:04:05"))
	if result.Data.Attributes.Description != "" {
		fmt.Printf("\nDescription:\n%s\n", result.Data.Attributes.Description)
	}

	return nil
}

// assignCase assigns a case to a user
func (c *CasesCommand) assignCase(ddClient *client.Client) error {
	if c.caseID == "" || c.assignee == "" {
		return fmt.Errorf("both --case-id and --assignee are required")
	}

	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"attributes": map[string]interface{}{
				"assignee_id": c.assignee,
			},
		},
	}

	resp, err := ddClient.AssignCase(c.caseID, payload)
	if err != nil {
		return fmt.Errorf("failed to assign case: %w", err)
	}

	if c.jsonOut {
		fmt.Println(string(resp))
		return nil
	}

	fmt.Printf("✅ Case %s assigned to %s\n", c.caseID, c.assignee)
	return nil
}

// unassignCase removes assignee from a case
func (c *CasesCommand) unassignCase(ddClient *client.Client) error {
	if c.caseID == "" {
		return fmt.Errorf("--case-id is required")
	}

	resp, err := ddClient.UnassignCase(c.caseID)
	if err != nil {
		return fmt.Errorf("failed to unassign case: %w", err)
	}

	if c.jsonOut {
		fmt.Println(string(resp))
		return nil
	}

	fmt.Printf("✅ Case %s unassigned\n", c.caseID)
	return nil
}

// archiveCase archives a case
func (c *CasesCommand) archiveCase(ddClient *client.Client) error {
	if c.caseID == "" {
		return fmt.Errorf("--case-id is required")
	}

	resp, err := ddClient.ArchiveCase(c.caseID)
	if err != nil {
		return fmt.Errorf("failed to archive case: %w", err)
	}

	if c.jsonOut {
		fmt.Println(string(resp))
		return nil
	}

	fmt.Printf("✅ Case %s archived\n", c.caseID)
	return nil
}

// unarchiveCase unarchives a case
func (c *CasesCommand) unarchiveCase(ddClient *client.Client) error {
	if c.caseID == "" {
		return fmt.Errorf("--case-id is required")
	}

	resp, err := ddClient.UnarchiveCase(c.caseID)
	if err != nil {
		return fmt.Errorf("failed to unarchive case: %w", err)
	}

	if c.jsonOut {
		fmt.Println(string(resp))
		return nil
	}

	fmt.Printf("✅ Case %s unarchived\n", c.caseID)
	return nil
}

// updateStatus updates case status
func (c *CasesCommand) updateStatus(ddClient *client.Client) error {
	if c.caseID == "" || c.status == "" {
		return fmt.Errorf("both --case-id and --status are required")
	}

	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"attributes": map[string]interface{}{
				"status": c.status,
			},
		},
	}

	resp, err := ddClient.UpdateCaseStatus(c.caseID, payload)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	if c.jsonOut {
		fmt.Println(string(resp))
		return nil
	}

	fmt.Printf("✅ Case %s status updated to: %s\n", c.caseID, c.status)
	return nil
}

// updatePriority updates case priority
func (c *CasesCommand) updatePriority(ddClient *client.Client) error {
	if c.caseID == "" || c.priority == "" {
		return fmt.Errorf("both --case-id and --priority are required")
	}

	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"attributes": map[string]interface{}{
				"priority": c.priority,
			},
		},
	}

	resp, err := ddClient.UpdateCasePriority(c.caseID, payload)
	if err != nil {
		return fmt.Errorf("failed to update priority: %w", err)
	}

	if c.jsonOut {
		fmt.Println(string(resp))
		return nil
	}

	fmt.Printf("✅ Case %s priority updated to: %s\n", c.caseID, c.priority)
	return nil
}

// addComment adds a comment to a case
func (c *CasesCommand) addComment(ddClient *client.Client) error {
	if c.caseID == "" || c.comment == "" {
		return fmt.Errorf("both --case-id and --comment are required")
	}

	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"attributes": map[string]interface{}{
				"body": c.comment,
			},
		},
	}

	resp, err := ddClient.AddCaseComment(c.caseID, payload)
	if err != nil {
		return fmt.Errorf("failed to add comment: %w", err)
	}

	if c.jsonOut {
		fmt.Println(string(resp))
		return nil
	}

	fmt.Printf("✅ Comment added to case %s\n", c.caseID)
	return nil
}

// Name returns the command name
func (c *CasesCommand) Name() string {
	return "cases"
}

// Description returns a short description of the command
func (c *CasesCommand) Description() string {
	return "Manage Datadog Case Management for issue tracking and resolution"
}

// Help displays help information
func (c *CasesCommand) Help() {
	help := `dd cases - Manage Datadog Case Management

DESCRIPTION:
  Manage cases and projects for issue tracking and resolution in Datadog.
  Case Management provides structured workflows for handling incidents,
  bugs, and other issues with your team.

USAGE:
  dd cases --action <action> [options]

PROJECT ACTIONS:
  projects-list      List all Case Management projects
  projects-create    Create a new project
  projects-get       Get project details
  projects-delete    Delete a project

CASE ACTIONS:
  list              Search and list cases
  create            Create a new case
  get               Get case details
  assign            Assign case to a user
  unassign          Unassign case
  archive           Archive case
  unarchive         Unarchive case
  update-status     Update case status
  update-priority   Update case priority
  comment           Add comment to case

OPTIONS:
  --action string        Action to perform (required)
  --project-key string   Project key (unique identifier)
  --project-id string    Project ID
  --case-id string       Case ID
  --title string         Case title
  --type-id string       Case type ID (required for create)
  --priority string      Case priority (P1-P5, default: P3)
  --status string        Case status
  --assignee string      Assignee user handle
  --description string   Case description
  --comment string       Comment text
  --query string         Search query for cases
  --json                 Output in JSON format

EXAMPLES:
  # List all projects
  dd cases --action projects-list

  # Create a new project
  dd cases --action projects-create --project-key my-app --title "My Application"

  # List all cases
  dd cases --action list

  # Search cases
  dd cases --action list --query "status:open priority:P1"

  # Create a new case
  dd cases --action create \
    --title "Database performance degradation" \
    --type-id <type-id> \
    --project-id <project-id> \
    --priority P1 \
    --description "Response times increased 3x"

  # Assign case to user
  dd cases --action assign --case-id <case-id> --assignee user@example.com

  # Update case status
  dd cases --action update-status --case-id <case-id> --status "In Progress"

  # Update case priority
  dd cases --action update-priority --case-id <case-id> --priority P1

  # Add comment
  dd cases --action comment --case-id <case-id> --comment "Investigation started"

  # Archive case
  dd cases --action archive --case-id <case-id>

  # Get JSON output
  dd cases --action list --json

CASE PRIORITIES:
  P1 - Critical (immediate attention required)
  P2 - High (resolve within hours)
  P3 - Medium (resolve within days) - default
  P4 - Low (resolve within weeks)
  P5 - Minimal (resolve when convenient)

AUTHENTICATION:
  Requires DD_API_KEY and DD_APP_KEY environment variables.
  OAuth scopes: cases_read, cases_write
`
	fmt.Println(strings.TrimSpace(help))
}
