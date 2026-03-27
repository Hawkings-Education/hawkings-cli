package cli

import (
	"fmt"

	"hawkings-cli/internal/api"
	"hawkings-cli/internal/output"

	"github.com/spf13/cobra"
)

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

			ctx, cancel := commandContext(rt)
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
	var dryRun bool

	command := &cobra.Command{
		Use:   "create-courses <program-id>",
		Short: "Crea los courses a partir del syllabus ya almacenado en el programa",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			if dryRun {
				return output.PrintJSON(map[string]any{
					"action":     "program create-courses",
					"program_id": args[0],
					"endpoint":   "/course-program/" + args[0] + "/syllabus/course",
					"payload":    map[string]any{},
				})
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			program, err := rt.Client.CreateProgramCoursesFromSyllabus(ctx, args[0])
			if err != nil {
				return err
			}

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

	command.Flags().BoolVar(&dryRun, "dry-run", false, "Muestra la operacion sin enviar peticiones")

	return command
}
