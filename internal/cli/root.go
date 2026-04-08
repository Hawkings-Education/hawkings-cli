package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"hawkings-cli/internal/api"
	"hawkings-cli/internal/config"
	"hawkings-cli/internal/output"

	"github.com/spf13/cobra"
)

type rootOptions struct {
	profile      string
	output       string
	configPath   string
	baseURL      string
	xAPIKey      string
	apiKey       string
	platformUUID string
	timeout      time.Duration
}

type runtime struct {
	Config     config.ResolvedConfig
	ConfigFile config.LoadResult
	Client     *api.Client
	Format     output.Format
}

func NewRootCommand() *cobra.Command {
	opts := &rootOptions{}

	cmd := &cobra.Command{
		Use:   "hawkings",
		Short: "CLI de Hawkings con perfiles, config TOML y salida estable en JSON",
	}

	cmd.PersistentFlags().StringVar(&opts.profile, "profile", "", "Perfil a usar")
	cmd.PersistentFlags().StringVar(&opts.output, "output", string(output.FormatAuto), "Formato de salida: auto, json o table")
	cmd.PersistentFlags().StringVar(&opts.configPath, "config", "", "Ruta alternativa al hawkings.toml local")
	cmd.PersistentFlags().StringVar(&opts.baseURL, "base-url", "", "Base URL de la API")
	cmd.PersistentFlags().StringVar(&opts.xAPIKey, "x-api-key", "", "x-api-key completo")
	cmd.PersistentFlags().StringVar(&opts.apiKey, "api-key", "", "API key sin platform UUID")
	cmd.PersistentFlags().StringVar(&opts.platformUUID, "platform-uuid", "", "UUID de plataforma para combinar con --api-key")
	cmd.PersistentFlags().DurationVar(&opts.timeout, "timeout", 0, "Timeout HTTP, por ejemplo 30s")

	cmd.AddCommand(newConfigCommand(opts))
	cmd.AddCommand(newAuthCommand(opts))
	cmd.AddCommand(newPlatformCommand(opts))
	cmd.AddCommand(newLanguageCommand(opts))
	cmd.AddCommand(newFacultyCommand(opts))
	cmd.AddCommand(newTemplateCommand(opts))
	cmd.AddCommand(newSpaceCommand(opts))
	cmd.AddCommand(newProgramCommand(opts))
	cmd.AddCommand(newCourseCommand(opts))
	cmd.AddCommand(newScormCommand(opts))
	cmd.AddCommand(newSectionCommand(opts))
	cmd.AddCommand(newModuleCommand(opts))
	cmd.AddCommand(newContentCommand(opts))
	cmd.AddCommand(newDescribeCommand(opts))

	return cmd
}

func buildRuntime(opts *rootOptions, requireAuth bool) (*runtime, error) {
	format, err := output.ParseFormat(strings.TrimSpace(opts.output))
	if err != nil {
		return nil, err
	}

	loadResult, err := config.Load(config.LoadOptions{
		LocalConfigPath: opts.configPath,
	})
	if err != nil {
		return nil, err
	}

	resolved, err := config.Resolve(loadResult, config.Overrides{
		Profile:      opts.profile,
		BaseURL:      opts.baseURL,
		XAPIKey:      opts.xAPIKey,
		APIKey:       opts.apiKey,
		PlatformUUID: opts.platformUUID,
		Timeout:      opts.timeout,
	})
	if err != nil {
		if requireAuth {
			return nil, err
		}
		resolved = config.ResolvedConfig{
			Sources: config.Sources{
				Paths: loadResult.Paths,
			},
		}
	}

	rt := &runtime{
		Config:     resolved,
		ConfigFile: loadResult,
		Format:     format,
	}
	if requireAuth {
		rt.Client = api.NewClient(resolved)
	}
	return rt, nil
}

func commandContext(rt *runtime) (context.Context, context.CancelFunc) {
	timeout := rt.Config.Timeout
	if timeout == 0 {
		timeout = config.DefaultTimeout
	}
	return api.WithTimeout(context.Background(), timeout)
}

func commandContextWithMinimum(rt *runtime, explicitOverride time.Duration, minimum time.Duration) (context.Context, context.CancelFunc) {
	timeout := rt.Config.Timeout
	if timeout == 0 {
		timeout = config.DefaultTimeout
	}
	if explicitOverride == 0 && timeout < minimum {
		timeout = minimum
	}
	return api.WithTimeout(context.Background(), timeout)
}

func failJSON(format output.Format, message string) error {
	if output.WantsJSON(format) {
		_ = output.PrintJSON(map[string]string{"error": message})
		return nil
	}
	return errors.New(message)
}

func writeLine(format string, args ...any) {
	fmt.Fprintf(os.Stdout, format+"\n", args...)
}
