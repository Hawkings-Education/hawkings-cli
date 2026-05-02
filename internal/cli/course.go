package cli

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"hawkings-cli/internal/api"
	"hawkings-cli/internal/output"

	"github.com/spf13/cobra"
)

func newCourseCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "course",
		Short: "Comandos sobre courses",
	}
	cmd.AddCommand(newCourseListCommand(opts))
	cmd.AddCommand(newCourseCreateCommand(opts))
	cmd.AddCommand(newCourseImageCommand(opts))
	cmd.AddCommand(newCourseGetCommand(opts))
	cmd.AddCommand(newCourseSectionsCommand(opts))
	cmd.AddCommand(newCourseModulesCommand(opts))
	cmd.AddCommand(newCourseModuleStatusCommand(opts))
	return cmd
}

func newCourseListCommand(opts *rootOptions) *cobra.Command {
	var limit int
	var page int
	var search string
	var status string
	var all bool
	var with []string

	command := &cobra.Command{
		Use:   "list",
		Short: "Lista los courses accesibles para el usuario y permite buscar por texto",
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			params := url.Values{}
			if limit > 0 && !all {
				params.Set("limit", intToString(limit))
			}
			if page > 0 && !all {
				params.Set("page", intToString(page))
			}
			if search != "" {
				params.Set("search", search)
			}
			if status != "" {
				params.Set("status", status)
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			var list api.CourseList
			if all {
				list, err = listAllCourses(ctx, rt.Client, params, with)
			} else {
				list, err = rt.Client.ListCourses(ctx, params, with)
			}
			if err != nil {
				return err
			}

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(list)
			}

			rows := make([][]string, 0, len(list.Data))
			for _, item := range list.Data {
				rows = append(rows, []string{
					intToString(item.ID),
					valueOrDash(stringPtrValue(item.RemoteID)),
					item.Name,
					valueOrDash(stringPtrValue(item.Status)),
					valueOrDash(languageLabel(item.Language)),
				})
			}
			if err := output.PrintTable([]string{"ID", "Remote ID", "Name", "Status", "Language"}, rows); err != nil {
				return err
			}

			writeLine("")
			writeLine("Page %d/%d  Total %d", list.Page, list.Pages, list.Total)
			if !all && list.Pages > 1 {
				writeLine("Hint: usa --all para recuperar todos los resultados en una sola salida.")
			}
			return nil
		},
	}

	command.Flags().IntVar(&limit, "limit", 20, "Limite por pagina")
	command.Flags().IntVar(&page, "page", 1, "Pagina")
	command.Flags().StringVar(&search, "search", "", "Texto libre para buscar por titulo, remote_id o uuid")
	command.Flags().StringVar(&status, "status", "", "Filtra por status")
	command.Flags().BoolVar(&all, "all", false, "Recorre todas las paginas y devuelve todos los resultados")
	command.Flags().StringArrayVar(&with, "with", nil, "Relaciones extra via with[]")

	return command
}

func listAllCourses(ctx context.Context, client *api.Client, params url.Values, with []string) (api.CourseList, error) {
	firstPage, err := client.ListCourses(ctx, params, with)
	if err != nil {
		return api.CourseList{}, err
	}

	items := append([]api.CourseSummary{}, firstPage.Data...)
	for nextPage := 2; nextPage <= firstPage.Pages; nextPage++ {
		pageParams := cloneURLValues(params)
		pageParams.Set("page", intToString(nextPage))

		next, err := client.ListCourses(ctx, pageParams, with)
		if err != nil {
			return api.CourseList{}, err
		}
		items = append(items, next.Data...)
	}

	return paginateCourseSummaries(items, 1, 0), nil
}

func paginateCourseSummaries(items []api.CourseSummary, page, limit int) api.CourseList {
	if limit <= 0 {
		limit = len(items)
		if limit == 0 {
			limit = 1
		}
	}
	if page <= 0 {
		page = 1
	}

	total := len(items)
	pages := (total + limit - 1) / limit
	if pages == 0 {
		pages = 1
	}
	if page > pages {
		page = pages
	}

	start := (page - 1) * limit
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}

	return api.CourseList{
		Data:   items[start:end],
		Pages:  pages,
		Page:   page,
		Offset: start,
		Total:  total,
	}
}

