package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"hawkings-cli/internal/api"
	"hawkings-cli/internal/output"

	"github.com/spf13/cobra"
)

func newProgramImageCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "image",
		Short: "Gestiona la imagen de portada de un program",
		Long: strings.Join([]string{
			"Gestiona la imagen de portada de un program.",
			"",
			"Use `program image generate` para pedir al backend que genere una portada.",
			"Use `program image upload` para subir manualmente un JPG o PNG como portada.",
		}, "\n"),
	}
	cmd.AddCommand(newProgramImageGenerateCommand(opts))
	cmd.AddCommand(newProgramImageUploadCommand(opts))
	return cmd
}

func newProgramImageGenerateCommand(opts *rootOptions) *cobra.Command {
	var force bool
	var dryRun bool

	command := &cobra.Command{
		Use:   "generate <program-id>",
		Short: "Genera la portada de un program con IA",
		Long:  "Genera la portada de un program via POST /course-program/{id}/image/generate. Use --force para regenerar aunque ya tenga imagen.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			if dryRun {
				return output.PrintJSON(map[string]any{
					"action":     "program image generate",
					"program_id": args[0],
					"endpoint":   "/course-program/{id}/image/generate",
					"payload":    map[string]any{"force": force},
				})
			}

			ctx, cancel := commandContextWithMinimum(rt, opts.timeout, imageGenerationMinTimeout)
			defer cancel()

			program, err := rt.Client.GenerateProgramImage(ctx, args[0], force)
			if err != nil {
				return err
			}
			program = normalizeProgramDetail(program)

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(program)
			}
			printImageResult("Program", program.ID, program.Name, stringPtrValue(program.Image))
			return nil
		},
	}

	command.Flags().BoolVar(&force, "force", false, "Regenera la imagen aunque el program ya tenga portada")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Muestra la operacion sin enviar la peticion")
	return command
}

func newProgramImageUploadCommand(opts *rootOptions) *cobra.Command {
	var filePath string
	var dryRun bool

	command := &cobra.Command{
		Use:   "upload <program-id> --file <cover.jpg|cover.png>",
		Short: "Sube manualmente un JPG o PNG como portada de un program",
		Long: strings.Join([]string{
			"Sube manualmente un JPG o PNG como portada de un program.",
			"",
			"El CLI lee primero el program actual para preservar sus campos y despues envia PATCH /course-program/{id} como multipart/form-data con el archivo en el campo image.",
			"Esto es distinto de `program image generate`, que pide al backend generar la portada con IA.",
		}, "\n"),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateImageFilePath(filePath); err != nil {
				return err
			}

			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			current, err := rt.Client.GetProgram(ctx, args[0], defaultProgramWith)
			if err != nil {
				return err
			}
			current = normalizeProgramDetail(current)
			fields := programImageUploadFields(current)

			if dryRun {
				return output.PrintJSON(map[string]any{
					"action":     "program image upload",
					"program_id": args[0],
					"endpoint":   "PATCH /course-program/{id}",
					"file":       filePath,
					"fields":     fields,
					"notes": []string{
						"dry-run no sube el archivo",
						"el archivo se enviara como multipart/form-data en el campo image",
					},
				})
			}

			program, err := rt.Client.UploadProgramImage(ctx, args[0], fields, filePath)
			if err != nil {
				return err
			}
			program = normalizeProgramDetail(program)

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(program)
			}
			printImageResult("Program", program.ID, program.Name, stringPtrValue(program.Image))
			return nil
		},
	}

	command.Flags().StringVar(&filePath, "file", "", "Ruta a la imagen JPG o PNG a subir manualmente")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Muestra la operacion sin subir el archivo")
	_ = command.MarkFlagRequired("file")
	return command
}

func newCourseImageCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "image",
		Short: "Gestiona la imagen de portada de un course",
		Long: strings.Join([]string{
			"Gestiona la imagen de portada de un course.",
			"",
			"Use `course image generate` para pedir al backend que genere una portada.",
			"Use `course image upload` para subir manualmente un JPG o PNG como portada.",
		}, "\n"),
	}
	cmd.AddCommand(newCourseImageGenerateCommand(opts))
	cmd.AddCommand(newCourseImageUploadCommand(opts))
	return cmd
}

