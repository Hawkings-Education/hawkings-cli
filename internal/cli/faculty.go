package cli

import (
	"context"
	"net/url"

	"hawkings-cli/internal/api"
	"hawkings-cli/internal/output"

	"github.com/spf13/cobra"
)

var defaultFacultyWith = []string{"courseArea"}

func newFacultyCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "faculty",
		Short: "Comandos sobre facultades de referencia",
	}
	cmd.AddCommand(newFacultyListCommand(opts))
	return cmd
}

func newFacultyListCommand(opts *rootOptions) *cobra.Command {
	var limit int
	var page int
	var search string
	var all bool
	var with []string

	command := &cobra.Command{
		Use:   "list",
		Short: "Lista las facultades accesibles para la platform activa",
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

			list, err := listAllCourseFaculties(ctx, rt.Client, params, uniqueStrings(append(defaultFacultyWith, with...)), targetPage, targetLimit)
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
					valueOrDash(courseAreaLabel(item.CourseArea)),
					boolToYesNo(item.Enabled),
				})
			}
			if err := output.PrintTable([]string{"ID", "Code", "Name", "Area", "Enabled"}, rows); err != nil {
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

func listAllCourseFaculties(ctx context.Context, client *api.Client, params url.Values, with []string, page, limit int) (api.CourseFacultyList, error) {
	firstPage, err := client.ListCourseFaculties(ctx, params, with)
	if err != nil {
		return api.CourseFacultyList{}, err
	}

	items := append([]api.CourseFaculty{}, firstPage.Data...)
	for nextPage := 2; nextPage <= firstPage.Pages; nextPage++ {
		pageParams := cloneURLValues(params)
		pageParams.Set("page", intToString(nextPage))

		next, err := client.ListCourseFaculties(ctx, pageParams, with)
		if err != nil {
			return api.CourseFacultyList{}, err
		}
		items = append(items, next.Data...)
	}

	return paginateCourseFaculties(items, page, limit), nil
}

func paginateCourseFaculties(items []api.CourseFaculty, page, limit int) api.CourseFacultyList {
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

	return api.CourseFacultyList{
		Data:   items[start:end],
		Pages:  pages,
		Page:   page,
		Offset: start,
		Total:  total,
	}
}

func cloneURLValues(values url.Values) url.Values {
	if values == nil {
		return url.Values{}
	}
	out := url.Values{}
	for key, value := range values {
		out[key] = append([]string{}, value...)
	}
	return out
}