func newCourseCreateCommand(opts *rootOptions) *cobra.Command {
	var programID int
	var input jsonInputOptions
	var dryRun bool

	command := &cobra.Command{
		Use:   "create",
		Short: "Crea o actualiza un course completo via /course/bulk",
		Long: `
Crea o actualiza un course completo, incluyendo sections, modules
y, si el payload los trae, course_contents.

Reglas del backend relevantes:
- course_sections es obligatorio.
- Un module markdown debe traer course_contents, salvo que lleve empty=true.
- Si pasas --program, el CLI relaciona el course despues via POST /course-program/{id}/course.
- /course/bulk es destructivo respecto al arbol enviado: borra sections/modules/contents
  que queden fuera del payload en el ambito gestionado.
- El backend puede responder 200 con errores parciales embebidos; el CLI los detecta y falla.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			payload, err := readJSONObject(input)
			if err != nil {
				return err
			}

			if err := validateCourseBulkPayload(payload); err != nil {
				return err
			}

			if dryRun {
				preview := map[string]any{
					"action":  "course create",
					"program": programID,
					"payload": payload,
					"notes": []string{
						"dry-run no hace escrituras en la API",
						"course_sections es obligatorio en /course/bulk",
						"modules markdown requieren course_contents salvo que uses empty=true",
						"el backend sincroniza el arbol: si omites sections/modules existentes, puede eliminarlos",
					},
				}
				if programID != 0 {
					preview["program_relation"] = map[string]any{
						"endpoint": "/course-program/{program-id}/course",
						"payload":  map[string]any{"add": []any{"<course-id-from-bulk-response>"}},
					}
				}
				return output.PrintJSON(preview)
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			result, err := rt.Client.CreateCourseBulk(ctx, payload)
			if err != nil {
				return err
			}
			if errs := collectBulkErrors(result); len(errs) > 0 {
				return fmt.Errorf("course bulk returned partial errors:\n- %s", strings.Join(errs, "\n- "))
			}

			var relatedCourses []api.CourseDetail
			if programID != 0 {
				courseID := anyInt(result["id"])
				if courseID == 0 {
					return fmt.Errorf("course bulk succeeded but did not return a course id; cannot relate to program %d", programID)
				}
				relatedCourses, err = rt.Client.UpdateProgramCourses(ctx, intToString(programID), map[string]any{
					"add": []int{courseID},
				})
				if err != nil {
					return err
				}
			}

			if output.WantsJSON(rt.Format) {
				response := map[string]any{
					"action": "course create",
					"result": result,
				}
				if programID != 0 {
					response["program_relation"] = map[string]any{
						"program_id": programID,
						"action":     "add",
						"courses":    relatedCourses,
					}
				}
				return output.PrintJSON(response)
			}

			rows := [][]string{
				{"Course ID", intToString(anyInt(result["id"]))},
				{"Program ID", anyStringOrDash(programID)},
				{"Name", mapString(result, "name")},
				{"Remote ID", valueOrDash(mapString(result, "remote_id"))},
				{"Sections", intToString(anyLen(result["course_sections"]))},
				{"Course-level modules", intToString(anyLen(result["course_modules"]))},
			}
			if err := output.PrintTable([]string{"Field", "Value"}, rows); err != nil {
				return err
			}
			if programID != 0 {
				writeLine("")
				writeLine("Relacionado al program %d via POST /course-program/%d/course (add).", programID, programID)
			}
			writeLine("")
			writeLine("Nota: usa `course get %d` o `course modules %d` para inspeccionar la estructura creada.", anyInt(result["id"]), anyInt(result["id"]))
			return nil
		},
	}

	command.Flags().IntVar(&programID, "program", 0, "ID del program a asociar despues via /course-program/{id}/course")
	addJSONInputFlags(command, &input)
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Muestra el payload final sin enviar peticiones")

	return command
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
				{"Image", stringPtrOrDash(course.Image)},
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

func validateCourseBulkPayload(payload map[string]any) error {
	if payload == nil {
		return fmt.Errorf("course payload is required")
	}

	sections, ok := payload["course_sections"]
	if !ok {
		return fmt.Errorf("course payload must include course_sections for /course/bulk")
	}
	if _, ok := sections.([]any); !ok {
		return fmt.Errorf("course_sections must be a JSON array")
	}

	return nil
}

func collectBulkErrors(result map[string]any) []string {
	errors := make([]string, 0)
	collectBulkErrorsInto(&errors, "result", result)
	return errors
}

func collectBulkErrorsInto(errors *[]string, path string, value any) {
	switch typed := value.(type) {
	case map[string]any:
		if message := strings.TrimSpace(mapString(typed, "error")); message != "" {
			*errors = append(*errors, path+": "+message)
		}
		for key, nested := range typed {
			if key == "error" {
				continue
			}
			collectBulkErrorsInto(errors, path+"."+key, nested)
		}
	case []any:
		for index, nested := range typed {
			collectBulkErrorsInto(errors, fmt.Sprintf("%s[%d]", path, index), nested)
		}
	}
}

func flattenCourseModules(course api.CourseDetail) []api.CourseModule {
	modules := make([]api.CourseModule, 0, courseAllModuleCount(course))
	modules = append(modules, course.CourseModules...)
	for _, section := range course.CourseSections {
		modules = append(modules, section.CourseModules...)
	}
	return modules
}
