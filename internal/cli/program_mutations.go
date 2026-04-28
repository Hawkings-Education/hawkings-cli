package cli

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"hawkings-cli/internal/api"
	"hawkings-cli/internal/output"

	"github.com/spf13/cobra"
)

var programCreatePayloadFields = []string{
	"remote_id",
	"name",
	"status",
	"metadata",
	"syllabus",
	"syllabus_prompt",
	"context",
	"course_module_prompt_custom",
	"research_instructions",
	"enabled",
	"language_id",
	"course_faculty_id",
	"course_program_template_id",
	"relate",
	"image",
	"image_delete",
	"image_generate",
}

var programUpdateOnlyPayloadFields = []string{
	"remote_id",
	"name",
	"status",
	"metadata",
	"syllabus",
	"syllabus_prompt",
	"context",
	"course_module_prompt_custom",
	"research_instructions",
	"course_faculty_id",
	"course_program_template_id",
	"language_id",
}

var programGenerateSyllabusPayloadFields = []string{
	"context",
	"syllabus_prompt",
	"force",
}

func newProgramCreateCommand(opts *rootOptions) *cobra.Command {
	var input jsonInputOptions
	var syllabusFile string
	var spaces []int
	var dryRun bool

	command := &cobra.Command{
		Use:   "create",
		Short: "Crea un programa a partir de un payload JSON y puede asignarle espacios",
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			payload, err := readJSONObject(input)
			if err != nil {
				return err
			}
			if err := validatePayloadFields("program create", payload, programCreatePayloadFields); err != nil {
				return err
			}
			if syllabusFile != "" {
				syllabus, err := readJSONFile(syllabusFile)
				if err != nil {
					return err
				}
				payload["syllabus"] = syllabus
			}

			preview := map[string]any{
				"action":         "program create",
				"create_payload": payload,
			}
			if len(spaces) > 0 {
				preview["spaces_payload"] = map[string]any{"selected": spaces}
			}

			if dryRun {
				return output.PrintJSON(preview)
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			program, err := rt.Client.CreateProgram(ctx, payload)
			if err != nil {
				return err
			}
			program = normalizeProgramDetail(program)
			if err := validateProgramPersistedPayloadFields("program create", payload, program); err != nil {
				return err
			}

			response := map[string]any{
				"program": program,
			}

			if len(spaces) > 0 {
				assignedSpaces, err := rt.Client.UpdateProgramSpaces(ctx, intToString(program.ID), spaces)
				if err != nil {
					return err
				}
				response["spaces"] = assignedSpaces
			}

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(response)
			}

			rows := [][]string{
				{"ID", intToString(program.ID)},
				{"Name", program.Name},
				{"Status", normalizedStatus(program.Status)},
				{"Language", valueOrDash(languageLabel(program.Language))},
				{"Courses count", intToString(anyInt(program.CoursesCount))},
			}
			if err := output.PrintTable([]string{"Field", "Value"}, rows); err != nil {
				return err
			}
			if len(spaces) > 0 {
				writeLine("")
				writeLine("Espacios asignados: %v", spaces)
			}
			return nil
		},
	}

	addJSONInputFlags(command, &input)
	command.Flags().StringVar(&syllabusFile, "syllabus-file", "", "Ruta a un JSON con el syllabus a inyectar en el payload")
	command.Flags().IntSliceVar(&spaces, "space", nil, "IDs de spaces a asignar justo despues de crear")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Muestra los payloads sin enviar peticiones")

	return command
}

