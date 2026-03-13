package cli

import (
	"hawkings-cli/internal/output"

	"github.com/spf13/cobra"
)

func newPlatformCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "platform",
		Short: "Comandos sobre learning platforms",
	}
	cmd.AddCommand(newPlatformListCommand(opts))
	return cmd
}

func newPlatformListCommand(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Lista las platforms disponibles para el usuario",
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			platforms, err := rt.Client.GetPlatforms(ctx)
			if err != nil {
				return err
			}

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(platforms)
			}

			rows := make([][]string, 0, len(platforms))
			for _, item := range platforms {
				rows = append(rows, []string{
					intToString(item.ID),
					item.UUID,
					item.Name,
				})
			}
			return output.PrintTable([]string{"ID", "UUID", "Name"}, rows)
		},
	}
}
