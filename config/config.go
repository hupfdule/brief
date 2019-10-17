package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

var (
	defaultEditors         = []string{"nvim", "vim", "$VISUAL", "sensible-editor", "$EDITOR"}
	defaultPdfCommands     = []string{"latexrun", "latexmk", "lualatex", "xelatex", "pdflatex"}
	defaultPreviewCommands = []string{"mupdf", "zathura", "katarakt", "evince", "okular", "qpdfview", "skim", "SumatraPDF", "xpdf"}
)

// Config contains the configuration for the brief application.
// It can (and should) be prefilled via call to NewConfig() and can
// (and should) be overriden via config file or command line flags.
type Config struct {
	Editor           string
	TexTemplateDir   string
	DocumentRoots    []string
	AddressBook      string
	SenderList       string
	PdfCommand       string
	PreviewCommand   string
	FindCommand      string
	ListerCommand    string
	MarkupConverters map[string]string
}

// NewConfig creates a new Config with the default configuration.
// The default configuration will check for the existence of some default
// applications. Therefore, depending on the available applications on the
// current system, this created config may be different than the same call
// on other systems. This is done to ensure default values that actually
// make sense.
//
// If some fields cannot be initialzied, because none of the default
// applications is availble the field will be left uninitialized.
//
// This function returns a pointer to the generated default config
func NewConfig() *Config {
	c := new(Config)

	editor, err := findExecutable(defaultEditors)
	if err != nil {
		log.Println(fmt.Errorf("No default editor found: %w", err))
	}
	c.Editor = editor

	texTemplateDir, err := expandFileName("~/.config/brief/tex-templates")
	if err != nil {
		log.Println(fmt.Errorf("Error expanding text template dir %s: %w", "~/.config/brief/tex-templates", err))
	}
	c.TexTemplateDir = texTemplateDir

	c.DocumentRoots = []string{"."}

	addresBook, err := expandFileName("~/.config/brief/addressbook")
	if err != nil {
		log.Println(fmt.Errorf("Error expanding address book file %s: %w", "~/.config/brief/addressbook", err))
	}
	c.AddressBook = addresBook

	senderList, err := expandFileName("~/.config/brief/sender-list")
	if err != nil {
		log.Println(fmt.Errorf("Error expanding sender list file %s: %w", "~/.config/brief/sender-list", err))
	}
	c.SenderList = senderList

	pdfCommand, err := findExecutable(defaultPdfCommands)
	if err != nil {
		log.Println(fmt.Errorf("No default pdf-command found: %w", err))
	}
	c.PdfCommand = pdfCommand

	previewCommand, err := findExecutable(defaultPreviewCommands)
	if err != nil {
		log.Println(fmt.Errorf("No default preview-command found: %w", err))
	}
	c.PreviewCommand = previewCommand

	findCommand, err := findDefaultFindCommand()
	if err != nil {
		log.Println(fmt.Errorf("No default find-command found: %w", err))
	}
	c.FindCommand = findCommand

	listerCommand, err := findDefaultListerCommand()
	if err != nil {
		log.Println(fmt.Errorf("No default lister-command found: %w", err))
	}
	c.ListerCommand = listerCommand

	markupConverters := findDefaultMarkupConverters()
	c.MarkupConverters = markupConverters

	return c
}

// findExecutable iterates over a list of excutable names and tries whether
// those can be found in the $PATH. The first executable that is found is
// returned.
//
// If a given executable name starts with a dollar sign ($) it is
// interpreted as an environment variable and therefore those environment
// variable will be expanded before checking for the existance of the
// executable.
func findExecutable(executables []string) (string, error) {
	for _, executable := range executables {
		// resolve content of environment variables
		var executableName string
		if executable[0] == '$' {
			executableName = os.Getenv(executable[1:])
		} else {
			executableName = executable
		}

		path, _ := exec.LookPath(executableName)
		if path != "" {
			return executableName, nil
		}
	}

	return "", errors.New("No usable executable found")
}

func findDefaultFindCommand() (string, error) {
	//TODO: Real decision logic
	return "rg", nil
}

func findDefaultListerCommand() (string, error) {
	//TODO: Real decision logic
	return "fzf", nil
}

// findDefaultMarkupConverters tries to find executables to convert content
// in markup sections to LaTeX code.
//
// By default this registers either 'asciidoctor' or 'asciidoc' to convert
// asciidoc content and 'pandoc' for all other markup.
func findDefaultMarkupConverters() map[string]string {
	m := make(map[string]string)

	// asciidoctor or asciidoc + docbook + pandoc for asciidoc markup
	x := lookPaths("asciidoctor", "asciidoc", "pandoc")
	if x["pandoc"] != "" {
		if x["asciidoctor"] != "" {
			m["asciidoc"] = "asciidoctor -b docbook5 - | pandoc -f docbook -t latex"
		} else if x["asciidoc"] != "" {
			m["asciidoc"] = "asciidoc -b docbook5 - | pandoc -f docbook -t latex"
		}
	}
	m["adoc"] = m["asciidoc"]

	// pandoc as default for all other markups
	if x["pandoc"] != "" {
		m["*"] = "pandoc -f %m -t latex"
	}

	return m
}

// lookPaths tries to find the given executables in the current $PATH.
//
// The return value is a map with the given executable names as key.
// If the executable was found in $PATH the absolute path to the executable
// is set as the value in the map. Otherwise the value will be an empty
// string.
func lookPaths(executables ...string) map[string]string {
	m := make(map[string]string)

	for _, e := range executables {
		path, err := exec.LookPath(e)
		if err != nil {
			log.Println(fmt.Errorf("Error checking existance of executable %s: %w", e, err))
		}
		m[e] = path
	}

	return m
}

// expandFileName returns the fill path of file by replacing a leading '~/'
// with the actual path of the users home directory.
//
// An error is returned if the current user cannot be determined.
func expandFileName(fileName string) (string, error) {
	if fileName == "~" {
		return homeDir()
	} else if strings.HasPrefix(fileName, "~/") {
		homeDir, err := homeDir()
		if err != nil {
			return "", fmt.Errorf("Error detecting home directory: %w", err)
		}
		return filepath.Join(homeDir, fileName[2:]), nil
	} else {
		return fileName, nil
	}
}

// homeDir returns the path of the home directory of the current user.
//
// An error is returned if the current user cannot be determined.
func homeDir() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("Cannot determine current user: %w", err)
	}
	return usr.HomeDir, nil
}