func newProgramUpdateCommand(opts *rootOptions) *cobra.Command {
	var input jsonInputOptions
	var dryRun bool

	command := &cobra.Command{
		Use:   "update <program-id>",
		Short: "Hace PATCH /only del programa con solo los campos enviados",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			patch, err := readJSONObject(input)
			if err != nil {
				return err
			}
			if err := validatePayloadFields("program update", patch, programUpdateOnlyPayloadFields); err != nil {
				return err
			}

			if dryRun {
				return output.PrintJSON(map[string]any{
					"action":     "program update",
					"program_id": args[0],
					"patch":      patch,
					"endpoint":   "PATCH /course-program/{id}/only",
				})
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			updated, err := rt.Client.UpdateProgramOnly(ctx, args[0], patch)
			if err != nil {
				return err
			}
			updated = normalizeProgramDetail(updated)
			if err := validateProgramPersistedPayloadFields("program update", patch, updated); err != nil {
				return err
			}

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(updated)
			}

			rows := [][]string{
				{"ID", intToString(updated.ID)},
				{"Name", updated.Name},
				{"Status", normalizedStatus(updated.Status)},
				{"Syllabus", boolToAvailability(programHasSyllabus(updated))},
				{"Courses", boolToAvailability(programHasCourses(updated))},
			}
			if err := output.PrintTable([]string{"Field", "Value"}, rows); err != nil {
				return err
			}
			writeLine("")
			writeLine("Hint: %s", programStatusHint(updated))
			return nil
		},
	}

	addJSONInputFlags(command, &input)
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Muestra el patch tal como se enviara a PATCH /only")

	return command
}

func newProgramDeleteCommand(opts *rootOptions) *cobra.Command {
	var dryRun bool

	command := &cobra.Command{
		Use:   "delete <program-id>",
		Short: "Elimina un programa y avisa de la semantica real de courses asociados",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			warning := "Deleting a program does not cascade-delete shared courses. The backend only deletes courses linked exclusively to this program; courses associated to other programs are preserved."

			if dryRun {
				return output.PrintJSON(map[string]any{
					"action":   "program delete",
					"program":  args[0],
					"warning":  warning,
					"endpoint": "/course-program/" + args[0],
				})
			}

			ctx, cancel := commandContextWithMinimum(rt, opts.timeout, 10*time.Minute)
			defer cancel()

			if err := rt.Client.DeleteProgram(ctx, args[0]); err != nil {
				return err
			}

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(map[string]any{
					"deleted": true,
					"id":      args[0],
					"warning": warning,
				})
			}

			writeLine("Program %s deleted.", args[0])
			writeLine("Warning: %s", warning)
			return nil
		},
	}

	command.Flags().BoolVar(&dryRun, "dry-run", false, "Muestra la operacion y el aviso sin enviar peticiones")

	return command
}

func newProgramSetSpacesCommand(opts *rootOptions) *cobra.Command {
	var spaces []int
	var dryRun bool

	command := &cobra.Command{
		Use:   "set-spaces <program-id>",
		Short: "Reemplaza la seleccion de spaces de un programa",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}
			if len(spaces) == 0 {
				return fmt.Errorf("at least one --space is required")
			}

			payload := map[string]any{"selected": spaces}
			if dryRun {
				return output.PrintJSON(map[string]any{
					"action":     "program set-spaces",
					"program_id": args[0],
					"payload":    payload,
				})
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			assigned, err := rt.Client.UpdateProgramSpaces(ctx, args[0], spaces)
			if err != nil {
				return err
			}

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(assigned)
			}

			rows := make([][]string, 0, len(assigned))
			for _, item := range assigned {
				rows = append(rows, []string{
					intToString(anyInt(item["id"])),
					valueOrDash(mapString(item, "remote_id")),
					valueOrDash(mapString(item, "name")),
				})
			}
			return output.PrintTable([]string{"ID", "Remote ID", "Name"}, rows)
		},
	}

	command.Flags().IntSliceVar(&spaces, "space", nil, "IDs de spaces seleccionados")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Muestra el payload sin enviar peticiones")

	return command
}

