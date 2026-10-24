package commands

import (
	"encoding/json"
	"flag"
	"fmt"

	"github.com/datadog/skill/internal/client"
	"github.com/datadog/skill/internal/observability"
)

// NotebooksCommand manages Datadog notebooks for documentation and investigation
type NotebooksCommand struct {
	flags      *flag.FlagSet
	action     string
	notebookID string
	name       string
	query      string
	author     string
	jsonOut    bool
}

// Notebook represents a parsed notebook from Datadog API
type Notebook struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Author   string `json:"author,omitempty"`
	Created  string `json:"created,omitempty"`
	Modified string `json:"modified,omitempty"`
	Cells    int    `json:"cells,omitempty"`
}

// NotebooksResponse represents the formatted notebooks response
type NotebooksResponse struct {
	Status         string     `json:"status"`
	TotalNotebooks int        `json:"total_notebooks"`
	Notebooks      []Notebook `json:"notebooks,omitempty"`
}

// NewNotebooksCommand creates a new notebooks command
func NewNotebooksCommand() *NotebooksCommand {
	cmd := &NotebooksCommand{
		flags: flag.NewFlagSet("notebooks", flag.ExitOnError),
	}

	cmd.flags.StringVar(&cmd.action, "action", "list", "Action: list, create, get, update, delete")
	cmd.flags.StringVar(&cmd.notebookID, "notebook-id", "", "Notebook ID (for get/update/delete)")
	cmd.flags.StringVar(&cmd.name, "name", "", "Notebook name")
	cmd.flags.StringVar(&cmd.query, "query", "", "Search query for notebooks")
	cmd.flags.StringVar(&cmd.author, "author", "", "Filter by author handle")
	cmd.flags.BoolVar(&cmd.jsonOut, "json", false, "Output as JSON")

	return cmd
}

// Name returns the command name
func (c *NotebooksCommand) Name() string {
	return "notebooks"
}

// Description returns the command description
func (c *NotebooksCommand) Description() string {
	return "Manage Datadog notebooks for documentation and investigation"
}

// Run executes the notebooks command
func (c *NotebooksCommand) Run(args []string) error {
	// Initialize observability
	obs, err := observability.Init("manage-notebooks", "production")
	if err != nil {
		return fmt.Errorf("failed to init observability: %w", err)
	}
	defer obs.Shutdown(0)

	// Parse flags
	if err := c.flags.Parse(args); err != nil {
		return err
	}

	// Start tracing
	span := obs.StartSpan("notebooks.manage")
	defer obs.FinishSpan(span)

	obs.LogInfo(fmt.Sprintf("Managing notebooks with action: %s", c.action))

	// Initialize Datadog client
	ddClient, err := client.NewClient()
	if err != nil {
		return fmt.Errorf("failed to initialize Datadog client: %w", err)
	}

	// Execute action
	switch c.action {
	case "list", "list-notebooks":
		return c.listNotebooks(ddClient, obs)
	case "get", "get-notebook":
		return c.getNotebook(ddClient, obs)
	case "delete", "delete-notebook":
		return c.deleteNotebook(ddClient, obs)
	default:
		return fmt.Errorf("unknown action: %s (use: list, get, delete)", c.action)
	}
}

// listNotebooks lists all notebooks
func (c *NotebooksCommand) listNotebooks(ddClient *client.Client, obs *observability.Observability) error {
	obs.LogInfo("Listing notebooks")

	// Query notebooks
	data, err := ddClient.ListNotebooks(c.query, c.author)
	if err != nil {
		return fmt.Errorf("failed to list notebooks: %w", err)
	}

	// Parse and display results
	return c.parseAndDisplayList(data, obs)
}

// getNotebook gets a specific notebook
func (c *NotebooksCommand) getNotebook(ddClient *client.Client, obs *observability.Observability) error {
	if c.notebookID == "" {
		return fmt.Errorf("--notebook-id is required for get action")
	}

	obs.LogInfo(fmt.Sprintf("Getting notebook: %s", c.notebookID))

	// Get notebook
	data, err := ddClient.GetNotebook(c.notebookID)
	if err != nil {
		return fmt.Errorf("failed to get notebook: %w", err)
	}

	// Parse and display result
	return c.parseAndDisplaySingle(data, obs)
}

// deleteNotebook deletes a notebook
func (c *NotebooksCommand) deleteNotebook(ddClient *client.Client, obs *observability.Observability) error {
	if c.notebookID == "" {
		return fmt.Errorf("--notebook-id is required for delete action")
	}

	obs.LogInfo(fmt.Sprintf("Deleting notebook: %s", c.notebookID))

	// Delete notebook
	err := ddClient.DeleteNotebook(c.notebookID)
	if err != nil {
		return fmt.Errorf("failed to delete notebook: %w", err)
	}

	fmt.Printf("✓ Notebook %s deleted successfully\n", c.notebookID)
	return nil
}

