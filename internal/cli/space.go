package cli

import (
	"fmt"
	"net/url"
	"strings"

	"hawkings-cli/internal/api"
	"hawkings-cli/internal/output"

	"github.com/spf13/cobra"
)

var defaultSpaceWith = []string{"user", "courseProgramsCount", "usersCount"}

func newSpaceCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "space",
		Short: "Comandos sobre spaces",
	}
	cmd.AddCommand(newSpaceListCommand(opts))
	cmd.AddCommand(newSpaceGetCommand(opts))
	cmd.AddCommand(newSpaceCreateCommand(opts))
	cmd.AddCommand(newSpaceUpdateCommand(opts))
	cmd.AddCommand(newSpaceProgramsCommand(opts))
	cmd.AddCommand(newSpaceAssignProgramsCommand(opts))
	cmd.AddCommand(newSpaceUsersCommand(opts))
	cmd.AddCommand(newSpaceAssignUsersCommand(opts))
	cmd.AddCommand(newSpaceDeleteCommand(opts))
	return cmd
}

func newSpaceListCommand(opts *rootOptions) *cobra.Command {
	var limit int
	var page int
	var search string
	var enabled string
	var personal string
	var with []string

	command := &cobra.Command{
		Use:   "list",
		Short: "Lista spaces accesibles para el usuario activo",
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			params := url.Values{}
			if limit > 0 {
				params.Set("limit", intToString(limit))
			}
			if page > 0 {
				params.Set("page", intToString(page))
			}
			if search != "" {
				params.Set("search", search)
			}
			if enabled != "" {
				params.Set("enabled", enabled)
			}
			if personal != "" {
				params.Set("personal", personal)
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			list, err := rt.Client.ListSpaces(ctx, params, uniqueStrings(append(defaultSpaceWith, with...)))
			if err != nil {
				return err
			}

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(list)
			}

			rows := make([][]string, 0, len(list.Data))
			for _, item := range list.Data {
				rows = append(rows, []string{
					intToString(item.ID),
					item.Name,
					item.Color,
					boolToYesNo(item.Personal),
					boolToYesNo(item.Enabled),
					intToString(anyInt(item.CourseProgramsCount)),
					intToString(anyInt(item.UsersCount)),
				})
			}
			if err := output.PrintTable([]string{"ID", "Name", "Color", "Personal", "Enabled", "Programs", "Users"}, rows); err != nil {
				return err
			}
			writeLine("")
			writeLine("Page %d/%d  Total %d", list.Page, list.Pages, list.Total)
			return nil
		},
	}

	command.Flags().IntVar(&limit, "limit", 20, "Limite por pagina")
	command.Flags().IntVar(&page, "page", 1, "Pagina")
	command.Flags().StringVar(&search, "search", "", "Texto de busqueda")
	command.Flags().StringVar(&enabled, "enabled", "", "Filtra por enabled=true|false")
	command.Flags().StringVar(&personal, "personal", "", "Filtra por personal=true|false")
	command.Flags().StringArrayVar(&with, "with", nil, "Relaciones extra via with[]")

	return command
}

func newSpaceGetCommand(opts *rootOptions) *cobra.Command {
	var with []string

	command := &cobra.Command{
		Use:   "get <space-id>",
		Short: "Muestra el detalle de un space",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			space, err := rt.Client.GetSpace(ctx, args[0], uniqueStrings(append(defaultSpaceWith, with...)))
			if err != nil {
				return err
			}

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(space)
			}

			rows := [][]string{
				{"ID", intToString(space.ID)},
				{"Name", space.Name},
				{"Remote ID", stringPtrOrDash(space.RemoteID)},
				{"Description", stringPtrOrDash(space.Description)},
				{"Color", space.Color},
				{"Personal", boolToYesNo(space.Personal)},
				{"Enabled", boolToYesNo(space.Enabled)},
				{"Programs", intToString(len(space.CoursePrograms))},
				{"Programs count", intToString(anyInt(space.CourseProgramsCount))},
				{"Users", intToString(len(space.Users))},
				{"Users count", intToString(anyInt(space.UsersCount))},
				{"Owner", valueOrDash(spaceOwnerLabel(space.User))},
			}
			if err := output.PrintTable([]string{"Field", "Value"}, rows); err != nil {
				return err
			}

			if len(space.CoursePrograms) > 0 {
				writeLine("")
				writeLine("Programs:")
				programRows := make([][]string, 0, len(space.CoursePrograms))
				for _, item := range space.CoursePrograms {
					programRows = append(programRows, []string{
						intToString(item.ID),
						item.Name,
						normalizedStatus(item.Status),
					})
				}
				return output.PrintTable([]string{"ID", "Name", "Status"}, programRows)
			}

			return nil
		},
	}

	command.Flags().StringArrayVar(&with, "with", nil, "Relaciones extra via with[]")

	return command
}