func newProgramSetCoursesCommand(opts *rootOptions) *cobra.Command {
	var courseIDs []int
	var dryRun bool

	command := &cobra.Command{
		Use:   "set-courses <program-id>",
		Short: "Reemplaza la seleccion de courses asociados a un programa",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}
			if len(courseIDs) == 0 {
				return fmt.Errorf("at least one --course is required")
			}

			payload := map[string]any{"selected": courseIDs}
			if dryRun {
				return output.PrintJSON(map[string]any{
					"action":     "program set-courses",
					"program_id": args[0],
					"payload":    payload,
				})
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			courses, err := rt.Client.UpdateProgramCourses(ctx, args[0], payload)
			if err != nil {
				return err
			}

			return printProgramCoursesMutationResult(rt.Format, courses)
		},
	}

	command.Flags().IntSliceVar(&courseIDs, "course", nil, "IDs de courses que deben quedar asociados")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Muestra el payload sin enviar peticiones")

	return command
}

func newProgramReorderCoursesCommand(opts *rootOptions) *cobra.Command {
	var input jsonInputOptions
	var dryRun bool

	command := &cobra.Command{
		Use:   "reorder-courses <program-id>",
		Short: "Reordena los courses de un programa exigiendo un payload completo y consistente",
		Long: `
Lanza POST /course-program/{id}/course con un payload JSON que debe incluir:

- selected: lista completa de course IDs que deben quedar asociados
- order: mapa course_id -> posicion

Validaciones defensivas del CLI:
- selected y order son obligatorios
- ambos deben referirse exactamente al mismo conjunto de courses
- order debe cubrir todas las posiciones 1..N sin huecos ni duplicados
- selected debe venir ya ordenado segun el campo order
- antes de mutar, el CLI comprueba que el payload incluye exactamente los courses
  asociados ahora mismo al programa; si no coincide, falla sin enviar el POST

Ejemplo:
- hawkings program reorder-courses 5315 --json '{"selected":[33967,33968],"order":{"33967":1,"33968":2}}'
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			rawPayload, err := readJSONObject(input)
			if err != nil {
				return err
			}

			reorder, err := parseProgramCourseReorderPayload(rawPayload)
			if err != nil {
				return err
			}

			if dryRun {
				return output.PrintJSON(map[string]any{
					"action":     "program reorder-courses",
					"program_id": args[0],
					"endpoint":   "/course-program/" + args[0] + "/course",
					"payload":    reorder.Payload,
					"notes": []string{
						"dry-run valida el payload localmente pero no consulta el estado actual del programa",
						"la ejecucion real comprobara que selected coincide exactamente con los courses actuales antes de enviar el POST",
					},
				})
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			program, err := rt.Client.GetProgram(ctx, args[0], []string{"courses"})
			if err != nil {
				return err
			}

			currentCourseIDs := sortedProgramCourseIDs(program.Courses)
			if err := validateProgramCourseSetMatchesCurrent(currentCourseIDs, reorder.Selected); err != nil {
				return fmt.Errorf("reorder aborted: %w", err)
			}

			courses, err := rt.Client.UpdateProgramCourses(ctx, args[0], reorder.Payload)
			if err != nil {
				return err
			}

			if err := validateProgramCourseSetMatchesCurrent(sortedProgramCourseIDs(courses), reorder.Selected); err != nil {
				return fmt.Errorf("reorder finished with unexpected course set: %w", err)
			}

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(map[string]any{
					"action":            "program reorder-courses",
					"program_id":        args[0],
					"payload":           reorder.Payload,
					"validated_current": currentCourseIDs,
					"courses":           courses,
				})
			}

			return printProgramCoursesReorderResult(courses, reorder)
		},
	}

	addJSONInputFlags(command, &input)
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Muestra el payload validado sin enviar peticiones")

	return command
}

func newProgramAddCourseCommand(opts *rootOptions) *cobra.Command {
	var courseIDs []int
	var dryRun bool

	command := &cobra.Command{
		Use:   "add-course <program-id>",
		Short: "Anade uno o varios courses a un programa sin tocar los ya asociados",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}
			if len(courseIDs) == 0 {
				return fmt.Errorf("at least one --course is required")
			}

			payload := map[string]any{"add": courseIDs}
			if dryRun {
				return output.PrintJSON(map[string]any{
					"action":     "program add-course",
					"program_id": args[0],
					"payload":    payload,
				})
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			courses, err := rt.Client.UpdateProgramCourses(ctx, args[0], payload)
			if err != nil {
				return err
			}

			return printProgramCoursesMutationResult(rt.Format, courses)
		},
	}

	command.Flags().IntSliceVar(&courseIDs, "course", nil, "IDs de courses a anadir")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Muestra el payload sin enviar peticiones")

	return command
}

func newProgramRemoveCourseCommand(opts *rootOptions) *cobra.Command {
	var courseIDs []int
	var dryRun bool

	command := &cobra.Command{
		Use:   "remove-course <program-id>",
		Short: "Quita uno o varios courses de un programa sin tocar los demas",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}
			if len(courseIDs) == 0 {
				return fmt.Errorf("at least one --course is required")
			}

			payload := map[string]any{"remove": courseIDs}
			if dryRun {
				return output.PrintJSON(map[string]any{
					"action":     "program remove-course",
					"program_id": args[0],
					"payload":    payload,
				})
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			courses, err := rt.Client.UpdateProgramCourses(ctx, args[0], payload)
			if err != nil {
				return err
			}

			return printProgramCoursesMutationResult(rt.Format, courses)
		},
	}

	command.Flags().IntSliceVar(&courseIDs, "course", nil, "IDs de courses a quitar")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Muestra el payload sin enviar peticiones")

	return command
}

func printProgramCoursesMutationResult(format output.Format, courses []api.CourseDetail) error {
	if output.WantsJSON(format) {
		return output.PrintJSON(courses)
	}

	rows := make([][]string, 0, len(courses))
	for _, course := range courses {
		rows = append(rows, []string{
			intToString(course.ID),
			valueOrDash(stringPtrValue(course.RemoteID)),
			course.Name,
			valueOrDash(stringPtrValue(course.Status)),
		})
	}

	return output.PrintTable([]string{"ID", "Remote ID", "Name", "Status"}, rows)
}

func validatePayloadFields(commandName string, payload map[string]any, allowedFields []string) error {
	allowed := make(map[string]struct{}, len(allowedFields))
	for _, field := range allowedFields {
		allowed[field] = struct{}{}
	}

	unknown := make([]string, 0)
	for field := range payload {
		if _, ok := allowed[field]; !ok {
			unknown = append(unknown, field)
		}
	}
	if len(unknown) == 0 {
		return nil
	}

	sort.Strings(unknown)
	allowedSorted := append([]string(nil), allowedFields...)
	sort.Strings(allowedSorted)

	return fmt.Errorf(
		"%s JSON payload contains unsupported field(s): %s; allowed fields: %s",
		commandName,
		strings.Join(unknown, ", "),
		strings.Join(allowedSorted, ", "),
	)
}

func validateProgramPersistedPayloadFields(commandName string, payload map[string]any, program api.ProgramDetail) error {
	value, ok := payload["research_instructions"]
	if !ok {
		return nil
	}

	switch typed := value.(type) {
	case nil:
		if program.ResearchInstructions != nil {
			return fmt.Errorf("%s sent research_instructions=null but API returned %q", commandName, *program.ResearchInstructions)
		}
	case string:
		if program.ResearchInstructions == nil {
			return fmt.Errorf("%s sent research_instructions but API response did not include it; refusing to report a successful write that may have been ignored", commandName)
		}
		if *program.ResearchInstructions != typed {
			return fmt.Errorf("%s sent research_instructions=%q but API returned %q", commandName, typed, *program.ResearchInstructions)
		}
	}

	return nil
}

type programCourseReorderPayload struct {
	Selected []int
	Order    map[int]int
	Payload  map[string]any
}

func parseProgramCourseReorderPayload(payload map[string]any) (programCourseReorderPayload, error) {
	selected, err := parseRequiredIntList(payload, "selected")
	if err != nil {
		return programCourseReorderPayload{}, err
	}

	order, err := parseRequiredIntMap(payload, "order")
	if err != nil {
		return programCourseReorderPayload{}, err
	}

	selectedSet := make(map[int]struct{}, len(selected))
	for _, id := range selected {
		if _, exists := selectedSet[id]; exists {
			return programCourseReorderPayload{}, fmt.Errorf("selected contains duplicated course id %d", id)
		}
		selectedSet[id] = struct{}{}
	}

	if len(order) != len(selected) {
		return programCourseReorderPayload{}, fmt.Errorf("order must contain exactly %d entries, got %d", len(selected), len(order))
	}

	seenPositions := make(map[int]int, len(order))
	for courseID, position := range order {
		if _, exists := selectedSet[courseID]; !exists {
			return programCourseReorderPayload{}, fmt.Errorf("order contains course id %d that is not present in selected", courseID)
		}
		if previousCourseID, exists := seenPositions[position]; exists {
			return programCourseReorderPayload{}, fmt.Errorf("order position %d is duplicated for course ids %d and %d", position, previousCourseID, courseID)
		}
		seenPositions[position] = courseID
	}

	for expectedPosition := 1; expectedPosition <= len(selected); expectedPosition++ {
		if _, exists := seenPositions[expectedPosition]; !exists {
			return programCourseReorderPayload{}, fmt.Errorf("order must include every position from 1 to %d; missing %d", len(selected), expectedPosition)
		}
	}

	for index, courseID := range selected {
		expectedCourseID := seenPositions[index+1]
		if courseID != expectedCourseID {
			return programCourseReorderPayload{}, fmt.Errorf("selected must be sorted by order; expected course id %d at position %d, got %d", expectedCourseID, index+1, courseID)
		}
	}

	return programCourseReorderPayload{
		Selected: append([]int(nil), selected...),
		Order:    order,
		Payload: map[string]any{
			"selected": append([]int(nil), selected...),
			"order":    order,
		},
	}, nil
}

func parseRequiredIntList(payload map[string]any, field string) ([]int, error) {
	value, ok := payload[field]
	if !ok {
		return nil, fmt.Errorf("missing %q in JSON payload", field)
	}

	values, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("%q must be an array of integers", field)
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("%q must not be empty", field)
	}

	parsed := make([]int, 0, len(values))
	for index, item := range values {
		number, err := parsePositiveInt(item)
		if err != nil {
			return nil, fmt.Errorf("%s[%d]: %w", field, index, err)
		}
		parsed = append(parsed, number)
	}

	return parsed, nil
}

func parseRequiredIntMap(payload map[string]any, field string) (map[int]int, error) {
	value, ok := payload[field]
	if !ok {
		return nil, fmt.Errorf("missing %q in JSON payload", field)
	}

	values, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%q must be an object with integer course ids as keys", field)
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("%q must not be empty", field)
	}

	parsed := make(map[int]int, len(values))
	for rawKey, item := range values {
		courseID, err := parsePositiveInt(rawKey)
		if err != nil {
			return nil, fmt.Errorf("%s key %q: %w", field, rawKey, err)
		}

		position, err := parsePositiveInt(item)
		if err != nil {
			return nil, fmt.Errorf("%s[%q]: %w", field, rawKey, err)
		}

		parsed[courseID] = position
	}

	return parsed, nil
}

func parsePositiveInt(value any) (int, error) {
	switch v := value.(type) {
	case int:
		if v <= 0 {
			return 0, fmt.Errorf("must be a positive integer")
		}
		return v, nil
	case int32:
		if v <= 0 {
			return 0, fmt.Errorf("must be a positive integer")
		}
		return int(v), nil
	case int64:
		if v <= 0 {
			return 0, fmt.Errorf("must be a positive integer")
		}
		return int(v), nil
	case float64:
		if v <= 0 || float64(int(v)) != v {
			return 0, fmt.Errorf("must be a positive integer")
		}
		return int(v), nil
	case json.Number:
		if strings.Contains(v.String(), ".") {
			return 0, fmt.Errorf("must be a positive integer")
		}
		parsed, err := v.Int64()
		if err != nil || parsed <= 0 {
			return 0, fmt.Errorf("must be a positive integer")
		}
		return int(parsed), nil
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0, fmt.Errorf("must be a positive integer")
		}
		var parsed int
		if _, err := fmt.Sscanf(trimmed, "%d", &parsed); err != nil || intToString(parsed) != trimmed || parsed <= 0 {
			return 0, fmt.Errorf("must be a positive integer")
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("must be a positive integer")
	}
}

func sortedProgramCourseIDs(courses []api.CourseDetail) []int {
	ids := make([]int, 0, len(courses))
	for _, course := range courses {
		ids = append(ids, course.ID)
	}
	sort.Ints(ids)
	return ids
}

func validateProgramCourseSetMatchesCurrent(currentIDs []int, selectedIDs []int) error {
	currentSet := make(map[int]struct{}, len(currentIDs))
	for _, id := range currentIDs {
		currentSet[id] = struct{}{}
	}

	selectedSet := make(map[int]struct{}, len(selectedIDs))
	for _, id := range selectedIDs {
		selectedSet[id] = struct{}{}
	}

	missing := make([]int, 0)
	for _, id := range currentIDs {
		if _, exists := selectedSet[id]; !exists {
			missing = append(missing, id)
		}
	}

	extra := make([]int, 0)
	for _, id := range selectedIDs {
		if _, exists := currentSet[id]; !exists {
			extra = append(extra, id)
		}
	}

	if len(missing) == 0 && len(extra) == 0 {
		return nil
	}

	sort.Ints(missing)
	sort.Ints(extra)

	parts := make([]string, 0, 2)
	if len(missing) > 0 {
		parts = append(parts, fmt.Sprintf("missing current courses %v", missing))
	}
	if len(extra) > 0 {
		parts = append(parts, fmt.Sprintf("unexpected courses %v", extra))
	}

	return fmt.Errorf("payload selected must match the program's current courses exactly: %s", strings.Join(parts, "; "))
}

func printProgramCoursesReorderResult(courses []api.CourseDetail, reorder programCourseReorderPayload) error {
	courseByID := make(map[int]api.CourseDetail, len(courses))
	for _, course := range courses {
		courseByID[course.ID] = course
	}

	rows := make([][]string, 0, len(reorder.Selected))
	for _, courseID := range reorder.Selected {
		course := courseByID[courseID]
		rows = append(rows, []string{
			intToString(reorder.Order[courseID]),
			intToString(courseID),
			valueOrDash(stringPtrValue(course.RemoteID)),
			valueOrDash(course.Name),
			valueOrDash(stringPtrValue(course.Status)),
		})
	}

	return output.PrintTable([]string{"Order", "ID", "Remote ID", "Name", "Status"}, rows)
}

func newProgramGenerateSyllabusCommand(opts *rootOptions) *cobra.Command {
	var input jsonInputOptions
	var force bool
	var context string
	var syllabusPrompt string
	var dryRun bool

	command := &cobra.Command{
		Use:   "generate-syllabus <program-id>",
		Short: "Genera el syllabus usando el context actual o uno proporcionado",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			payload := map[string]any{}
			if input.JSON != "" || input.JSONFile != "" {
				payload, err = readJSONObject(input)
				if err != nil {
					return err
				}
				if err := validatePayloadFields("program generate-syllabus", payload, programGenerateSyllabusPayloadFields); err != nil {
					return err
				}
			}

			if force {
				payload["force"] = true
			}
			if context != "" {
				payload["context"] = context
			}
			if syllabusPrompt != "" {
				payload["syllabus_prompt"] = syllabusPrompt
			}
			if _, ok := payload["force"]; !ok {
				payload["force"] = false
			}

			if dryRun {
				return output.PrintJSON(map[string]any{
					"action":     "program generate-syllabus",
					"program_id": args[0],
					"payload":    payload,
				})
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			program, err := rt.Client.GenerateProgramSyllabus(ctx, args[0], payload)
			if err != nil {
				return err
			}
			program = normalizeProgramDetail(program)

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(program)
			}

			rows := [][]string{
				{"ID", intToString(program.ID)},
				{"Name", program.Name},
				{"Status", normalizedStatus(program.Status)},
				{"Has syllabus", boolToYesNo(programHasSyllabus(program))},
				{"Context chars", intToString(len(stringPtrValue(program.Context)))},
			}
			if err := output.PrintTable([]string{"Field", "Value"}, rows); err != nil {
				return err
			}
			return nil
		},
	}

	addJSONInputFlags(command, &input)
	command.Flags().BoolVar(&force, "force", false, "Regenera aunque ya exista syllabus")
	command.Flags().StringVar(&context, "context", "", "Sobrescribe el context usado para generar")
	command.Flags().StringVar(&syllabusPrompt, "syllabus-prompt", "", "Sobrescribe el syllabus_prompt de esta generacion")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Muestra el payload sin enviar peticiones")

	return command
}

func newProgramCreateCoursesCommand(opts *rootOptions) *cobra.Command {
	var force bool
	var dryRun bool

	command := &cobra.Command{
		Use:   "create-courses <program-id>",
		Short: "Crea los courses a partir del syllabus ya almacenado en el programa",
		Long: `
Lanza POST /course-program/{id}/syllabus/course.

El backend lee el syllabus ya guardado en el programa y crea automaticamente
los courses, sections y modules. No hace falta mandar el syllabus en el body.

Requisitos reales del backend:
- el programa debe tener syllabus
- el programa debe tener course_program_template_id
- el programa no debe tener courses ya creados, salvo que el backend admita force
- la operacion puede tardar minutos; usa un --timeout alto en programas grandes

Si recibes un 422 con algun campo interno como "type", el problema no suele ser
el body del comando, sino algun dato derivado del syllabus o de la template del programa.

Comportamiento operativo importante:
- un timeout del cliente no garantiza que el backend haya cancelado el trabajo
- el programa puede quedar temporalmente en status courses-creating
- si reintentas despues de un timeout, el backend puede responder 422 porque ya detecta courses creados
- tras un timeout, comprueba primero con program get o program courses antes de relanzar

Ejemplos:
- hawkings --timeout 300s program create-courses 5315
- hawkings program get 5315
- hawkings program courses 5315
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			payload := map[string]any{}
			if force {
				payload["force"] = true
			}

			if dryRun {
				return output.PrintJSON(map[string]any{
					"action":     "program create-courses",
					"program_id": args[0],
					"endpoint":   "/course-program/" + args[0] + "/syllabus/course",
					"payload":    payload,
				})
			}

			ctx, cancel := commandContextWithMinimum(rt, opts.timeout, 10*time.Minute)
			defer cancel()

			program, err := rt.Client.CreateProgramCoursesFromSyllabus(ctx, args[0], payload)
			if err != nil {
				return err
			}
			program = normalizeProgramDetail(program)

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(program)
			}

			rows := [][]string{
				{"ID", intToString(program.ID)},
				{"Name", program.Name},
				{"Status", normalizedStatus(program.Status)},
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

	command.Flags().BoolVar(&force, "force", false, "Pide al backend forzar la operacion si esa variante esta soportada")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Muestra la operacion sin enviar peticiones")

	return command
}
