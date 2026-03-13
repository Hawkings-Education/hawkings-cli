package cli

import (
	"hawkings-cli/internal/output"

	"github.com/spf13/cobra"
)

func newSectionCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "section",
		Short: "Comandos sobre course sections",
	}
	cmd.AddCommand(newSectionGenerateContentCommand(opts))
	cmd.AddCommand(newSectionGenerateActivitiesCommand(opts))
	return cmd
}

func newSectionGenerateContentCommand(opts *rootOptions) *cobra.Command {
	var researchEnabled bool
	var researchProvider string
	var researchQuality string
	var researchInstructions string
	var researchIDs []int
	var promptCustom string
	var dryRun bool

	command := &cobra.Command{
		Use:   "generate-content <section-id>",
		Short: "Lanza la generacion asincrona de contenido para todos los modules de una section",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			payload := sectionGenerationPayload(researchEnabled, researchProvider, researchQuality, researchInstructions, researchIDs, promptCustom)

			if dryRun {
				return output.PrintJSON(map[string]any{
					"action":     "section generate-content",
					"section_id": args[0],
					"payload":    payload,
				})
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			result, err := rt.Client.GenerateCourseSectionContent(ctx, args[0], payload)
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
	command.Flags().StringVar(&promptCustom, "prompt-custom", "", "Instrucciones de redaccion para todos los modulos de la section")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Muestra el payload sin enviar peticiones")

	return command
}

func newSectionGenerateActivitiesCommand(opts *rootOptions) *cobra.Command {
	var dryRun bool

	command := &cobra.Command{
		Use:   "generate-activities <section-id>",
		Short: "Lanza la generacion asincrona de actividades para una section completa",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			payload := map[string]any{"async": true}
			if dryRun {
				return output.PrintJSON(map[string]any{
					"action":     "section generate-activities",
					"section_id": args[0],
					"payload":    payload,
				})
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			result, err := rt.Client.GenerateCourseSectionActivities(ctx, args[0], payload)
			if err != nil {
				return err
			}

			return output.PrintJSON(result)
		},
	}

	command.Flags().BoolVar(&dryRun, "dry-run", false, "Muestra el payload sin enviar peticiones")

	return command
}
