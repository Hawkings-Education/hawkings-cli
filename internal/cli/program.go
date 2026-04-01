package cli

import (
	"context"
	"net/url"
	"sort"
	"strings"

	"hawkings-cli/internal/api"
	"hawkings-cli/internal/output"

	"github.com/spf13/cobra"
)

var defaultProgramWith = []string{"courseProgramTemplate", "language", "user", "spaces", "courseFaculty", "coursesCount"}

func newProgramCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "program",
		Short: "Comandos sobre course programs",
	}
	cmd.AddCommand(newProgramListCommand(opts))
	cmd.AddCommand(newProgramCreateCommand(opts))
	cmd.AddCommand(newProgramUpdateCommand(opts))
	cmd.AddCommand(newProgramDeleteCommand(opts))
	cmd.AddCommand(newProgramSetSpacesCommand(opts))
	cmd.AddCommand(newProgramSetCoursesCommand(opts))
	cmd.AddCommand(newProgramAddCourseCommand(opts))
	cmd.AddCommand(newProgramRemoveCourseCommand(opts))
	cmd.AddCommand(newProgramGenerateSyllabusCommand(opts))
	cmd.AddCommand(newProgramCreateCoursesCommand(opts))
	cmd.AddCommand(newProgramGetCommand(opts))
	cmd.AddCommand(newProgramTreeCommand(opts))
	cmd.AddCommand(newProgramSyllabusCommand(opts))
	cmd.AddCommand(newProgramConfigCommand(opts))
	cmd.AddCommand(newProgramCoursesCommand(opts))
	cmd.AddCommand(newProgramStatusMatrixCommand(opts))
	return cmd
}

