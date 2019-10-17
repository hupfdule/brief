package cmd

import (
	"fmt"
	"strings"

	"gopkg.in/alecthomas/kingpin.v2"
	"poiu.de/brief/cmdline"
	"poiu.de/brief/config"
	"poiu.de/brief/utils"
)

// PreviewCommand calls an external application to display the PDF file
// generated from a .brf file.
//
// If necessary it creates the corresponding PDF file by calling the
// PdfCommand (which itself may call the TexCommand if necessary).
type PreviewCommand struct {
	// The .brf file to preview.
	brfFile string
	// The configuration for this BriefCmd.
	Config config.Config
}

// Configure configures the command line parser for this BriefCmd
//
// kingpin is used for commandline parsing and therefore the only accepted
// parameter is a pointer to a kingpin.Application.
func (c *PreviewCommand) Configure(app *kingpin.Application) {
	preview := app.Command("preview", "Convert the given <brfFile> into a preview file (via tex).").Action(c.run)
	preview.Arg("brfFile", "brf file to convert.").Required().StringVar(&c.brfFile)
}

// Run executes this BriefCmd
//
// kingpin is used for commandline parsing and therefore the only
// accepted parameter is a pointer to a kingpin.ParseContext.
func (c *PreviewCommand) run(ctx *kingpin.ParseContext) error {
	if c.Config.PreviewCommand == "" {
		return fmt.Errorf("No preview command configured. Cannot open preview.")
	}

	pdfFile := utils.DeriveFilePath(c.brfFile, "pdf")

	// create the PDF file first, if necessary
	needsPdfRun := !IsNewerThan(pdfFile, c.brfFile)
	if needsPdfRun {
		//FIXME: We already create a PdfCommand in the main method.
		//       We should reuse that one.
		pdfCmd := &PdfCommand{brfFile: c.brfFile, Config: c.Config}
		err := pdfCmd.run(nil)
		if err != nil {
			return fmt.Errorf("Cannot create pdf file for %s: %w", c.brfFile, err)
		}
	}

	// execute the Previewer
	cmdLine := c.Config.PreviewCommand + " " + pdfFile
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	err := cmdline.Execute(cmdLine, "", nil, stdout, stderr)
	if err != nil {
		return fmt.Errorf("Error opening previewer for %s via %s, %w", c.brfFile, cmdLine, err)
	}

	return nil
}