func newCourseImageGenerateCommand(opts *rootOptions) *cobra.Command {
	var force bool
	var dryRun bool

	command := &cobra.Command{
		Use:   "generate <course-id>",
		Short: "Genera la portada de un course con IA",
		Long:  "Genera la portada de un course via POST /course/{id}/image/generate. Use --force para regenerar aunque ya tenga imagen.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			if dryRun {
				return output.PrintJSON(map[string]any{
					"action":    "course image generate",
					"course_id": args[0],
					"endpoint":  "/course/{id}/image/generate",
					"payload":   map[string]any{"force": force},
				})
			}

			ctx, cancel := commandContextWithMinimum(rt, opts.timeout, imageGenerationMinTimeout)
			defer cancel()

			course, err := rt.Client.GenerateCourseImage(ctx, args[0], force)
			if err != nil {
				return err
			}

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(course)
			}
			printImageResult("Course", course.ID, course.Name, stringPtrValue(course.Image))
			return nil
		},
	}

	command.Flags().BoolVar(&force, "force", false, "Regenera la imagen aunque el course ya tenga portada")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Muestra la operacion sin enviar la peticion")
	return command
}

func newCourseImageUploadCommand(opts *rootOptions) *cobra.Command {
	var filePath string
	var input jsonInputOptions
	var dryRun bool

	command := &cobra.Command{
		Use:   "upload <course-id> --file <cover.jpg|cover.png> --json-file <course-update.json>",
		Short: "Sube manualmente un JPG o PNG como portada de un course",
		Long: strings.Join([]string{
			"Sube manualmente un JPG o PNG como portada de un course.",
			"",
			"El backend recibe esta subida en PATCH /course/{id} como multipart/form-data con el archivo en el campo image.",
			"A diferencia de program, el PATCH normal de course puede tocar campos que `course get` no devuelve; por eso el CLI exige un payload JSON de actualizacion completo con --json o --json-file.",
			"Esto es distinto de `course image generate`, que pide al backend generar la portada con IA.",
		}, "\n"),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateImageFilePath(filePath); err != nil {
				return err
			}
			fields, err := readJSONObject(input)
			if err != nil {
				return err
			}
			if err := validateCourseImageUploadFields(fields); err != nil {
				return err
			}
			delete(fields, "image")
			fields["image_delete"] = false
			fields["image_generate"] = false

			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			if dryRun {
				return output.PrintJSON(map[string]any{
					"action":    "course image upload",
					"course_id": args[0],
					"endpoint":  "PATCH /course/{id}",
					"file":      filePath,
					"fields":    fields,
					"notes": []string{
						"dry-run no sube el archivo",
						"el archivo se enviara como multipart/form-data en el campo image",
						"incluye en el payload todos los campos del course que quieras preservar",
					},
				})
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			course, err := rt.Client.UploadCourseImage(ctx, args[0], fields, filePath)
			if err != nil {
				return err
			}

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(course)
			}
			printImageResult("Course", course.ID, course.Name, stringPtrValue(course.Image))
			return nil
		},
	}

	command.Flags().StringVar(&filePath, "file", "", "Ruta a la imagen JPG o PNG a subir manualmente")
	addJSONInputFlags(command, &input)
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Muestra la operacion sin subir el archivo")
	_ = command.MarkFlagRequired("file")
	return command
}

func validateImageFilePath(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("--file is required")
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat image file: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("image file cannot be a directory: %s", path)
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg", ".png":
		return nil
	default:
		return fmt.Errorf("image file must be .jpg, .jpeg or .png")
	}
}

func validateCourseImageUploadFields(fields map[string]any) error {
	name, hasName := fields["name"]
	if !hasName || name == nil || strings.TrimSpace(fmt.Sprintf("%v", name)) == "" {
		return fmt.Errorf("course image upload requires payload field name")
	}
	if anyInt(fields["language_id"]) == 0 {
		return fmt.Errorf("course image upload requires payload field language_id")
	}
	return nil
}

func programImageUploadFields(program api.ProgramDetail) map[string]any {
	fields := map[string]any{
		"name":                        program.Name,
		"remote_id":                   stringOrNil(program.RemoteID),
		"status":                      stringOrNil(program.Status),
		"metadata":                    program.Metadata,
		"syllabus":                    program.Syllabus,
		"syllabus_prompt":             stringOrNil(program.SyllabusPrompt),
		"context":                     stringOrNil(program.Context),
		"course_module_prompt_custom": stringOrNil(program.CourseModulePromptCustom),
		"research_instructions":       stringOrNil(program.ResearchInstructions),
		"enabled":                     program.Enabled,
		"image_delete":                false,
		"image_generate":              false,
	}
	if program.Language != nil && program.Language.ID > 0 {
		fields["language_id"] = program.Language.ID
	}
	if id := anyInt(program.CourseFaculty["id"]); id > 0 {
		fields["course_faculty_id"] = id
	}
	if id := anyInt(program.CourseProgramTemplate["id"]); id > 0 {
		fields["course_program_template_id"] = id
	}
	return fields
}

func printImageResult(kind string, id int, name string, imageURL string) {
	rows := [][]string{
		{"Type", kind},
		{"ID", intToString(id)},
		{"Name", name},
		{"Image", valueOrDash(imageURL)},
	}
	_ = output.PrintTable([]string{"Field", "Value"}, rows)
}
