package output

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"golang.org/x/term"
)

type Format string

const (
	FormatAuto  Format = "auto"
	FormatJSON  Format = "json"
	FormatTable Format = "table"
)

func ParseFormat(value string) (Format, error) {
	switch Format(value) {
	case FormatAuto, FormatJSON, FormatTable:
		return Format(value), nil
	default:
		return "", fmt.Errorf("unsupported output format %q; use auto, json, or table", value)
	}
}

func IsTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func WantsJSON(format Format) bool {
	if format == FormatJSON {
		return true
	}
	if format == FormatAuto && !IsTTY() {
		return true
	}
	return false
}

func PrintJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func PrintTable(headers []string, rows [][]string) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	if len(headers) > 0 {
		for i, header := range headers {
			if i > 0 {
				fmt.Fprint(w, "\t")
			}
			fmt.Fprint(w, header)
		}
		fmt.Fprintln(w)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i > 0 {
				fmt.Fprint(w, "\t")
			}
			fmt.Fprint(w, cell)
		}
		fmt.Fprintln(w)
	}
	return w.Flush()
}
