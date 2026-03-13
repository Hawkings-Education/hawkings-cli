package cli

import (
	"hawkings-cli/internal/output"

	"github.com/spf13/cobra"
)

func newLanguageCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "language",
		Short: "Comandos sobre idiomas de referencia",
	}
	cmd.AddCommand(newLanguageListCommand(opts))
	return cmd
}

func newLanguageListCommand(opts *rootOptions) *cobra.Command {
	command := &cobra.Command{
		Use:   "list",
		Short: "Lista los idiomas disponibles en Hawkings",
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			languages, err := rt.Client.ListLanguages(ctx)
			if err != nil {
				return err
			}

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(languages)
			}

			rows := make([][]string, 0, len(languages))
			for _, item := range languages {
				rows = append(rows, []string{
					intToString(item.ID),
					valueOrDash(item.Code),
					item.Name,
					boolToYesNo(item.RTL),
				})
			}
			return output.PrintTable([]string{"ID", "Code", "Name", "RTL"}, rows)
		},
	}

	return command
}