func newSpaceCreateCommand(opts *rootOptions) *cobra.Command {
	var input jsonInputOptions
	var dryRun bool

	command := &cobra.Command{
		Use:   "create",
		Short: "Crea un space a partir de un payload JSON",
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
					"action":  "space create",
					"payload": payload,
				})
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			space, err := rt.Client.CreateSpace(ctx, payload)
			if err != nil {
				return err
			}

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(space)
			}

			rows := [][]string{
				{"ID", intToString(space.ID)},
				{"Name", space.Name},
				{"Remote ID", stringPtrOrDash(space.RemoteID)},
				{"Color", space.Color},
				{"Personal", boolToYesNo(space.Personal)},
				{"Enabled", boolToYesNo(space.Enabled)},
			}
			return output.PrintTable([]string{"Field", "Value"}, rows)
		},
	}

	addJSONInputFlags(command, &input)
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Muestra el payload sin enviar peticiones")

	return command
}

func newSpaceUpdateCommand(opts *rootOptions) *cobra.Command {
	var input jsonInputOptions
	var dryRun bool

	command := &cobra.Command{
		Use:   "update <space-id>",
		Short: "Hace GET del space, merge del patch y PATCH del recurso completo",
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

			ctx, cancel := commandContext(rt)
			defer cancel()

			current, err := rt.Client.GetSpace(ctx, args[0], nil)
			if err != nil {
				return err
			}

			merged := mergeSpaceUpdatePatch(current, patch)

			if dryRun {
				return output.PrintJSON(map[string]any{
					"action":       "space update",
					"space_id":     args[0],
					"input_patch":  patch,
					"merged_patch": merged,
				})
			}

			updated, err := rt.Client.UpdateSpace(ctx, args[0], merged)
			if err != nil {
				return err
			}

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(updated)
			}

			rows := [][]string{
				{"ID", intToString(updated.ID)},
				{"Name", updated.Name},
				{"Remote ID", stringPtrOrDash(updated.RemoteID)},
				{"Description", stringPtrOrDash(updated.Description)},
				{"Color", updated.Color},
				{"Personal", boolToYesNo(updated.Personal)},
				{"Enabled", boolToYesNo(updated.Enabled)},
			}
			return output.PrintTable([]string{"Field", "Value"}, rows)
		},
	}

	addJSONInputFlags(command, &input)
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Muestra el patch final sin enviar peticiones")

	return command
}

func newSpaceProgramsCommand(opts *rootOptions) *cobra.Command {
	var with []string

	command := &cobra.Command{
		Use:   "programs <space-id>",
		Short: "Lista los programas asociados a un space",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			programs, err := rt.Client.GetSpacePrograms(ctx, args[0], with)
			if err != nil {
				return err
			}

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(programs)
			}

			rows := make([][]string, 0, len(programs))
			for _, item := range programs {
				rows = append(rows, []string{
					intToString(item.ID),
					valueOrDash(metadataString(item.Metadata, "code")),
					item.Name,
					normalizedStatus(item.Status),
					valueOrDash(metadataString(item.Metadata, "hours")),
				})
			}
			return output.PrintTable([]string{"ID", "Code", "Name", "Status", "Hours"}, rows)
		},
	}

	command.Flags().StringArrayVar(&with, "with", nil, "Relaciones extra via with[]")

	return command
}

func newSpaceAssignProgramsCommand(opts *rootOptions) *cobra.Command {
	var selected []int
	var add []int
	var remove []int
	var all []int
	var dryRun bool

	command := &cobra.Command{
		Use:   "assign-programs <space-id>",
		Short: "Asigna programas a un space mediante selected, add, remove o all",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			payload := map[string]any{}
			if len(selected) > 0 {
				payload["selected"] = selected
			}
			if len(add) > 0 {
				payload["add"] = add
			}
			if len(remove) > 0 {
				payload["remove"] = remove
			}
			if len(all) > 0 {
				payload["all"] = all
			}
			if len(payload) == 0 {
				return fmt.Errorf("use at least one of --selected, --add, --remove or --all")
			}

			if dryRun {
				return output.PrintJSON(map[string]any{
					"action":   "space assign-programs",
					"space_id": args[0],
					"payload":  payload,
				})
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			programs, err := rt.Client.UpdateSpacePrograms(ctx, args[0], payload)
			if err != nil {
				return err
			}

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(programs)
			}

			rows := make([][]string, 0, len(programs))
			for _, item := range programs {
				rows = append(rows, []string{
					intToString(item.ID),
					valueOrDash(metadataString(item.Metadata, "code")),
					item.Name,
					normalizedStatus(item.Status),
				})
			}
			return output.PrintTable([]string{"ID", "Code", "Name", "Status"}, rows)
		},
	}

	command.Flags().IntSliceVar(&selected, "selected", nil, "Lista exacta de program IDs que debe quedar asignada")
	command.Flags().IntSliceVar(&add, "add", nil, "Program IDs a anadir")
	command.Flags().IntSliceVar(&remove, "remove", nil, "Program IDs a quitar")
	command.Flags().IntSliceVar(&all, "all", nil, "Program IDs sobre los que operar internamente como conjunto base")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Muestra el payload sin enviar peticiones")

	return command
}

