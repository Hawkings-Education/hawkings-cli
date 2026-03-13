package cli

import (
	"fmt"
	"strings"

	"hawkings-cli/internal/metadata"
	"hawkings-cli/internal/output"

	"github.com/spf13/cobra"
)

func newDescribeCommand(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "describe [hierarchy|entity <name>|command <path...>]",
		Short: "Describe la jerarquia, entidades y comandos del CLI en JSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			model := metadata.ModelCatalog()
			if len(args) == 0 {
				return output.PrintJSON(model)
			}

			switch args[0] {
			case "hierarchy":
				return output.PrintJSON(map[string]any{
					"name":      model.Name,
					"version":   model.Version,
					"hierarchy": model.Hierarchy,
				})
			case "entity":
				if len(args) < 2 {
					return fmt.Errorf("usage: hawkings describe entity <name>")
				}
				name := strings.TrimSpace(args[1])
				for _, entity := range model.Entities {
					if entity.Name == name {
						return output.PrintJSON(entity)
					}
				}
				return fmt.Errorf("entity %q not found", name)
			case "command":
				if len(args) < 2 {
					return fmt.Errorf("usage: hawkings describe command <path...>")
				}
				path := strings.Join(args[1:], " ")
				for _, command := range model.Commands {
					if command.Path == path {
						return output.PrintJSON(command)
					}
				}
				return fmt.Errorf("command %q not found", path)
			default:
				path := strings.Join(args, " ")
				filtered := make([]metadata.Command, 0, 1)
				for _, item := range model.Commands {
					if item.Path == path {
						filtered = append(filtered, item)
					}
				}
				if len(filtered) == 0 {
					return fmt.Errorf("unknown describe target %q", path)
				}
				return output.PrintJSON(filtered)
			}
		},
	}
}
