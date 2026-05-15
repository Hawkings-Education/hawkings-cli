package cli

import (
	"context"
	"net/url"

	"hawkings-cli/internal/api"
	"hawkings-cli/internal/output"

	"github.com/spf13/cobra"
)

func newAreaCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "area",
		Short: "Comandos sobre areas de referencia",
	}
	cmd.AddCommand(newAreaListCommand(opts))
	cmd.AddCommand(newAreaGetCommand(opts))
	return cmd
}

func newAreaListCommand(opts *rootOptions) *cobra.Command {
	var limit int
	var page int
	var search string
	var all bool
	var with []string

	command := &cobra.Command{
		Use:   "list",
		Short: "Lista las areas accesibles para la platform activa",
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			params := url.Values{}
			if search != "" {
				params.Set("search", search)
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			targetPage := page
			targetLimit := limit
			if all {
				targetPage = 1
				targetLimit = 0
			}

			list, err := listAllCourseAreas(ctx, rt.Client, params, with, targetPage, targetLimit)
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
					valueOrDash(item.Code),
					item.Name,
					boolToYesNo(item.Enabled),
				})
			}
			if err := output.PrintTable([]string{"ID", "Code", "Name", "Enabled"}, rows); err != nil {
				return err
			}

			writeLine("")
			writeLine("Page %d/%d  Total %d", list.Page, list.Pages, list.Total)
			if !all && list.Pages > 1 {
				writeLine("Hint: usa --all para recuperar todos los resultados en una sola salida.")
			}
			return nil
		},
	}

	command.Flags().IntVar(&limit, "limit", 20, "Limite por pagina")
	command.Flags().IntVar(&page, "page", 1, "Pagina")
	command.Flags().StringVar(&search, "search", "", "Texto de busqueda")
	command.Flags().BoolVar(&all, "all", false, "Recorre todas las paginas y devuelve todos los resultados")
	command.Flags().StringArrayVar(&with, "with", nil, "Relaciones extra via with[]")

	return command
}

func newAreaGetCommand(opts *rootOptions) *cobra.Command {
	var with []string

	command := &cobra.Command{
		Use:   "get <area-id>",
		Short: "Muestra el detalle de un area",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := buildRuntime(opts, true)
			if err != nil {
				return err
			}

			ctx, cancel := commandContext(rt)
			defer cancel()

			area, err := rt.Client.GetCourseArea(ctx, args[0], with)
			if err != nil {
				return err
			}

			if output.WantsJSON(rt.Format) {
				return output.PrintRawJSON(rt.Client.LastRawBody())
			}

			rows := [][]string{
				{"ID", intToString(area.ID)},
				{"Code", valueOrDash(area.Code)},
				{"Name", area.Name},
				{"Description", stringPtrOrDash(area.Description)},
				{"Enabled", boolToYesNo(area.Enabled)},
			}
			return output.PrintTable([]string{"Field", "Value"}, rows)
		},
	}

	command.Flags().StringArrayVar(&with, "with", nil, "Relaciones extra via with[]")

	return command
}

func listAllCourseAreas(ctx context.Context, client *api.Client, params url.Values, with []string, page, limit int) (api.CourseAreaList, error) {
	firstPage, err := client.ListCourseAreas(ctx, params, with)
	if err != nil {
		return api.CourseAreaList{}, err
	}

	items := append([]api.CourseArea{}, firstPage.Data...)
	for nextPage := 2; nextPage <= firstPage.Pages; nextPage++ {
		pageParams := cloneURLValues(params)
		pageParams.Set("page", intToString(nextPage))

		next, err := client.ListCourseAreas(ctx, pageParams, with)
		if err != nil {
			return api.CourseAreaList{}, err
		}
		items = append(items, next.Data...)
	}

	return paginateCourseAreas(items, page, limit), nil
}

func paginateCourseAreas(items []api.CourseArea, page, limit int) api.CourseAreaList {
	if limit <= 0 {
		limit = len(items)
		if limit == 0 {
			limit = 1
		}
	}
	if page <= 0 {
		page = 1
	}

	total := len(items)
	pages := (total + limit - 1) / limit
	if pages == 0 {
		pages = 1
	}
	if page > pages {
		page = pages
	}

	start := (page - 1) * limit
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}

	return api.CourseAreaList{
		Data:   items[start:end],
		Pages:  pages,
		Page:   page,
		Offset: start,
		Total:  total,
	}
}
