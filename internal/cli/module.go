package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"hawkings-cli/internal/api"
	"hawkings-cli/internal/output"

	"github.com/spf13/cobra"
)

func newModuleCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "module",
		Short: "Comandos sobre course modules",
	}
	cmd.AddCommand(newModuleGetCommand(opts))
	cmd.AddCommand(newModuleContentCommand(opts))
	cmd.AddCommand(newModuleCreateCommand(opts))
	cmd.AddCommand(newModuleSetContentCommand(opts))
	cmd.AddCommand(newModulePatchCommand(opts))
	cmd.AddCommand(newModuleGenerateContentCommand(opts))
	cmd.AddCommand(newModuleApproveCommand(opts))
	return cmd
}

func newModuleGetCommand(opts *rootOptions) *cobra.Command {
	var contents bool

	command := &cobra.Command{
		Use:   "get <module-id>",
		Short: "Muestra un module y lista sus contents sin traer el cuerpo completo por defecto",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			module, err := rt.Client.GetCourseModule(ctx, args[0], []string{"courseContents"}, contents)
			if err != nil {
				return err
			}

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(module)
			}

			rows := [][]string{
				{"ID", intToString(module.ID)},
				{"Name", module.Name},
				{"Type", module.Type},
				{"Status", stringPtrOrDash(module.Status)},
				{"Order", intToString(module.Order)},
				{"Approved At", stringPtrOrDash(module.ApprovedAt)},
				{"Contents", intToString(len(module.CourseContents))},
			}
			if err := output.PrintTable([]string{"Field", "Value"}, rows); err != nil {
				return err
			}

			if len(module.CourseContents) == 0 {
				writeLine("")
				writeLine("El module no tiene course contents.")
				return nil
			}

			writeLine("")
			contentRows := make([][]string, 0, len(module.CourseContents))
			for _, content := range module.CourseContents {
				contentRows = append(contentRows, []string{
					intToString(content.ID),
					content.Type,
					content.Name,
					valueOrDash(courseContentFileString(content, "mime")),
					intToString(courseContentFileInt(content, "size")),
				})
			}
			if err := output.PrintTable([]string{"ID", "Type", "Name", "Mime", "Size"}, contentRows); err != nil {
				return err
			}

			if contents {
				writeLine("")
				writeLine("Nota: has pedido file.contents completo. Usa `module content %d` si solo quieres un fragmento controlado.", module.ID)
			}
			return nil
		},
	}

	command.Flags().BoolVar(&contents, "contents", false, "Incluye file.contents completo; puede devolver mucho texto")
	return command
}

