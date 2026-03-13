package cli

import (
	"fmt"

	"hawkings-cli/internal/output"

	"github.com/spf13/cobra"
)

func newTemplateCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "template",
		Short: "Comandos sobre templates de programa",
	}
	cmd.AddCommand(newTemplateListCommand(opts))
	return cmd
}

func newTemplateListCommand(opts *rootOptions) *cobra.Command {
	command := &cobra.Command{
		Use:   "list",
		Short: "Lista los templates de programa disponibles para el customer activo",
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			templates, err := rt.Client.ListProgramTemplates(ctx)
			if err != nil {
				return err
			}

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(templates)
			}

			rows := make([][]string, 0, len(templates))
			for _, item := range templates {
				rows = append(rows, []string{
					intToString(item.ID),
					valueOrDash(item.Code),
					item.Name,
					templateRangeLabel(item.CoursesMin, item.CoursesMax),
					templateRangeLabel(item.CoursesHoursMin, item.CoursesHoursMax),
					intToString(len(item.Related)),
				})
			}
			return output.PrintTable([]string{"ID", "Code", "Name", "Courses", "Hours", "Relations"}, rows)
		},
	}

	return command
}

func templateRangeLabel(minValue any, maxValue any) string {
	minimum := anyInt(minValue)
	maximum := anyInt(maxValue)

	switch {
	case maximum <= 0:
		return fmt.Sprintf("%d..inf", minimum)
	default:
		return fmt.Sprintf("%d..%d", minimum, maximum)
	}
}