func newProgramListCommand(opts *rootOptions) *cobra.Command {
	var limit int
	var page int
	var search string
	var status string
	var spaceID int
	var all bool
	var with []string

	command := &cobra.Command{
		Use:   "list",
		Short: "Lista los programas accesibles para el usuario",
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
			for _, value := range with {
				if value != "" {
					params.Add("with[]", value)
				}
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			var list api.ProgramList

			if spaceID > 0 {
				programs, err := rt.Client.GetSpacePrograms(ctx, intToString(spaceID), uniqueStrings(append(defaultProgramWith, with...)))
				if err != nil {
					return err
				}
				summaries := make([]api.ProgramSummary, 0, len(programs))
				for _, item := range programs {
					summaries = append(summaries, programSummaryFromDetail(item))
				}
				if all {
					list = paginateProgramSummaries(filterProgramSummaries(summaries, search, status), 1, 0)
				} else {
					list = paginateProgramSummaries(filterProgramSummaries(summaries, search, status), page, limit)
				}
			} else {
				if all {
					list, err = listAllPrograms(ctx, rt.Client, params)
				} else {
					list, err = rt.Client.ListPrograms(ctx, params)
				}
				if err != nil {
					return err
				}
			}

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(list)
			}

			rows := make([][]string, 0, len(list.Data))
			for _, item := range list.Data {
				rows = append(rows, []string{
					intToString(item.ID),
					valueOrDash(metadataString(item.Metadata, "code")),
					item.Name,
					stringPtrOrDash(item.Status),
					valueOrDash(metadataString(item.Metadata, "hours")),
				})
			}
			if err := output.PrintTable([]string{"ID", "Code", "Name", "Status", "Hours"}, rows); err != nil {
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
	command.Flags().StringVar(&search, "search", "", "Texto de busqueda")
	command.Flags().StringVar(&status, "status", "", "Filtra por status")
	command.Flags().IntVar(&spaceID, "space-id", 0, "Filtra por membresia real en un space usando /space/{id}/course-program")
	command.Flags().BoolVar(&all, "all", false, "Recorre todas las paginas y devuelve todos los resultados")
	command.Flags().StringArrayVar(&with, "with", nil, "Anade relaciones with[]")

	return command
}

func listAllPrograms(ctx context.Context, client *api.Client, params url.Values) (api.ProgramList, error) {
	firstPage, err := client.ListPrograms(ctx, params)
	if err != nil {
		return api.ProgramList{}, err
	}

	items := append([]api.ProgramSummary{}, firstPage.Data...)
	for nextPage := 2; nextPage <= firstPage.Pages; nextPage++ {
		pageParams := cloneURLValues(params)
		pageParams.Set("page", intToString(nextPage))

		next, err := client.ListPrograms(ctx, pageParams)
		if err != nil {
			return api.ProgramList{}, err
		}
		items = append(items, next.Data...)
	}

	return paginateProgramSummaries(items, 1, 0), nil
}

func newProgramGetCommand(opts *rootOptions) *cobra.Command {
	var with []string
	var withCourses bool

	command := &cobra.Command{
		Use:   "get <program-id>",
		Short: "Muestra el detalle de un programa",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			relations := append([]string{}, defaultProgramWith...)
			relations = append(relations, with...)
			if withCourses {
				relations = append(relations, "coursesSectionsModules")
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			program, err := rt.Client.GetProgram(ctx, args[0], uniqueStrings(relations))
			if err != nil {
				return err
			}

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(program)
			}

			rows := [][]string{
				{"ID", intToString(program.ID)},
				{"Name", program.Name},
				{"Status", stringPtrOrDash(program.Status)},
				{"Enabled", boolToYesNo(program.Enabled)},
				{"Remote ID", stringPtrOrDash(program.RemoteID)},
				{"Code", valueOrDash(metadataString(program.Metadata, "code"))},
				{"Hours", valueOrDash(metadataString(program.Metadata, "hours"))},
				{"Template", valueOrDash(mapString(program.CourseProgramTemplate, "name"))},
				{"Language", valueOrDash(languageLabel(program.Language))},
				{"Spaces", intToString(len(program.Spaces))},
				{"Syllabus", boolToAvailability(programHasSyllabus(program))},
				{"Courses", boolToAvailability(programHasCourses(program))},
				{"Courses count", intToString(anyInt(program.CoursesCount))},
			}
			if err := output.PrintTable([]string{"Field", "Value"}, rows); err != nil {
				return err
			}
			writeLine("")
			writeLine("Hint: %s", programStatusHint(program))
			return nil
		},
	}

	command.Flags().StringArrayVar(&with, "with", nil, "Relaciones extra via with[]")
	command.Flags().BoolVar(&withCourses, "with-courses", false, "Incluye coursesSectionsModules")

	return command
}

func newProgramSyllabusCommand(opts *rootOptions) *cobra.Command {
	command := &cobra.Command{
		Use:   "syllabus <program-id>",
		Short: "Muestra el syllabus almacenado de un programa",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			program, err := rt.Client.GetProgram(ctx, args[0], nil)
			if err != nil {
				return err
			}

			payload := map[string]any{
				"id":           program.ID,
				"name":         program.Name,
				"status":       stringPtrValue(program.Status),
				"has_syllabus": programHasSyllabus(program),
				"syllabus":     program.Syllabus,
				"hint":         programStatusHint(program),
			}

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(payload)
			}

			if !programHasSyllabus(program) {
				writeLine("El programa %d no tiene syllabus.", program.ID)
				writeLine("Hint: %s", programStatusHint(program))
				return nil
			}
			return output.PrintJSON(payload)
		},
	}

	return command
}