func newModuleContentCommand(opts *rootOptions) *cobra.Command {
	var contentID int
	var maxChars int
	var full bool
	var raw bool

	command := &cobra.Command{
		Use:   "content <module-id>",
		Short: "Devuelve el contenido de un module con truncado por defecto",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			module, err := rt.Client.GetCourseModule(ctx, args[0], []string{"courseContents"}, true)
			if err != nil {
				return err
			}

			if len(module.CourseContents) == 0 {
				if raw {
					return nil
				}
				payload := map[string]any{
					"module": map[string]any{
						"id":      module.ID,
						"name":    module.Name,
						"type":    module.Type,
						"status":  normalizedStatus(module.Status),
						"content": nil,
					},
					"error": "module has no course contents",
				}
				if output.WantsJSON(rt.Format) {
					return output.PrintJSON(payload)
				}
				writeLine("El module %d no tiene course contents.", module.ID)
				return nil
			}

			content, err := selectCourseContent(module.CourseContents, contentID)
			if err != nil {
				return err
			}

			body := courseContentBody(content)
			snippet := body
			truncated := false
			totalChars := utf8.RuneCountInString(body)
			if !full {
				snippet, truncated, totalChars = truncateText(body, maxChars)
			}

			contentInventory := make([]map[string]any, 0, len(module.CourseContents))
			for _, item := range module.CourseContents {
				contentInventory = append(contentInventory, map[string]any{
					"id":   item.ID,
					"name": item.Name,
					"type": item.Type,
					"file": map[string]any{
						"id":   courseContentFileInt(item, "id"),
						"mime": courseContentFileString(item, "mime"),
						"size": courseContentFileInt(item, "size"),
					},
				})
			}

			payload := map[string]any{
				"hierarchy": "module -> content -> file.contents",
				"module": map[string]any{
					"id":             module.ID,
					"name":           module.Name,
					"type":           module.Type,
					"status":         normalizedStatus(module.Status),
					"contents_count": len(module.CourseContents),
				},
				"available_contents": contentInventory,
				"selected_content": map[string]any{
					"id":             content.ID,
					"name":           content.Name,
					"type":           content.Type,
					"file_id":        courseContentFileInt(content, "id"),
					"mime":           courseContentFileString(content, "mime"),
					"size":           courseContentFileInt(content, "size"),
					"returned_chars": utf8.RuneCountInString(snippet),
					"total_chars":    totalChars,
					"truncated":      truncated,
					"contents":       snippet,
				},
				"notes": []string{
					"module content trunca por defecto para no saturar el contexto.",
					"Usa --full para devolver el cuerpo completo o --content-id para elegir otro content.",
				},
			}

			if raw {
				if snippet != "" {
					fmt.Fprint(os.Stdout, snippet)
					if !strings.HasSuffix(snippet, "\n") {
						fmt.Fprint(os.Stdout, "\n")
					}
				}
				if truncated {
					fmt.Fprintf(os.Stdout, "\n[truncated at %d chars; rerun with --full or increase --max-chars]\n", maxChars)
				}
				return nil
			}

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(payload)
			}

			rows := [][]string{
				{"Module ID", intToString(module.ID)},
				{"Module Name", module.Name},
				{"Module Type", module.Type},
				{"Module Status", normalizedStatus(module.Status)},
				{"Content ID", intToString(content.ID)},
				{"Content Name", content.Name},
				{"Content Type", content.Type},
				{"Mime", valueOrDash(courseContentFileString(content, "mime"))},
				{"Size", intToString(courseContentFileInt(content, "size"))},
				{"Returned chars", intToString(utf8.RuneCountInString(snippet))},
				{"Total chars", intToString(totalChars)},
				{"Truncated", boolToYesNo(truncated)},
			}
			if err := output.PrintTable([]string{"Field", "Value"}, rows); err != nil {
				return err
			}

			if len(module.CourseContents) > 1 {
				writeLine("")
				inventoryRows := make([][]string, 0, len(module.CourseContents))
				for _, item := range module.CourseContents {
					inventoryRows = append(inventoryRows, []string{
						intToString(item.ID),
						item.Type,
						item.Name,
						valueOrDash(courseContentFileString(item, "mime")),
						intToString(courseContentFileInt(item, "size")),
					})
				}
				if err := output.PrintTable([]string{"ID", "Type", "Name", "Mime", "Size"}, inventoryRows); err != nil {
					return err
				}
			}

			if body == "" {
				writeLine("")
				writeLine("El content seleccionado no trae file.contents en la respuesta.")
				return nil
			}

			writeLine("")
			writeLine("Content:")
			writeLine("%s", snippet)
			if truncated {
				writeLine("")
				writeLine("Truncated at %d chars. Repite con --full o sube --max-chars.", maxChars)
			}
			return nil
		},
	}

	command.Flags().IntVar(&contentID, "content-id", 0, "Selecciona un content concreto por ID")
	command.Flags().IntVar(&maxChars, "max-chars", 1000, "Maximo de caracteres devueltos cuando no se usa --full")
	command.Flags().BoolVar(&full, "full", false, "Devuelve el contenido completo sin truncado")
	command.Flags().BoolVar(&raw, "raw", false, "Imprime solo el cuerpo de texto")
	return command
}

