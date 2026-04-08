package cli

import (
	"strings"

	"hawkings-cli/internal/output"

	"github.com/spf13/cobra"
)

func newScormCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scorm",
		Short: "Comandos sobre recursos SCORM",
	}
	cmd.AddCommand(newScormCreateCommand(opts))
	return cmd
}

func newScormCreateCommand(opts *rootOptions) *cobra.Command {
	var input jsonInputOptions
	var dryRun bool

	command := &cobra.Command{
		Use:   "create",
		Short: "Crea un recurso SCORM via /scorm",
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			payload, err := readJSONObject(input)
			if err != nil {
				return err
			}

			payload, ignoredFields := sanitizeScormPayload(payload)

			if dryRun {
				preview := map[string]any{
					"action":  "scorm create",
					"payload": payload,
				}
				if len(ignoredFields) > 0 {
					preview["ignored_fields"] = ignoredFields
				}
				return output.PrintJSON(preview)
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			result, err := rt.Client.CreateScorm(ctx, payload)
			if err != nil {
				return err
			}

			if output.WantsJSON(rt.Format) {
				response := map[string]any{
					"action": "scorm create",
					"result": result,
				}
				if len(ignoredFields) > 0 {
					response["ignored_fields"] = ignoredFields
				}
				return output.PrintJSON(response)
			}

			rows := [][]string{
				{"ID", valueOrDash(mapString(result, "id"))},
				{"Name", valueOrDash(mapString(result, "name"))},
				{"Status", valueOrDash(canonicalProgramStatus(mapString(result, "status")))},
			}
			if len(ignoredFields) > 0 {
				rows = append(rows, []string{"Ignored fields", strings.Join(ignoredFields, ", ")})
			}
			return output.PrintTable([]string{"Field", "Value"}, rows)
		},
	}

	addJSONInputFlags(command, &input)
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Muestra el payload saneado sin enviar peticiones")

	return command
}

func sanitizeScormPayload(payload map[string]any) (map[string]any, []string) {
	sanitized := cloneMap(payload)
	ignored := make([]string, 0, 2)
	for _, key := range []string{"user_id", "language_id"} {
		if _, ok := sanitized[key]; ok {
			delete(sanitized, key)
			ignored = append(ignored, key)
		}
	}
	return sanitized, ignored
}
