package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/alecthomas/kingpin.v2"
	"poiu.de/brief/cmdline"
	"poiu.de/brief/config"
	"poiu.de/brief/utils"
)

// PdfCommand calls an external application to generate the PDF file for a
// .brf file.
//
// If necessary it creates the corresponding TeX file by calling the
// TexCommand.
type PdfCommand struct {
	// The .brf file for which to generate the PDF.
	brfFile string
	// The configuration for this BriefCmd.
	Config config.Config
}

// Configure configures the command line parser for this BriefCmd
//
// kingpin is used for commandline parsing and therefore the only
// accepted parameter is a pointer to a kingpin.Application.
func (c *PdfCommand) Configure(app *kingpin.Application) {
	pdf := app.Command("pdf", "Convert the given <brfFile> into a pdf file (via tex).").Action(c.run)
	pdf.Arg("brfFile", "brf file to convert.").Required().StringVar(&c.brfFile)
}

// Run executes this BriefCmd
//
// kingpin is used for commandline parsing and therefore the only
// accepted parameter is a pointer to a kingpin.ParseContext.
func (c *PdfCommand) run(ctx *kingpin.ParseContext) error {
	if c.Config.PdfCommand == "" {
		return fmt.Errorf("No pdf command configured. Cannot produce pdf file.")
	}

	texFile := utils.DeriveFilePath(c.brfFile, "tex")

	// create the TeX file first, if necessary
	needsTexRun := !IsNewerThan(texFile, c.brfFile)
	if needsTexRun {
		//FIXME: We already create a TexCommand in the main method.
		//       We should reuse that one.
		texCmd := &TexCommand{brfFile: c.brfFile, Config: c.Config}
		err := texCmd.run(nil)
		if err != nil {
			return fmt.Errorf("Cannot create tex file for %s: %w", c.brfFile, err)
		}
	}

	// now execute the command to generate the pdf
	cmdLine := c.Config.PdfCommand + " " + texFile
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	err := cmdline.Execute(cmdLine, filepath.Dir(c.brfFile), nil, stdout, stderr)
	if err != nil {
		return fmt.Errorf("Error generating pdf file for %s via %s, %w", c.brfFile, cmdLine, err)
	}

	return nil
}

// IsNewerThan check whether file1 is newer than file2.
//
// If file2 does not exists, 'true' is returned.
// If file1 does not exist, but file2 does, 'false' is returned.
// If both files don't exist, true is returned.
// Otherwise this returns 'true' if the modification time of file1 is newer
// than the modification time of file2.
func IsNewerThan(file1, file2 string) bool {
	//TODO: We check here only for errors, not specific NotExistError.
	//      Should we change that? What should happen on other errors?
	info2, err := os.Stat(file2)
	if err != nil {
		return true
	}

	info1, err := os.Stat(file1)
	if err != nil {
		return false
	}

	mTime1 := info1.ModTime()
	mTime2 := info2.ModTime()

	return mTime1.After(mTime2)
}
