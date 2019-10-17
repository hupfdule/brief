package markup

import (
	"fmt"
	"strings"

	"poiu.de/brief/cmdline"
	"poiu.de/brief/config"
)

// Converter is converts content written in a specific markup language into
// LaTeX code.
type Converter struct {
	// The markup type that this Converter handles.
	markupType string
	// The command to use to convert markup text to LaTeX.
	converterCmd string
}

type converter interface {
	// Convert converts the given lines of markup text into a string of latex code.
	//
	// If conversion fails for some reason, an empty string and an error is
	// returned.
	Convert(lines []string) (string, error)
}

// NewConverter returns a new Converter for the given markup type.
// If no converter can be found for the given markup type, an error will be
// returned.
func NewConverter(markupType string, cfg *config.Config) (*Converter, error) {
	if cfg.MarkupConverters[markupType] != "" {
		return &Converter{markupType: markupType, converterCmd: cfg.MarkupConverters[markupType]}, nil
	} else if cfg.MarkupConverters["*"] != "" {
		return &Converter{markupType: markupType, converterCmd: strings.ReplaceAll(cfg.MarkupConverters["*"], "%m", markupType)}, nil
	} else {
		return nil, fmt.Errorf("No converter configured for markupType %s. Consider installing pandoc.", markupType)
	}
	if markupType == "markdown" {
		return &Converter{markupType: markupType}, nil
	} else {
		return nil, fmt.Errorf("No converter configured for markupType %s", markupType)
	}
}

// Convert converts the given lines of markup text into a string of latex code.
//
// If conversion fails for some reason, an empty string and an error is
// returned.
func (c *Converter) Convert(lines []string) (string, error) {
	if len(strings.TrimSpace(c.converterCmd)) == 0 {
		return "", fmt.Errorf("No converterCmd defined for markup %s", c.markupType)
	}

	cmdLine := c.converterCmd
	stdin := strings.NewReader(strings.Join(lines, "\n"))
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	err := cmdline.Execute(cmdLine, "", stdin, stdout, stderr)
	if err != nil {
		return "", fmt.Errorf("Error converting %s markup via %s: %w", c.markupType, c.converterCmd, err)
	}

	return stdout.String(), nil
}