// parseAndDisplayList parses and displays list of notebooks
func (c *NotebooksCommand) parseAndDisplayList(data []byte, obs *observability.Observability) error {
	// Parse JSON response
	var apiResponse struct {
		Data []struct {
			ID         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				Name     string `json:"name"`
				Author   struct {
					Handle string `json:"handle"`
				} `json:"author"`
				Created  string `json:"created"`
				Modified string `json:"modified"`
				Cells    []interface{} `json:"cells"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &apiResponse); err != nil {
		return fmt.Errorf("failed to parse API response: %w", err)
	}

	// Build response
	response := NotebooksResponse{
		Status:         "success",
		TotalNotebooks: len(apiResponse.Data),
		Notebooks:      make([]Notebook, 0),
	}

	// Parse notebooks
	for _, item := range apiResponse.Data {
		notebook := Notebook{
			ID:       item.ID,
			Name:     item.Attributes.Name,
			Author:   item.Attributes.Author.Handle,
			Created:  item.Attributes.Created,
			Modified: item.Attributes.Modified,
			Cells:    len(item.Attributes.Cells),
		}

		response.Notebooks = append(response.Notebooks, notebook)
	}

	// Output results
	if c.jsonOut {
		output, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(output))
	} else {
		c.displayFormattedList(response)
	}

	return nil
}

// parseAndDisplaySingle parses and displays a single notebook
func (c *NotebooksCommand) parseAndDisplaySingle(data []byte, obs *observability.Observability) error {
	// Parse JSON response
	var apiResponse struct {
		Data struct {
			ID         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				Name     string `json:"name"`
				Author   struct {
					Handle string `json:"handle"`
				} `json:"author"`
				Created  string `json:"created"`
				Modified string `json:"modified"`
				Cells    []interface{} `json:"cells"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &apiResponse); err != nil {
		return fmt.Errorf("failed to parse API response: %w", err)
	}

	// Build notebook object
	notebook := Notebook{
		ID:       apiResponse.Data.ID,
		Name:     apiResponse.Data.Attributes.Name,
		Author:   apiResponse.Data.Attributes.Author.Handle,
		Created:  apiResponse.Data.Attributes.Created,
		Modified: apiResponse.Data.Attributes.Modified,
		Cells:    len(apiResponse.Data.Attributes.Cells),
	}

	// Output result
	if c.jsonOut {
		output, err := json.MarshalIndent(notebook, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(output))
	} else {
		c.displayFormattedSingle(notebook)
	}

	return nil
}

// displayFormattedList displays formatted list output
func (c *NotebooksCommand) displayFormattedList(response NotebooksResponse) {
	fmt.Println("Notebooks Summary")
	fmt.Println("=================")
	fmt.Printf("Total notebooks: %d\n", response.TotalNotebooks)

	if len(response.Notebooks) > 0 {
		fmt.Println("\nNotebooks:")
		for i, nb := range response.Notebooks {
			if i >= 20 {
				fmt.Printf("\n... and %d more (use --json for full list)\n", len(response.Notebooks)-20)
				break
			}

			fmt.Printf("\n[%s]\n", nb.ID)
			fmt.Printf("  Name: %s\n", nb.Name)
			if nb.Author != "" {
				fmt.Printf("  Author: %s\n", nb.Author)
			}
			fmt.Printf("  Cells: %d\n", nb.Cells)
			if nb.Modified != "" {
				fmt.Printf("  Modified: %s\n", nb.Modified)
			}
		}
	} else {
		fmt.Println("\nNo notebooks found.")
	}
}

// displayFormattedSingle displays formatted single notebook output
func (c *NotebooksCommand) displayFormattedSingle(notebook Notebook) {
	fmt.Println("Notebook Details")
	fmt.Println("================")
	fmt.Printf("ID: %s\n", notebook.ID)
	fmt.Printf("Name: %s\n", notebook.Name)

	if notebook.Author != "" {
		fmt.Printf("Author: %s\n", notebook.Author)
	}

	fmt.Printf("Cells: %d\n", notebook.Cells)

	if notebook.Created != "" {
		fmt.Printf("Created: %s\n", notebook.Created)
	}

	if notebook.Modified != "" {
		fmt.Printf("Modified: %s\n", notebook.Modified)
	}
}

// Help displays help information
func (c *NotebooksCommand) Help() {
	help := `dd notebooks - Manage Datadog Notebooks

DESCRIPTION:
  Manage Datadog notebooks for documentation, investigation, and collaboration.
  Notebooks combine graphs, logs, and text for comprehensive analysis.

USAGE:
  dd notebooks --action <action> [options]

ACTIONS:
  list              List all notebooks
  get               Get a specific notebook
  delete            Delete a notebook

EXAMPLES:
  # List all notebooks
  dd notebooks --action list

  # Search notebooks by query
  dd notebooks --action list --query "incident"

  # Filter by author
  dd notebooks --action list --author "user@example.com"

  # Get specific notebook
  dd notebooks --action get --notebook-id abc123-def456

  # Delete notebook
  dd notebooks --action delete --notebook-id abc123-def456

  # Get JSON output
  dd notebooks --action list --json

OPTIONS:
  --action          Action to perform (list, get, delete)
  --notebook-id     Notebook ID (required for get/delete)
  --name            Notebook name
  --query           Search query for notebooks
  --author          Filter by author handle
  --json            Output as JSON

NOTES:
  - Notebooks combine graphs, logs, and markdown for documentation
  - Use for incident investigations, runbooks, and reporting
  - Create/update operations available via Datadog UI
  - Search supports keyword matching in notebook names

For more information: https://docs.datadoghq.com/api/latest/notebooks/
`
	fmt.Println(help)
}
