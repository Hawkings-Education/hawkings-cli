package cli

import (
	"fmt"
	"strconv"

	"hawkings-cli/internal/output"

	"github.com/spf13/cobra"
)

func newContentCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "content",
		Short: "Comandos semanticos sobre el contenido de un module",
	}
	cmd.AddCommand(newContentApproveCommand(opts))
	cmd.AddCommand(newContentDeleteCommand(opts))
	return cmd
}

func newContentApproveCommand(opts *rootOptions) *cobra.Command {
	var approved bool
	var dryRun bool

	command := &cobra.Command{
		Use:   "approve <module-id>",
		Short: "Aprueba o desaprueba el contenido de un module via approved_at",
		Long:  "Alias semantico del endpoint de aprobacion del module. En Hawkings, la aprobacion del contenido vive en approved_at del course-module, no en course-content.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			if dryRun {
				return output.PrintJSON(map[string]any{
					"action":    "content approve",
					"module_id": args[0],
					"approved":  approved,
					"endpoint":  "PATCH /course-module/{id}/boolean/approved_at",
				})
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			module, err := approveModuleContent(ctx, rt.Client, args[0], approved)
			if err != nil {
				return err
			}

			payload := map[string]any{
				"action": "content approve",
				"module": map[string]any{
					"id":          module.ID,
					"name":        module.Name,
					"type":        module.Type,
					"status":      normalizedStatus(module.Status),
					"approved_at": module.ApprovedAt,
				},
				"note": "La aprobacion del contenido se persiste en approved_at del course-module.",
			}

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(payload)
			}

			rows := [][]string{
				{"Module ID", intToString(module.ID)},
				{"Module Name", module.Name},
				{"Module Type", module.Type},
				{"Module Status", normalizedStatus(module.Status)},
				{"Approved", boolToYesNo(module.ApprovedAt != nil)},
				{"Approved At", stringPtrOrDash(module.ApprovedAt)},
			}
			return output.PrintTable([]string{"Field", "Value"}, rows)
		},
	}

	command.Flags().BoolVar(&approved, "approved", true, "true para aprobar, false para desaprobar")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Muestra la operacion sin enviar peticiones")

	return command
}

func newContentDeleteCommand(opts *rootOptions) *cobra.Command {
	var dryRun bool

	command := &cobra.Command{
		Use:   "delete <content-id>",
		Short: "Elimina un course-content por ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			contentID, err := strconv.Atoi(args[0])
			if err != nil || contentID <= 0 {
				return fmt.Errorf("content-id must be a positive integer")
			}
			contentIDString := intToString(contentID)

			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			if dryRun {
				return output.PrintJSON(map[string]any{
					"action":     "content delete",
					"content_id": contentID,
					"endpoint":   "/course-content/" + contentIDString,
					"method":     "DELETE",
				})
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			if err := rt.Client.DeleteCourseContent(ctx, contentIDString); err != nil {
				return err
			}

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(map[string]any{
					"deleted":    true,
					"content_id": contentID,
				})
			}

			writeLine("Course content %s deleted.", contentIDString)
			return nil
		},
	}

	command.Flags().BoolVar(&dryRun, "dry-run", false, "Muestra la operacion sin enviar peticiones")

	return command
}
