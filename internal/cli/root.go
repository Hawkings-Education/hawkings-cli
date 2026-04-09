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
	"golang.org/x/term"
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
		Short: "Hawkings CLI: mutaciones seguras y salida estable para el backend Laravel",
		Long:  rootHelpIntro(),
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

func rootHelpIntro() string {
	bannerLines := []string{
		" _                _    _",
		"| |__   __ ___      _| | _(_)_ __   __ _ ___",
		"| '_ \\ / _' \\ \\ /\\ / / |/ / | '_ \\ / _' / __|",
		"| | | | (_| |\\ V  V /|   <| | | | | (_| \\__ \\",
		"|_| |_|\\__,_| \\_/\\_/ |_|\\_\\_|_| |_|\\__, |___/",
		"                                   |___/",
	}

	intro := strings.Join([]string{
		"CLI de Hawkings para operar contra el backend Laravel con perfiles,",
		"config TOML, dry-run y salida estable en JSON.",
	}, "\n")

	if !supportsANSIColor() {
		return "\n" + strings.Join(bannerLines, "\n") + "\n\n" + intro + "\n"
	}

	return "\n" + gradientASCII(bannerLines) + "\n\n" + ansiRGB(136, 229, 255) + intro + ansiReset() + "\n"
}

func supportsANSIColor() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if strings.EqualFold(os.Getenv("TERM"), "dumb") {
		return false
	}
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func gradientASCII(lines []string) string {
	start := [3]int{76, 201, 240}
	mid := [3]int{67, 97, 238}
	end := [3]int{181, 23, 158}

	var out strings.Builder

	for lineIndex, line := range lines {
		visible := 0
		for _, r := range line {
			if r != ' ' {
				visible++
			}
		}

		if visible == 0 {
			out.WriteString(line)
		} else {
			seen := 0
			for _, r := range line {
				if r == ' ' {
					out.WriteRune(r)
					continue
				}

				ratio := 0.0
				if visible > 1 {
					ratio = float64(seen) / float64(visible-1)
				}

				color := interpolateGradient(start, mid, end, ratio)
				out.WriteString(ansiRGB(color[0], color[1], color[2]))
				out.WriteRune(r)
				seen++
			}
			out.WriteString(ansiReset())
		}

		if lineIndex < len(lines)-1 {
			out.WriteByte('\n')
		}
	}

	return out.String()
}

func interpolateGradient(start, mid, end [3]int, ratio float64) [3]int {
	if ratio <= 0.5 {
		return interpolateColor(start, mid, ratio*2)
	}
	return interpolateColor(mid, end, (ratio-0.5)*2)
}

func interpolateColor(from, to [3]int, ratio float64) [3]int {
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}

	return [3]int{
		from[0] + int(float64(to[0]-from[0])*ratio),
		from[1] + int(float64(to[1]-from[1])*ratio),
		from[2] + int(float64(to[2]-from[2])*ratio),
	}
}

func ansiRGB(r, g, b int) string {
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b)
}

func ansiReset() string {
	return "\x1b[0m"
}