func newProgramTreeCommand(opts *rootOptions) *cobra.Command {
	command := &cobra.Command{
		Use:   "tree <program-id>",
		Short: "Imprime el arbol navegable de program a course, section y module",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			program, err := rt.Client.GetProgram(ctx, args[0], []string{"coursesSectionsModules"})
			if err != nil {
				return err
			}

			courses := sortedCourses(program.Courses)
			courseNodes := make([]map[string]any, 0, len(courses))
			for _, course := range courses {
				courseModules := sortedModules(course.CourseModules)
				sections := sortedSections(course.CourseSections)

				directModules := make([]map[string]any, 0, len(courseModules))
				for _, module := range courseModules {
					directModules = append(directModules, map[string]any{
						"id":     module.ID,
						"name":   module.Name,
						"type":   module.Type,
						"order":  module.Order,
						"status": normalizedStatus(module.Status),
					})
				}

				sectionNodes := make([]map[string]any, 0, len(sections))
				for _, section := range sections {
					modules := sortedModules(section.CourseModules)
					moduleNodes := make([]map[string]any, 0, len(modules))
					for _, module := range modules {
						moduleNodes = append(moduleNodes, map[string]any{
							"id":     module.ID,
							"name":   module.Name,
							"type":   module.Type,
							"order":  module.Order,
							"status": normalizedStatus(module.Status),
						})
					}
					sectionNodes = append(sectionNodes, map[string]any{
						"id":      section.ID,
						"name":    section.Name,
						"order":   section.Order,
						"modules": moduleNodes,
					})
				}

				courseNodes = append(courseNodes, map[string]any{
					"id":             course.ID,
					"name":           course.Name,
					"status":         normalizedStatus(course.Status),
					"sections_count": len(course.CourseSections),
					"modules_count":  courseAllModuleCount(course),
					"direct_modules": directModules,
					"sections":       sectionNodes,
				})
			}

			payload := map[string]any{
				"hierarchy": "program -> course -> section -> module -> content",
				"program": map[string]any{
					"id":           program.ID,
					"name":         program.Name,
					"status":       normalizedStatus(program.Status),
					"has_syllabus": programHasSyllabus(program),
					"has_courses":  programHasCourses(program),
					"courses":      len(program.Courses),
					"hint":         programStatusHint(program),
				},
				"courses": courseNodes,
				"notes": []string{
					"El arbol no carga file.contents de los modules para evitar payloads grandes.",
					"Usa module content <module-id> para leer el contenido real del modulo.",
				},
				"next_commands": []string{
					"course get <course-id>",
					"course modules <course-id>",
					"module content <module-id>",
					"describe hierarchy",
				},
			}

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(payload)
			}

			writeLine("Program %d: %s", program.ID, program.Name)
			writeLine("Status: %s", normalizedStatus(program.Status))
			writeLine("Hint: %s", programStatusHint(program))
			writeLine("Hierarchy: program -> course -> section -> module -> content")
			writeLine("")

			if !programHasCourses(program) {
				writeLine("No hay courses asociados a este programa.")
				writeLine("Usa `program syllabus %d` o `program get %d` para inspeccionar lo disponible.", program.ID, program.ID)
				return nil
			}

			for _, course := range courses {
				writeLine("  Course %d: %s [status=%s, sections=%d, modules=%d]", course.ID, course.Name, normalizedStatus(course.Status), len(course.CourseSections), courseAllModuleCount(course))

				for _, module := range sortedModules(course.CourseModules) {
					writeLine("    Module %d: (%s, order=%d, status=%s) %s", module.ID, module.Type, module.Order, normalizedStatus(module.Status), module.Name)
				}

				for _, section := range sortedSections(course.CourseSections) {
					writeLine("    Section %d: (%d modules) %s", section.ID, len(section.CourseModules), section.Name)
					for _, module := range sortedModules(section.CourseModules) {
						writeLine("      Module %d: (%s, order=%d, status=%s) %s", module.ID, module.Type, module.Order, normalizedStatus(module.Status), module.Name)
					}
				}
			}

			writeLine("")
			writeLine("Siguiente paso: usa `module content <module-id>` para leer el texto de un modulo concreto.")
			return nil
		},
	}

	return command
}

func newProgramConfigCommand(opts *rootOptions) *cobra.Command {
	command := &cobra.Command{
		Use:   "config <program-id>",
		Short: "Muestra la configuracion relevante del programa",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			program, err := rt.Client.GetProgram(ctx, args[0], defaultProgramWith)
			if err != nil {
				return err
			}

			payload := map[string]any{
				"id":                          program.ID,
				"name":                        program.Name,
				"status":                      stringPtrValue(program.Status),
				"remote_id":                   stringPtrValue(program.RemoteID),
				"enabled":                     program.Enabled,
				"metadata":                    program.Metadata,
				"template":                    program.CourseProgramTemplate,
				"language":                    program.Language,
				"course_faculty":              program.CourseFaculty,
				"spaces":                      program.Spaces,
				"user":                        program.User,
				"context":                     stringPtrValue(program.Context),
				"syllabus_prompt":             stringPtrValue(program.SyllabusPrompt),
				"course_module_prompt_custom": stringPtrValue(program.CourseModulePromptCustom),
			}

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(payload)
			}

			rows := [][]string{
				{"ID", intToString(program.ID)},
				{"Name", program.Name},
				{"Status", stringPtrOrDash(program.Status)},
				{"Template", valueOrDash(mapString(program.CourseProgramTemplate, "name"))},
				{"Language", valueOrDash(languageLabel(program.Language))},
				{"Faculty", valueOrDash(mapString(program.CourseFaculty, "name"))},
				{"Spaces", valueOrDash(joinNames(program.Spaces))},
				{"Code", valueOrDash(metadataString(program.Metadata, "code"))},
				{"Hours", valueOrDash(metadataString(program.Metadata, "hours"))},
				{"Context chars", intToString(len(stringPtrValue(program.Context)))},
				{"Syllabus prompt chars", intToString(len(stringPtrValue(program.SyllabusPrompt)))},
				{"Module prompt chars", intToString(len(stringPtrValue(program.CourseModulePromptCustom)))},
			}
			if err := output.PrintTable([]string{"Field", "Value"}, rows); err != nil {
				return err
			}

			printOptionalBlock("Context", stringPtrValue(program.Context))
			printOptionalBlock("Syllabus Prompt", stringPtrValue(program.SyllabusPrompt))
			printOptionalBlock("Course Module Prompt Custom", stringPtrValue(program.CourseModulePromptCustom))
			return nil
		},
	}

	return command
}

