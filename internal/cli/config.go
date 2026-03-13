package cli

import (
	"hawkings-cli/internal/output"

	"github.com/spf13/cobra"
)

func newConfigCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Inspecciona la configuración resuelta",
	}
	cmd.AddCommand(newConfigShowCommand(opts))
	return cmd
}

func newConfigShowCommand(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Muestra la configuración local/home y el perfil resuelto",
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, false)
			if err != nil {
				return err
			}

			if rt.Config.ProfileName == "" {
				view := map[string]any{
					"sources": rt.Config.Sources,
					"error":   "no active profile configured",
				}
				if output.WantsJSON(rt.Format) {
					return output.PrintJSON(view)
				}
				writeLine("No hay perfil activo configurado.")
				writeLine("Local:  %s", rt.ConfigFile.Paths.Local)
				writeLine("Global: %s", rt.ConfigFile.Paths.Global)
				return nil
			}

			view := rt.Config.RedactedView(rt.ConfigFile)
			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(view)
			}

			writeLine("Perfil activo: %s", view.Profile)
			writeLine("Entorno:       %s", valueOrDash(view.Environment))
			writeLine("Base URL:      %s", view.BaseURL)
			writeLine("x-api-key:     %s", view.XAPIKey)
			writeLine("Platform UUID: %s", valueOrDash(view.PlatformUUID))
			writeLine("Config local:  %s", view.Sources.Paths.Local)
			writeLine("Config home:   %s", view.Sources.Paths.Global)
			return nil
		},
	}
}
