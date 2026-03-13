package cli

import (
	"hawkings-cli/internal/api"
	"hawkings-cli/internal/output"

	"github.com/spf13/cobra"
)

func newCourseCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "course",
		Short: "Comandos sobre courses",
	}
	cmd.AddCommand(newCourseGetCommand(opts))
	cmd.AddCommand(newCourseSectionsCommand(opts))
	cmd.AddCommand(newCourseModulesCommand(opts))
	cmd.AddCommand(newCourseModuleStatusCommand(opts))
	return cmd
}

func newCourseGetCommand(opts *rootOptions) *cobra.Command {
	var contents bool

	command := &cobra.Command{
		Use:   "get <course-id>",
		Short: "Muestra un course con su estructura de sections y modules",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			relations := []string{"courseModules", "courseSectionsModules"}
			if contents {
				relations = []string{"courseModulesContents", "courseSectionsModulesContents"}
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			course, err := rt.Client.GetCourse(ctx, args[0], relations)
			if err != nil {
				return err
			}

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(course)
			}

			rows := [][]string{
				{"ID", intToString(course.ID)},
				{"Name", course.Name},
				{"Status", stringPtrOrDash(course.Status)},
				{"Remote ID", stringPtrOrDash(course.RemoteID)},
				{"Language", valueOrDash(languageLabel(course.Language))},
				{"Sections", intToString(len(course.CourseSections))},
				{"Modules", intToString(courseAllModuleCount(course))},
				{"Course-level modules", intToString(len(course.CourseModules))},
			}
			if err := output.PrintTable([]string{"Field", "Value"}, rows); err != nil {
				return err
			}

			if len(course.CourseSections) == 0 && len(course.CourseModules) == 0 {
				writeLine("")
				writeLine("El course no trae sections ni modules con este payload.")
				return nil
			}

			writeLine("")
			writeLine("Sections:")
			sectionRows := make([][]string, 0, len(course.CourseSections))
			for _, section := range course.CourseSections {
				sectionRows = append(sectionRows, []string{
					intToString(section.ID),
					section.Name,
					intToString(len(section.CourseModules)),
				})
			}
			return output.PrintTable([]string{"ID", "Name", "Modules"}, sectionRows)
		},
	}

	command.Flags().BoolVar(&contents, "contents", false, "Incluye course contents dentro de los modules")
	return command
}

func newCourseSectionsCommand(opts *rootOptions) *cobra.Command {
	command := &cobra.Command{
		Use:   "sections <course-id>",
		Short: "Lista las sections de un course",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			course, err := rt.Client.GetCourse(ctx, args[0], []string{"courseSectionsModules"})
			if err != nil {
				return err
			}

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(course.CourseSections)
			}

			rows := make([][]string, 0, len(course.CourseSections))
			for _, section := range course.CourseSections {
				rows = append(rows, []string{
					intToString(section.ID),
					intToString(section.Order),
					section.Name,
					intToString(courseSectionModuleCount(section)),
				})
			}
			return output.PrintTable([]string{"ID", "Order", "Name", "Modules"}, rows)
		},
	}
	return command
}

func newCourseModulesCommand(opts *rootOptions) *cobra.Command {
	var contents bool

	command := &cobra.Command{
		Use:   "modules <course-id>",
		Short: "Lista todos los modules de un course, a nivel curso y seccion",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			relations := []string{"courseModules", "courseSectionsModules"}
			if contents {
				relations = []string{"courseModulesContents", "courseSectionsModulesContents"}
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			course, err := rt.Client.GetCourse(ctx, args[0], relations)
			if err != nil {
				return err
			}

			type moduleRow struct {
				ID            int            `json:"id"`
				Scope         string         `json:"scope"`
				Section       string         `json:"section,omitempty"`
				Order         int            `json:"order"`
				Type          string         `json:"type"`
				Name          string         `json:"name"`
				Status        string         `json:"status,omitempty"`
				ContentsCount int            `json:"contents_count"`
				Metadata      map[string]any `json:"metadata,omitempty"`
			}

			rows := make([]moduleRow, 0, courseAllModuleCount(course))
			for _, module := range course.CourseModules {
				rows = append(rows, moduleRow{
					ID:            module.ID,
					Scope:         "course",
					Order:         module.Order,
					Type:          module.Type,
					Name:          module.Name,
					Status:        stringPtrValue(module.Status),
					ContentsCount: len(module.CourseContents),
					Metadata:      module.Metadata,
				})
			}
			for _, section := range course.CourseSections {
				for _, module := range section.CourseModules {
					rows = append(rows, moduleRow{
						ID:            module.ID,
						Scope:         "section",
						Section:       section.Name,
						Order:         module.Order,
						Type:          module.Type,
						Name:          module.Name,
						Status:        stringPtrValue(module.Status),
						ContentsCount: len(module.CourseContents),
						Metadata:      module.Metadata,
					})
				}
			}

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(rows)
			}

			tableRows := make([][]string, 0, len(rows))
			for _, row := range rows {
				tableRows = append(tableRows, []string{
					intToString(row.ID),
					row.Scope,
					valueOrDash(row.Section),
					intToString(row.Order),
					row.Type,
					valueOrDash(row.Status),
					intToString(row.ContentsCount),
					row.Name,
				})
			}
			return output.PrintTable([]string{"ID", "Scope", "Section", "Order", "Type", "Status", "Contents", "Name"}, tableRows)
		},
	}

	command.Flags().BoolVar(&contents, "contents", false, "Incluye course contents dentro de los modules")
	return command
}

func newCourseModuleStatusCommand(opts *rootOptions) *cobra.Command {
	command := &cobra.Command{
		Use:   "module-status <course-id>",
		Short: "Obtiene el status de todos los modules de un course",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			statuses, err := rt.Client.GetCourseModulesStatus(ctx, args[0])
			if err != nil {
				return err
			}

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(statuses)
			}

			rows := make([][]string, 0, len(statuses))
			for id, status := range statuses {
				rows = append(rows, []string{id, status})
			}
			return output.PrintTable([]string{"Module ID", "Status"}, rows)
		},
	}
	return command
}

func flattenCourseModules(course api.CourseDetail) []api.CourseModule {
	modules := make([]api.CourseModule, 0, courseAllModuleCount(course))
	modules = append(modules, course.CourseModules...)
	for _, section := range course.CourseSections {
		modules = append(modules, section.CourseModules...)
	}
	return modules
}
