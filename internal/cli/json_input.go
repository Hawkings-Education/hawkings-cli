package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

type jsonInputOptions struct {
	JSON     string
	JSONFile string
}

func addJSONInputFlags(command *cobra.Command, opts *jsonInputOptions) {
	flags := command.Flags()
	flags.StringVar(&opts.JSON, "json", "", "Payload JSON inline")
	flags.StringVar(&opts.JSONFile, "json-file", "", "Ruta a un fichero JSON")
}

func readJSONObject(opts jsonInputOptions) (map[string]any, error) {
	if opts.JSON != "" && opts.JSONFile != "" {
		return nil, fmt.Errorf("use either --json or --json-file, not both")
	}
	if opts.JSON == "" && opts.JSONFile == "" {
		return nil, fmt.Errorf("missing JSON input; use --json or --json-file")
	}

	var data []byte
	if opts.JSONFile != "" {
		fileData, err := os.ReadFile(opts.JSONFile)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", opts.JSONFile, err)
		}
		data = fileData
	} else {
		data = []byte(opts.JSON)
	}

	var payload map[string]any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode JSON object: %w", err)
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return payload, nil
}

func readOptionalJSONObject(opts jsonInputOptions) (map[string]any, error) {
	if opts.JSON == "" && opts.JSONFile == "" {
		return nil, nil
	}
	return readJSONObject(opts)
}

func readJSONFile(path string) (any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var value any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("decode JSON value from %s: %w", path, err)
	}

	return value, nil
}

func cloneMap(source map[string]any) map[string]any {
	if source == nil {
		return nil
	}
	out := make(map[string]any, len(source))
	for key, value := range source {
		if nested, ok := value.(map[string]any); ok {
			out[key] = cloneMap(nested)
			continue
		}
		out[key] = value
	}
	return out
}

func cloneMaps(source []map[string]any) []map[string]any {
	if source == nil {
		return nil
	}
	out := make([]map[string]any, 0, len(source))
	for _, item := range source {
		out = append(out, cloneMap(item))
	}
	return out
}

func deepMergeMap(base, overlay map[string]any) map[string]any {
	if base == nil {
		return cloneMap(overlay)
	}

	out := cloneMap(base)
	for key, value := range overlay {
		baseMap, baseIsMap := out[key].(map[string]any)
		overlayMap, overlayIsMap := value.(map[string]any)

		if baseIsMap && overlayIsMap {
			out[key] = deepMergeMap(baseMap, overlayMap)
			continue
		}

		out[key] = value
	}

	return out
}
