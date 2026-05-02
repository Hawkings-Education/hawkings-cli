package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"hawkings-cli/internal/output"

	"github.com/spf13/cobra"
)

const imageGenerationMinTimeout = 800 * time.Second

func newPromptCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prompt",
		Short: "Herramientas de prompts (image, ...)",
	}
	cmd.AddCommand(newPromptImageCommand(opts))
	return cmd
}

type promptImageOptions struct {
	instructions      string
	instructionsFinal string
	context           string
	language          string
	format            string
	service           string
	quality           string
	relatedTable      string
	relatedID         int
	dryRun            bool
}

func newPromptImageCommand(opts *rootOptions) *cobra.Command {
	imgOpts := &promptImageOptions{}

	command := &cobra.Command{
		Use:   "image",
		Short: "Genera una imagen via POST /prompt/tool/image",
		Long: strings.Join([]string{
			"Genera una imagen via POST /prompt/tool/image.",
			"",
			"Use --instructions para enviar instrucciones que se combinan con el prompt base de Hawkings.",
			"Use --instructions-final para enviar instrucciones exactas en instructions y marcar instructions_final=true, sin aplicar el prompt base de Hawkings.",
			"Debe indicar solo uno de los dos.",
		}, "\n"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validatePromptImageInstructions(imgOpts); err != nil {
				return err
			}
			if imgOpts.format != "" && imgOpts.format != "url" && imgOpts.format != "base64" {
				return fmt.Errorf("--format debe ser url o base64")
			}
			if imgOpts.service != "" && imgOpts.service != "openai" && imgOpts.service != "google" {
				return fmt.Errorf("--service debe ser openai o google")
			}
			if imgOpts.quality != "" && imgOpts.quality != "low" && imgOpts.quality != "high" {
				return fmt.Errorf("--quality debe ser low o high")
			}

			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			payload := promptImagePayload(imgOpts)

			if imgOpts.dryRun {
				return output.PrintJSON(map[string]any{
					"action":  "prompt image",
					"payload": payload,
				})
			}

			ctx, cancel := commandContextWithMinimum(rt, 0, imageGenerationMinTimeout)
			defer cancel()

			result, err := rt.Client.GenerateImage(ctx, payload)
			if err != nil {
				return err
			}

			imageURL := extractImageURL(result)

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(map[string]any{
					"action": "prompt image",
					"url":    imageURL,
					"raw":    result,
				})
			}

			writeLine("%s", imageURL)
			return nil
		},
	}

	flags := command.Flags()
	flags.StringVar(&imgOpts.instructions, "instructions", "", "Instrucciones para generar la imagen usando el prompt base de Hawkings")
	flags.StringVar(&imgOpts.instructionsFinal, "instructions-final", "", "Instrucciones exactas para enviar en instructions con instructions_final=true; no se aplica el prompt base de Hawkings")
	flags.StringVar(&imgOpts.context, "context", "", "Contexto adicional para la generacion")
	flags.StringVar(&imgOpts.language, "language", "", "Codigo de idioma (ej: es_ES)")
	flags.StringVar(&imgOpts.format, "format", "url", "Formato de respuesta: url o base64")
	flags.StringVar(&imgOpts.service, "service", "", "Servicio de IA: openai o google")
	flags.StringVar(&imgOpts.quality, "quality", "", "Calidad: low o high")
	flags.StringVar(&imgOpts.relatedTable, "related-table", "", "Tabla relacionada (ej: course_content)")
	flags.IntVar(&imgOpts.relatedID, "related-id", 0, "ID relacionado en related-table")
	flags.BoolVar(&imgOpts.dryRun, "dry-run", false, "Muestra el payload sin enviar la peticion")

	return command
}

func validatePromptImageInstructions(opts *promptImageOptions) error {
	hasInstructions := strings.TrimSpace(opts.instructions) != ""
	hasInstructionsFinal := strings.TrimSpace(opts.instructionsFinal) != ""

	if !hasInstructions && !hasInstructionsFinal {
		return fmt.Errorf("debe indicar --instructions o --instructions-final")
	}
	if hasInstructions && hasInstructionsFinal {
		return fmt.Errorf("use solo uno de --instructions o --instructions-final")
	}
	return nil
}

func promptImagePayload(opts *promptImageOptions) map[string]any {
	payload := map[string]any{}
	if strings.TrimSpace(opts.instructionsFinal) != "" {
		payload["instructions"] = opts.instructionsFinal
		payload["instructions_final"] = true
	} else {
		payload["instructions"] = opts.instructions
	}
	if opts.context != "" {
		payload["context"] = opts.context
	}
	if opts.language != "" {
		payload["language"] = opts.language
	}
	if opts.format != "" {
		payload["format"] = opts.format
	}
	if opts.service != "" {
		payload["service"] = opts.service
	}
	if opts.quality != "" {
		payload["quality"] = opts.quality
	}
	if opts.relatedTable != "" {
		payload["related_table"] = opts.relatedTable
	}
	if opts.relatedID > 0 {
		payload["related_id"] = opts.relatedID
	}
	return payload
}

func extractImageURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	if strings.HasPrefix(trimmed, "{") {
		var parsed map[string]any
		if err := json.Unmarshal([]byte(trimmed), &parsed); err == nil {
			for _, key := range []string{"url", "image_url"} {
				if value, ok := parsed[key].(string); ok && value != "" {
					return value
				}
			}
			if data, ok := parsed["data"].(map[string]any); ok {
				if value, ok := data["url"].(string); ok && value != "" {
					return value
				}
			}
		}
	}

	return trimmed
}