func newProgramCoursesCommand(opts *rootOptions) *cobra.Command {
	command := &cobra.Command{
		Use:   "courses <program-id>",
		Short: "Lista los courses de un programa y resume su estructura",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			program, err := rt.Client.GetProgram(ctx, args[0], []string{"coursesSectionsModules"})
			if err != nil {
				return err
			}

			payload := map[string]any{
				"program": map[string]any{
					"id":           program.ID,
					"name":         program.Name,
					"status":       stringPtrValue(program.Status),
					"has_syllabus": programHasSyllabus(program),
					"has_courses":  programHasCourses(program),
					"hint":         programStatusHint(program),
				},
				"courses": program.Courses,
			}

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(payload)
			}

			writeLine("Program %d: %s", program.ID, program.Name)
			writeLine("Status: %s", stringPtrOrDash(program.Status))
			writeLine("Hint: %s", programStatusHint(program))
			writeLine("")

			if !programHasCourses(program) {
				writeLine("No hay courses asociados a este programa.")
				return nil
			}

			rows := make([][]string, 0, len(program.Courses))
			for _, course := range program.Courses {
				rows = append(rows, []string{
					intToString(course.ID),
					course.Name,
					stringPtrOrDash(course.Status),
					intToString(len(course.CourseSections)),
					intToString(courseAllModuleCount(course)),
				})
			}
			return output.PrintTable([]string{"ID", "Name", "Status", "Sections", "Modules"}, rows)
		},
	}

	return command
}