func newSpaceUsersCommand(opts *rootOptions) *cobra.Command {
	command := &cobra.Command{
		Use:   "users <space-id>",
		Short: "Lista los usuarios asociados a un space",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			users, err := rt.Client.GetSpaceUsers(ctx, args[0])
			if err != nil {
				return err
			}

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(users)
			}

			rows := make([][]string, 0, len(users))
			for _, user := range users {
				rows = append(rows, []string{
					fmt.Sprintf("%v", user.ID),
					user.Name,
					user.Surname,
					valueOrDash(user.Email),
				})
			}
			return output.PrintTable([]string{"ID", "Name", "Surname", "Email"}, rows)
		},
	}

	return command
}

func newSpaceAssignUsersCommand(opts *rootOptions) *cobra.Command {
	var selected []int
	var add []int
	var remove []int
	var all []int
	var dryRun bool

	command := &cobra.Command{
		Use:   "assign-users <space-id>",
		Short: "Asigna usuarios a un space mediante selected, add, remove o all",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			payload := map[string]any{}
			if len(selected) > 0 {
				payload["selected"] = selected
			}
			if len(add) > 0 {
				payload["add"] = add
			}
			if len(remove) > 0 {
				payload["remove"] = remove
			}
			if len(all) > 0 {
				payload["all"] = all
			}
			if len(payload) == 0 {
				return fmt.Errorf("use at least one of --selected, --add, --remove or --all")
			}

			if dryRun {
				return output.PrintJSON(map[string]any{
					"action":   "space assign-users",
					"space_id": args[0],
					"payload":  payload,
				})
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			users, err := rt.Client.UpdateSpaceUsers(ctx, args[0], payload)
			if err != nil {
				return err
			}

			if output.WantsJSON(rt.Format) {
				return output.PrintJSON(users)
			}

			rows := make([][]string, 0, len(users))
			for _, user := range users {
				rows = append(rows, []string{
					fmt.Sprintf("%v", user.ID),
					user.Name,
					user.Surname,
					valueOrDash(user.Email),
				})
			}
			return output.PrintTable([]string{"ID", "Name", "Surname", "Email"}, rows)
		},
	}

	command.Flags().IntSliceVar(&selected, "selected", nil, "Lista exacta de user IDs asignados")
	command.Flags().IntSliceVar(&add, "add", nil, "User IDs a anadir")
	command.Flags().IntSliceVar(&remove, "remove", nil, "User IDs a quitar")
	command.Flags().IntSliceVar(&all, "all", nil, "User IDs usados como conjunto base de operacion")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Muestra el payload sin enviar peticiones")

	return command
}

func newSpaceDeleteCommand(opts *rootOptions) *cobra.Command {
	var dryRun bool

	command := &cobra.Command{
		Use:   "delete <space-id>",
		Short: "Elimina un space si no tiene programas asociados",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			if dryRun {
				return output.PrintJSON(map[string]any{
					"action":   "space delete",
					"space_id": args[0],
				})
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			if err := rt.Client.DeleteSpace(ctx, args[0]); err != nil {
				return err
			}

			return output.PrintJSON(map[string]any{
				"deleted":  true,
				"space_id": args[0],
			})
		},
	}

	command.Flags().BoolVar(&dryRun, "dry-run", false, "Muestra la operacion sin enviar peticiones")

	return command
}

func mergeSpaceUpdatePatch(current api.SpaceDetail, patch map[string]any) map[string]any {
	out := map[string]any{
		"remote_id":   stringOrNil(current.RemoteID),
		"name":        current.Name,
		"description": stringOrNil(current.Description),
		"color":       current.Color,
		"personal":    current.Personal,
		"enabled":     current.Enabled,
	}

	for key, value := range patch {
		out[key] = value
	}

	return out
}

func spaceOwnerLabel(user *api.UserSummary) string {
	if user == nil {
		return ""
	}
	parts := []string{strings.TrimSpace(user.Name), strings.TrimSpace(user.Surname)}
	label := strings.Join(parts, " ")
	label = strings.TrimSpace(label)
	if user.Email != "" {
		if label == "" {
			return user.Email
		}
		return label + " <" + user.Email + ">"
	}
	return label
}
