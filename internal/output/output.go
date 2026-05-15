package output

import (
	"bytes"
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

// PrintRawJSON reemite el body JSON original de la API, reindentado para
// legibilidad. Si raw no es JSON valido cae al body bruto. Devuelve el
// objeto tal cual llega de la API, sin filtrar por las structs del CLI.
func PrintRawJSON(raw []byte) error {
	if len(raw) == 0 {
		_, err := os.Stdout.Write([]byte("null\n"))
		return err
	}
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, raw, "", "  "); err != nil {
		if _, werr := os.Stdout.Write(raw); werr != nil {
			return werr
		}
		if len(raw) == 0 || raw[len(raw)-1] != '\n' {
			_, _ = os.Stdout.Write([]byte("\n"))
		}
		return nil
	}
	if _, err := os.Stdout.Write(pretty.Bytes()); err != nil {
		return err
	}
	_, err := os.Stdout.Write([]byte("\n"))
	return err
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
