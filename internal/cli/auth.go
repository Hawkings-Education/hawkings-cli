package cli

import (
	"hawkings-cli/internal/output"

	"github.com/spf13/cobra"
)

func newAuthCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Comandos de autenticación",
	}
	cmd.AddCommand(newAuthWhoAmICommand(opts))
	return cmd
}

func newAuthWhoAmICommand(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Muestra el usuario autenticado",
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			profile, err := rt.Client.GetProfile(ctx)
			if err != nil {
				return err
			}

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(profile)
			}

			rows := [][]string{
				{"ID", intToString(profile.ID)},
				{"Email", profile.Email},
				{"Nombre", stringsTrimSpace(profile.Name + " " + profile.Surname)},
				{"Admin", boolToYesNo(profile.Admin)},
				{"Manager", boolToYesNo(profile.Manager)},
				{"Teacher", boolToYesNo(profile.Teacher)},
				{"Student", boolToYesNo(profile.Student)},
			}
			if profile.Language != nil {
				rows = append(rows, []string{"Idioma", profile.Language.Code})
			}
			if profile.LearningPlatform != nil {
				rows = append(rows, []string{"Platform", profile.LearningPlatform.Name})
				rows = append(rows, []string{"Platform UUID", profile.LearningPlatform.UUID})
			}
			return output.PrintTable([]string{"Campo", "Valor"}, rows)
		},
	}
}