func newProgramStatusMatrixCommand(opts *rootOptions) *cobra.Command {
	var limit int
	var samples int

	command := &cobra.Command{
		Use:   "status-matrix",
		Short: "Agrupa programas por status y disponibilidad real de courses",
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			page := 1
			allPrograms := make([]api.ProgramSummary, 0)
			requestedPages := 0

			for {
				params := url.Values{}
				params.Set("limit", intToString(limit))
				params.Set("page", intToString(page))
				params.Add("with[]", "coursesCount")

				list, err := rt.Client.ListPrograms(ctx, params)
				if err != nil {
					return err
				}

				requestedPages++
				allPrograms = append(allPrograms, list.Data...)

				if list.Pages <= page || len(list.Data) == 0 {
					break
				}
				page++
			}

			type statusBucket struct {
				Status         string `json:"status"`
				Programs       int    `json:"programs"`
				WithCourses    int    `json:"with_courses"`
				WithoutCourses int    `json:"without_courses"`
				SampleIDs      []int  `json:"sample_program_ids"`
			}

			buckets := map[string]*statusBucket{}
			for _, program := range allPrograms {
				status := normalizedStatus(program.Status)
				bucket, ok := buckets[status]
				if !ok {
					bucket = &statusBucket{Status: status, SampleIDs: []int{}}
					buckets[status] = bucket
				}

				bucket.Programs++
				if anyInt(program.CoursesCount) > 0 {
					bucket.WithCourses++
				} else {
					bucket.WithoutCourses++
				}

				if samples > 0 && len(bucket.SampleIDs) < samples {
					bucket.SampleIDs = append(bucket.SampleIDs, program.ID)
				}
			}

			statuses := make([]statusBucket, 0, len(buckets))
			for _, bucket := range buckets {
				statuses = append(statuses, *bucket)
			}
			sort.Slice(statuses, func(i, j int) bool {
				leftKey := programStatusSortKey(statuses[i].Status)
				rightKey := programStatusSortKey(statuses[j].Status)
				if leftKey != rightKey {
					return leftKey < rightKey
				}
				return statuses[i].Status < statuses[j].Status
			})

			payload := map[string]any{
				"total_programs":  len(allPrograms),
				"requested_pages": requestedPages,
				"requested_limit": limit,
				"statuses":        statuses,
				"notes": []string{
					"La disponibilidad de courses se calcula con courses_count del listado.",
					"El syllabus no se inspecciona aqui; usa program get o program syllabus para verlo.",
				},
			}

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(payload)
			}

			rows := make([][]string, 0, len(statuses))
			for _, item := range statuses {
				sampleIDs := make([]string, 0, len(item.SampleIDs))
				for _, id := range item.SampleIDs {
					sampleIDs = append(sampleIDs, intToString(id))
				}
				rows = append(rows, []string{
					item.Status,
					intToString(item.Programs),
					intToString(item.WithCourses),
					intToString(item.WithoutCourses),
					valueOrDash(strings.Join(sampleIDs, ", ")),
				})
			}

			if err := output.PrintTable([]string{"Status", "Programs", "With courses", "Without courses", "Sample IDs"}, rows); err != nil {
				return err
			}
			writeLine("")
			writeLine("Nota: para syllabus real usa `program get <id>` o `program syllabus <id>`.")
			return nil
		},
	}

	command.Flags().IntVar(&limit, "limit", 100, "Tamano de pagina para recorrer el listado completo")
	command.Flags().IntVar(&samples, "samples", 3, "Numero de program IDs de ejemplo por status")

	return command
}

func programSummaryFromDetail(program api.ProgramDetail) api.ProgramSummary {
	return api.ProgramSummary{
		ID:           program.ID,
		Name:         program.Name,
		RemoteID:     program.RemoteID,
		Enabled:      program.Enabled,
		Status:       program.Status,
		Syllabus:     program.Syllabus,
		CreatedAt:    program.CreatedAt,
		UpdatedAt:    program.UpdatedAt,
		Metadata:     cloneMap(program.Metadata),
		CoursesCount: program.CoursesCount,
		Language:     program.Language,
		User:         program.User,
		Spaces:       cloneMaps(program.Spaces),
	}
}

func filterProgramSummaries(programs []api.ProgramSummary, search, status string) []api.ProgramSummary {
	search = strings.ToLower(strings.TrimSpace(search))
	status = strings.TrimSpace(status)

	filtered := make([]api.ProgramSummary, 0, len(programs))
	for _, item := range programs {
		if status != "" && stringPtrValue(item.Status) != status {
			continue
		}
		if search != "" {
			haystack := strings.ToLower(strings.Join([]string{
				item.Name,
				stringPtrValue(item.RemoteID),
				metadataString(item.Metadata, "code"),
			}, " "))
			if !strings.Contains(haystack, search) {
				continue
			}
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func paginateProgramSummaries(programs []api.ProgramSummary, page, limit int) api.ProgramList {
	if limit <= 0 {
		limit = len(programs)
		if limit == 0 {
			limit = 1
		}
	}
	if page <= 0 {
		page = 1
	}

	total := len(programs)
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

	return api.ProgramList{
		Data:   programs[start:end],
		Pages:  pages,
		Page:   page,
		Offset: start,
		Total:  total,
	}
}

func sortedCourses(courses []api.CourseDetail) []api.CourseDetail {
	out := append([]api.CourseDetail(nil), courses...)
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

func sortedSections(sections []api.CourseSection) []api.CourseSection {
	out := append([]api.CourseSection(nil), sections...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Order != out[j].Order {
			return out[i].Order < out[j].Order
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func sortedModules(modules []api.CourseModule) []api.CourseModule {
	out := append([]api.CourseModule(nil), modules...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Order != out[j].Order {
			return out[i].Order < out[j].Order
		}
		return out[i].ID < out[j].ID
	})
	return out
}