func newModuleCreateCommand(opts *rootOptions) *cobra.Command {
	var moduleType string
	var name string
	var url string
	var status string
	var courseID int
	var sectionID int
	var order int
	var position string
	var optional bool
	var evaluable bool
	var remoteID string
	var remoteUpdatedAt string
	var metadataInput jsonInputOptions
	var dryRun bool

	command := &cobra.Command{
		Use:   "create",
		Short: "Crea un module a nivel course o section con calculo automatico de order si no se indica",
		Long: strings.TrimSpace(`
Crea un module nuevo via POST /course-module.

Reglas:
- Usa --course-id para modulos a nivel curso.
- Usa --section-id para modulos dentro de una section; el CLI resuelve automaticamente el course_id.
- Si no pasas --order, el CLI calcula max(order)+1 dentro del ambito elegido.
- Si pasas --order y ya existe, el backend desplaza los modulos siguientes.
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			if strings.TrimSpace(name) == "" {
				return fmt.Errorf("--name is required")
			}
			moduleType = strings.TrimSpace(moduleType)
			if moduleType == "" {
				return fmt.Errorf("--type is required")
			}
			if courseID == 0 && sectionID == 0 {
				return fmt.Errorf("use --course-id or --section-id")
			}
			if courseID != 0 && sectionID != 0 {
				return fmt.Errorf("use either --course-id or --section-id, not both")
			}

			metadata, err := readOptionalJSONObject(metadataInput)
			if err != nil {
				return err
			}

			resolvedCourseID := courseID
			resolvedOrder := order
			scope := "course"
			parentID := courseID

			payload := map[string]any{
				"name":      strings.TrimSpace(name),
				"type":      moduleType,
				"url":       strings.TrimSpace(url),
				"status":    defaultString(status, "empty"),
				"order":     resolvedOrder,
				"optional":  optional,
				"evaluable": evaluable,
				"course_id": resolvedCourseID,
			}
			if sectionID != 0 {
				payload["course_section_id"] = sectionID
			}
			if strings.TrimSpace(position) != "" {
				payload["position"] = strings.TrimSpace(position)
			}
			if strings.TrimSpace(remoteID) != "" {
				payload["remote_id"] = strings.TrimSpace(remoteID)
			}
			if strings.TrimSpace(remoteUpdatedAt) != "" {
				payload["remote_updated_at"] = strings.TrimSpace(remoteUpdatedAt)
			}
			if metadata != nil {
				payload["metadata"] = metadata
			}

			if dryRun {
				if sectionID != 0 {
					scope = "section"
					parentID = sectionID
					delete(payload, "course_id")
					payload["course_section_id"] = sectionID
				}
				return output.PrintJSON(map[string]any{
					"action":    "module create",
					"scope":     scope,
					"parent_id": parentID,
					"resolved": map[string]any{
						"course_id": anyResolvedValue(scope == "course", resolvedCourseID, "from section at runtime"),
						"order":     anyResolvedValue(resolvedOrder != 0, resolvedOrder, "auto (next order in selected scope)"),
					},
					"payload": payload,
					"notes": []string{
						"dry-run no hace lecturas ni escrituras en la API",
						"si order no se indica, el CLI calcula max(order)+1 en el ambito elegido durante la ejecucion real",
						"si el backend recibe un order ya ocupado, desplaza los modulos siguientes",
					},
				})
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			if sectionID != 0 {
				scope = "section"
				parentID = sectionID
				section, err := rt.Client.GetCourseSection(ctx, intToString(sectionID), []string{"courseModules", "course"})
				if err != nil {
					return err
				}
				resolvedCourseID = anyInt(section.Course["id"])
				if resolvedCourseID == 0 {
					return fmt.Errorf("section %d does not expose course.id; cannot infer --course-id", sectionID)
				}
				if resolvedOrder == 0 {
					resolvedOrder = nextModuleOrder(section.CourseModules)
				}
			} else if resolvedOrder == 0 {
				course, err := rt.Client.GetCourse(ctx, intToString(courseID), []string{"courseModules"})
				if err != nil {
					return err
				}
				resolvedOrder = nextModuleOrder(course.CourseModules)
			}

			payload["course_id"] = resolvedCourseID
			payload["order"] = resolvedOrder

			module, err := rt.Client.CreateCourseModule(ctx, payload)
			if err != nil {
				return err
			}

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(map[string]any{
					"action":        "module create",
					"scope":         scope,
					"parent_id":     parentID,
					"resolved_order": resolvedOrder,
					"module":        module,
				})
			}

			rows := [][]string{
				{"ID", intToString(module.ID)},
				{"Name", module.Name},
				{"Type", module.Type},
				{"Status", stringPtrOrDash(module.Status)},
				{"Order", intToString(module.Order)},
				{"Scope", scope},
				{"Parent ID", intToString(parentID)},
				{"Course ID", intToString(resolvedCourseID)},
			}
			if err := output.PrintTable([]string{"Field", "Value"}, rows); err != nil {
				return err
			}
			writeLine("")
			writeLine("Nota: si luego quieres escribir markdown manual, usa `module set-content %d --file ...`.", module.ID)
			return nil
		},
	}

	command.Flags().StringVar(&name, "name", "", "Nombre visible del module")
	command.Flags().StringVar(&moduleType, "type", "", "Tipo de module: markdown, activity, assignment o url")
	command.Flags().StringVar(&url, "url", "", "URL del module cuando el tipo lo requiera")
	command.Flags().StringVar(&status, "status", "empty", "Status inicial del module")
	command.Flags().IntVar(&courseID, "course-id", 0, "ID del course para modulos a nivel curso")
	command.Flags().IntVar(&sectionID, "section-id", 0, "ID de la section para modulos anidados; el CLI resuelve course_id")
	command.Flags().IntVar(&order, "order", 0, "Posicion deseada; si se omite, el CLI calcula el siguiente order")
	command.Flags().StringVar(&position, "position", "", "Posicion logica before|after si quieres persistirla en el recurso")
	command.Flags().BoolVar(&optional, "optional", false, "Marca el module como opcional")
	command.Flags().BoolVar(&evaluable, "evaluable", false, "Marca el module como evaluable")
	command.Flags().StringVar(&remoteID, "remote-id", "", "remote_id opcional")
	command.Flags().StringVar(&remoteUpdatedAt, "remote-updated-at", "", "Timestamp remoto en formato YYYY-MM-DD HH:MM:SS")
	command.Flags().StringVar(&metadataInput.JSON, "metadata-json", "", "Objeto JSON inline para metadata")
	command.Flags().StringVar(&metadataInput.JSONFile, "metadata-file", "", "Ruta a un fichero JSON para metadata")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Muestra el payload resuelto sin enviar peticiones")

	return command
}

func newModuleSetContentCommand(opts *rootOptions) *cobra.Command {
	var filePath string
	var inlineContent string
	var name string
	var mime string
	var contentStatus string
	var moduleStatus string
	var contentID int
	var dryRun bool

	command := &cobra.Command{
		Use:   "set-content <module-id>",
		Short: "Escribe contenido manual en un module sin usar el generador de IA del modulo",
		Long: strings.TrimSpace(`
Escribe contenido markdown directamente en el course-content asociado al module.

Flujo:
- Si el module no tiene contents, crea un course-content nuevo.
- Si ya tiene contents, actualiza el primero o el indicado con --content-id.
- Despues, por defecto hace PATCH /course-module/{id}/only con status=processed.

Importante:
- Este comando evita POST /course-module/{id}/course-content/generate.
- El backend puede seguir calculando summary y metadatos derivados del content.
`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			body, source, err := readModuleContentInput(filePath, inlineContent)
			if err != nil {
				return err
			}

			if dryRun {
				return output.PrintJSON(map[string]any{
					"action":      "module set-content",
					"module_id":   args[0],
					"content_id":  contentID,
					"input":       source,
					"name":        name,
					"mime":        resolvedModuleContentMime(mime, filePath),
					"content": map[string]any{
						"type":    "markdown",
						"status":  defaultString(contentStatus, "processed"),
						"chars":   utf8.RuneCountInString(body),
						"preview": previewText(body, 200),
						"mode":    "auto (create if no content exists, otherwise update selected content)",
					},
					"module_patch": map[string]any{
						"status": moduleStatus,
					},
					"notes": []string{
						"dry-run no hace lecturas ni escrituras en la API",
						"el backend puede seguir generando summary y metadatos del content",
					},
				})
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			module, err := rt.Client.GetCourseModule(ctx, args[0], []string{"courseContents"}, false)
			if err != nil {
				return err
			}

			selected, hasSelected, err := selectCourseContentIfAny(module.CourseContents, contentID)
			if err != nil {
				return err
			}

			targetName := strings.TrimSpace(name)
			if targetName == "" {
				if hasSelected && strings.TrimSpace(selected.Name) != "" {
					targetName = selected.Name
				} else {
					targetName = module.Name
				}
			}

			targetMime := resolvedModuleContentMime(mime, filePath)
			if hasSelected && strings.TrimSpace(targetMime) == "" {
				targetMime = selected.Mime
			}

			targetContentStatus := defaultString(contentStatus, "processed")
			if hasSelected && targetContentStatus == "processed" && strings.TrimSpace(selected.Status) != "" {
				targetContentStatus = selected.Status
			}

			contentPayload := map[string]any{
				"name":             targetName,
				"type":             "markdown",
				"content":          body,
				"mime":             targetMime,
				"status":           targetContentStatus,
				"remote":           false,
				"course_module_id": module.ID,
				"summary_sync":     false,
				"image_generate":   false,
			}

			var content api.CourseContent
			contentAction := "create"
			if hasSelected {
				contentAction = "update"
				current, err := rt.Client.GetCourseContent(ctx, intToString(selected.ID), false)
				if err != nil {
					return err
				}
				contentPayload["name"] = firstNonEmpty(targetName, current.Name, module.Name)
				contentPayload["type"] = firstNonEmpty(current.Type, "markdown")
				contentPayload["mime"] = firstNonEmpty(targetMime, current.Mime, "text/markdown")
				contentPayload["status"] = firstNonEmpty(targetContentStatus, current.Status, "processed")
				contentPayload["url"] = current.URL
				contentPayload["remote_id"] = current.RemoteID
				contentPayload["remote_updated_at"] = current.RemoteUpdatedAt

				content, err = rt.Client.UpdateCourseContent(ctx, intToString(selected.ID), contentPayload)
				if err != nil {
					return err
				}
			} else {
				contentPayload["mime"] = firstNonEmpty(targetMime, "text/markdown")
				content, err = rt.Client.CreateCourseContent(ctx, contentPayload)
				if err != nil {
					return err
				}
			}

			var updatedModule *api.CourseModule
			if strings.TrimSpace(moduleStatus) != "" {
				patched, err := rt.Client.UpdateCourseModuleOnly(ctx, args[0], map[string]any{
					"status": moduleStatus,
				})
				if err != nil {
					return err
				}
				updatedModule = &patched
			}

			payload := map[string]any{
				"action":         "module set-content",
				"content_action": contentAction,
				"source":         source,
				"module": map[string]any{
					"id":             module.ID,
					"name":           module.Name,
					"type":           module.Type,
					"previous_status": normalizedStatus(module.Status),
				},
				"content": map[string]any{
					"id":     content.ID,
					"name":   content.Name,
					"type":   content.Type,
					"mime":   content.Mime,
					"status": content.Status,
				},
				"notes": []string{
					"este comando no usa el endpoint de generacion de contenido del module",
					"el backend puede seguir calculando summary y metadatos derivados",
				},
			}
			if updatedModule != nil {
				payload["module_after_patch"] = map[string]any{
					"id":     updatedModule.ID,
					"status": normalizedStatus(updatedModule.Status),
				}
			}

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(payload)
			}

			rows := [][]string{
				{"Module ID", intToString(module.ID)},
				{"Module Name", module.Name},
				{"Module Type", module.Type},
				{"Previous Status", normalizedStatus(module.Status)},
				{"Content Action", contentAction},
				{"Content ID", intToString(content.ID)},
				{"Content Name", content.Name},
				{"Content Type", content.Type},
				{"Content Mime", valueOrDash(content.Mime)},
				{"Content Status", valueOrDash(content.Status)},
				{"Chars Written", intToString(utf8.RuneCountInString(body))},
			}
			if updatedModule != nil {
				rows = append(rows, []string{"Module Status After Patch", normalizedStatus(updatedModule.Status)})
			}
			if err := output.PrintTable([]string{"Field", "Value"}, rows); err != nil {
				return err
			}
			writeLine("")
			writeLine("Nota: el backend puede seguir calculando summary y metadatos del content.")
			return nil
		},
	}

	command.Flags().StringVar(&filePath, "file", "", "Ruta a un fichero de texto o markdown para cargar como contenido")
	command.Flags().StringVar(&filePath, "content-file", "", "Alias explicito de --file para cargar contenido desde fichero")
	command.Flags().StringVar(&inlineContent, "content", "", "Contenido inline para escribir directamente en el module")
	command.Flags().StringVar(&name, "name", "", "Nombre del course-content; por defecto usa el del module")
	command.Flags().StringVar(&mime, "mime", "", "Mime del course-content; por defecto text/markdown para .md")
	command.Flags().StringVar(&contentStatus, "content-status", "processed", "Status del course-content creado o actualizado")
	command.Flags().StringVar(&moduleStatus, "module-status", "processed", "Status a aplicar despues al module via PATCH /only; usa cadena vacia para omitirlo")
	command.Flags().IntVar(&contentID, "content-id", 0, "Actualiza un course-content concreto si el modulo tiene varios")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Muestra la operacion sin enviar peticiones")

	return command
}

func newModulePatchCommand(opts *rootOptions) *cobra.Command {
	var input jsonInputOptions
	var dryRun bool

	command := &cobra.Command{
		Use:     "patch <module-id>",
		Aliases: []string{"update"},
		Short:   "Hace PATCH /only sobre un module con los campos enviados",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			payload, err := readJSONObject(input)
			if err != nil {
				return err
			}

			if dryRun {
				return output.PrintJSON(map[string]any{
					"action":    "module patch",
					"module_id": args[0],
					"payload":   payload,
				})
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			module, err := rt.Client.UpdateCourseModuleOnly(ctx, args[0], payload)
			if err != nil {
				return err
			}

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(module)
			}

			rows := [][]string{
				{"ID", intToString(module.ID)},
				{"Name", module.Name},
				{"Type", module.Type},
				{"Status", stringPtrOrDash(module.Status)},
				{"Order", intToString(module.Order)},
				{"Approved At", stringPtrOrDash(module.ApprovedAt)},
			}
			return output.PrintTable([]string{"Field", "Value"}, rows)
		},
	}

	addJSONInputFlags(command, &input)
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Muestra el payload sin enviar peticiones")

	return command
}

func newModuleGenerateContentCommand(opts *rootOptions) *cobra.Command {
	var researchEnabled bool
	var researchProvider string
	var researchQuality string
	var researchInstructions string
	var researchIDs []int
	var promptCustom string
	var dryRun bool

	command := &cobra.Command{
		Use:   "generate-content <module-id>",
		Short: "Lanza la generacion asincrona del contenido de un module",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			payload := moduleGenerationPayload(researchEnabled, researchProvider, researchQuality, researchInstructions, researchIDs, promptCustom)

			if dryRun {
				return output.PrintJSON(map[string]any{
					"action":    "module generate-content",
					"module_id": args[0],
					"payload":   payload,
				})
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			result, err := rt.Client.GenerateCourseModuleContent(ctx, args[0], payload)
			if err != nil {
				return err
			}

			return output.PrintJSON(result)
		},
	}

	command.Flags().BoolVar(&researchEnabled, "research-enabled", false, "Activa research para la generacion")
	command.Flags().StringVar(&researchProvider, "research-provider", "", "Proveedor de research: Parallel o Perplexity")
	command.Flags().StringVar(&researchQuality, "research-quality", "", "Calidad de research: high, medium o fast")
	command.Flags().StringVar(&researchInstructions, "research-instructions", "", "Instrucciones especificas para el research")
	command.Flags().IntSliceVar(&researchIDs, "research-id", nil, "IDs de research existentes a reutilizar")
	command.Flags().StringVar(&promptCustom, "prompt-custom", "", "Instrucciones de redaccion para este modulo")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Muestra el payload sin enviar peticiones")

	return command
}

func newModuleApproveCommand(opts *rootOptions) *cobra.Command {
	var approved bool
	var dryRun bool

	command := &cobra.Command{
		Use:   "approve <module-id>",
		Short: "Aprueba o desaprueba un module via approved_at",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			if dryRun {
				return output.PrintJSON(map[string]any{
					"action":    "module approve",
					"module_id": args[0],
					"approved":  approved,
				})
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			module, err := approveModuleContent(ctx, rt.Client, args[0], approved)
			if err != nil {
				return err
			}

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(module)
			}

			rows := [][]string{
				{"ID", intToString(module.ID)},
				{"Name", module.Name},
				{"Status", stringPtrOrDash(module.Status)},
				{"Approved", boolToYesNo(module.ApprovedAt != nil)},
				{"Approved At", stringPtrOrDash(module.ApprovedAt)},
			}
			return output.PrintTable([]string{"Field", "Value"}, rows)
		},
	}

	command.Flags().BoolVar(&approved, "approved", true, "Marca el module como aprobado; usa --approved=false para desaprobar")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Muestra la operacion sin enviar peticiones")

	return command
}

func approveModuleContent(ctx context.Context, client *api.Client, id string, approved bool) (api.CourseModule, error) {
	if _, err := client.ApproveCourseModule(ctx, id, approved); err != nil {
		return api.CourseModule{}, err
	}
	return client.GetCourseModule(ctx, id, []string{"courseContents"}, false)
}

func selectCourseContent(contents []api.CourseContent, contentID int) (api.CourseContent, error) {
	if len(contents) == 0 {
		return api.CourseContent{}, fmt.Errorf("module has no course contents")
	}
	if contentID == 0 {
		return contents[0], nil
	}
	for _, content := range contents {
		if content.ID == contentID {
			return content, nil
		}
	}
	return api.CourseContent{}, fmt.Errorf("content %d not found in module", contentID)
}

func selectCourseContentIfAny(contents []api.CourseContent, contentID int) (api.CourseContent, bool, error) {
	if len(contents) == 0 {
		if contentID != 0 {
			return api.CourseContent{}, false, fmt.Errorf("content %d not found in module", contentID)
		}
		return api.CourseContent{}, false, nil
	}
	content, err := selectCourseContent(contents, contentID)
	if err != nil {
		return api.CourseContent{}, false, err
	}
	return content, true, nil
}

func readModuleContentInput(filePath, inlineContent string) (string, string, error) {
	if strings.TrimSpace(filePath) != "" && strings.TrimSpace(inlineContent) != "" {
		return "", "", fmt.Errorf("use either --file or --content, not both")
	}
	if strings.TrimSpace(filePath) == "" && strings.TrimSpace(inlineContent) == "" {
		return "", "", fmt.Errorf("missing content input; use --file or --content")
	}
	if strings.TrimSpace(filePath) != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", "", fmt.Errorf("read %s: %w", filePath, err)
		}
		return string(data), "file:" + filePath, nil
	}
	return inlineContent, "inline", nil
}

func resolvedModuleContentMime(explicit, filePath string) string {
	if strings.TrimSpace(explicit) != "" {
		return strings.TrimSpace(explicit)
	}
	if strings.EqualFold(filepath.Ext(filePath), ".md") {
		return "text/markdown"
	}
	if strings.TrimSpace(filePath) != "" {
		return "text/plain"
	}
	return "text/markdown"
}

func defaultString(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func previewText(value string, maxChars int) string {
	preview, _, _ := truncateText(value, maxChars)
	return preview
}

func nextModuleOrder(modules []api.CourseModule) int {
	maxOrder := 0
	for _, module := range modules {
		if module.Order > maxOrder {
			maxOrder = module.Order
		}
	}
	return maxOrder + 1
}
