package commands

// Command interface for all CLI commands
type Command interface {
	Name() string
	Description() string
	Run(args []string) error
	Help()
}
