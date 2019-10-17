package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"text/template"

	"gopkg.in/alecthomas/kingpin.v2"
	"poiu.de/brief/address"
	"poiu.de/brief/config"
	"poiu.de/brief/markup"
	"poiu.de/brief/parser"
	"poiu.de/brief/utils"
)

var (
	// Regex for a line indicating the markup language for a whole section
	patternMarkupType *regexp.Regexp = regexp.MustCompile(`^\.([a-z]+)\s*$`)
	// Regex for a line indicating the start of a markup block
	patternMarkupTypeBlockStart *regexp.Regexp = regexp.MustCompile(`^\.([a-z]+)-{2,}\s*$`)
	// Regex for a line indicating the end of a markup block
	patternMarkupTypeBlockEnd *regexp.Regexp = regexp.MustCompile(`^-{2,}\s*$`)

	// Replacer to replace special characters with their LaTeX equivalents
	// FIXME: Find a better name, since it is not only about Unicode
	utfCharReplacer = strings.NewReplacer(
		` `, `\,`, // thin space
		`_`, "\\string_", // underscore character
		`^`, "\\string^", // caret character
	)

	inlineMarkupReplacer = strings.NewReplacer(
		`/`, `\/`, // italics
	)
)

// TexCommand calls an external application to generate the TeX file for a
// .brf file.
//
// If utilized the configured markup converters to convert markup blocks
// (like markdown or asciidoc) to LaTeX code.
type TexCommand struct {
	// The .brf file for which to generate the TeX file.
	brfFile string
	// The configuration for this BriefCmd.
	Config config.Config
}

// Configure configures the command line parser for this BriefCmd
//
// kingpin is used for commandline parsing and therefore the only
// accepted parameter is a pointer to a kingpin.Application.
func (c *TexCommand) Configure(app *kingpin.Application) {
	tex := app.Command("tex", "Convert the given <brfFile> into a tex file.").Action(c.run)
	tex.Arg("brfFile", "brf file to convert.").Required().StringVar(&c.brfFile)
}

// Run executes this BriefCmd
//
// kingpin is used for commandline parsing and therefore the only
// accepted parameter is a pointer to a kingpin.ParseContext.
func (c *TexCommand) run(ctx *kingpin.ParseContext) error {
	brf, err := parser.ReadBrfFile(c.brfFile)
	if err != nil {
		return fmt.Errorf("Invalid brf source file: %w", err)
	}

	//TODO: Check validity of brf. Only proceed if valid
	//TODO: These section names should be specified in an enum
	texTemplate, err := getSingleValue(brf.Sections["TEMPLATE"])
	if err != nil {
		return fmt.Errorf("Invalid tex template name: %s, \n%w", brf.Sections["TEMPLATE"], err)
	}

	tmpl, err := template.ParseFiles(c.Config.TexTemplateDir + "/" + texTemplate)
	if err != nil {
		return fmt.Errorf("Error reading tex template %s: %w", c.Config.TexTemplateDir+"/"+texTemplate, err)
	}

	senderAddresses, err := address.ReadFromFile(c.Config.SenderList)
	if err != nil {
		return fmt.Errorf("Error reading sender list from %s: %w", c.Config.SenderList, err)
	}

	fromAddress, err := getSingleValue(brf.Sections["FROM"])
	if err != nil {
		return fmt.Errorf("Invalid from-address name %s: %w", brf.Sections["FROM"], err)
	}

	senderInput := senderToTemplateInput(senderAddresses[fromAddress])
	contentInput := c.contentToTemplateInput(brf)
	mergedInput := merge(contentInput, senderInput)

	//prepare the target file
	outFile := utils.DeriveFilePath(c.brfFile, "tex")
	f, err := os.Create(outFile)
	if err != nil {
		return fmt.Errorf("Could create output file %s: %w", outFile, err)
	}
	defer f.Close()

	//write to target file (using template)
	w := bufio.NewWriter(f)
	err = tmpl.Execute(w, mergedInput)
	if err != nil {
		return fmt.Errorf("Error converting %s: %w", c.brfFile, err)
	}

	w.Flush()

	return nil
}

// senderToTemplateInput converts an Address into a map of key-value-pairs
// suitable to be filled into a brief template.
func senderToTemplateInput(address address.Address) map[string]string {
	r := make(map[string]string)
	for k, v := range address.Fields {
		r[k] = strings.Join(v, "\\\\\n")
		r[k] = utfCharReplacer.Replace(r[k])
	}

	return r
}

//FIXME: Find a better name.
// contentToTemplateInput converts the content of the sections of a .brf file into
// a text suitable to be filled into a brief template.
//
// It does this by converting any markup in the CONTENT section into LaTeX
// code, replacing certain special characters into LaTeX equivalents and
// replacing all newlines in sections other than CONTENT into double
// backslashes (+ newline) to force a line break in LaTeX.
//
// The returned map contains the section names as the key and their content
// as the corresponding value.
func (c *TexCommand) contentToTemplateInput(brf parser.Brf) map[string]string {
	r := make(map[string]string)
	for k, v := range brf.Sections {
		v = trimSurroundingEmptyLines(v)

		// all sections except CONTENT get newlines replaced with double
		// backslashes
		if k != "CONTENT" {
			r[k] = strings.Join(v, "\\\\\n")
		} else {
			if len(v) > 0 {
				sectionMarkupType := patternMarkupType.FindStringSubmatch(v[0])
				if sectionMarkupType != nil {
					// the whole section gets a single markup type
					mc, err := markup.NewConverter(sectionMarkupType[1], &c.Config)
					if err != nil {
						log.Println(fmt.Errorf("Cannot convert markup %s. Leaving as is. %w", sectionMarkupType[1], err))
						r[k] = strings.Join(v, "\n")
					} else {
						r[k], err = mc.Convert(v[1:])
						if err != nil {
							log.Println(fmt.Errorf("Cannot convert markup %s. Leaving as is. %w", sectionMarkupType[1], err))
							r[k] = strings.Join(v, "\n")
						}
					}
				} else {
					// join all lines with \n and convert markup blocks
					r[k] = c.convertMarkup(v)
				}
			}
		}

		// Replace special characters to their LaTeX equivalents
		r[k] = utfCharReplacer.Replace(r[k])

		// TODO: Replace some brief-specific markup into LaTeX
		//r[k] = inlineMarkupReplacer.Replace(r[k])
	}

	return r
}

// convertMarkup joins the given slice of strings with \n into a single
// string.
// If markup blocks are found in the given slice, those will be converted
// first and then joined.
// If markup conversion fails those lines will be joined as is (with the
// surrounding markup block separators).
func (c *TexCommand) convertMarkup(a []string) string {
	sep := "\n"
	switch len(a) {
	case 0:
		return ""
	case 1:
		return a[0]
	}

	// Calculate the necessary size for the Builder.
	// This isn't accurate when there are markup blocks. But then the worst
	// case is one new allocation per markup block.
	n := len(sep) * (len(a) - 1)
	for i := 0; i < len(a); i++ {
		n += len(a[i])
	}

	var b strings.Builder
	b.Grow(n)
	for i := 0; i < len(a); i++ {
		s := a[i]
		// if this line is the start of a markup block, convert that block
		markupBlockType := patternMarkupTypeBlockStart.FindStringSubmatch(s)
		if markupBlockType != nil {
			markupBlock := make([]string, len(a)-i)
			blockEnd := -1
			for x := i; x < len(a); x++ {
				l := a[x]
				blockEndFound := patternMarkupTypeBlockEnd.MatchString(l)
				if blockEndFound {
					blockEnd = x
					break
				} else {
					markupBlock = append(markupBlock, l)
				}
			}
			// if there was block end found, convert the block, otherwise write
			// it unconverted
			if blockEnd != -1 {
				c, err := markup.NewConverter(markupBlockType[1], &c.Config)
				if err != nil {
					log.Println(fmt.Errorf("Cannot convert markup %s. Leaving as is. %w", markupBlockType[1], err))
					b.WriteString(strings.Join(a[i:blockEnd], sep))
					b.WriteString(sep)
				} else {
					r, err := c.Convert(a[i+1 : blockEnd-1])
					if err != nil {
						log.Println(fmt.Errorf("Cannot convert markup %s. Leaving as is. %w", markupBlockType[1], err))
						b.WriteString(strings.Join(a[i:blockEnd], sep))
						b.WriteString(sep)
					} else {
						b.WriteString(r)
						b.WriteString(sep)
						i = blockEnd
					}
				}
			} else {
				// no markup block end
				b.WriteString(strings.Join(a[i:], sep))
				b.WriteString(sep)
			}
		} else {
			// normal line (not inside markup block)
			b.WriteString(s)
			b.WriteString(sep)
		}
	}

	return b.String()
}

// getSingleValue returns a string with the content of the only non-empty
// line in the given BrfLines.
// If no or more than one non-empty line is contained in the given BrfLines
// an error will be returned.
func getSingleValue(brfLines parser.BrfLines) (string, error) {
	brfLines = trimSurroundingEmptyLines(brfLines)
	if len(brfLines) == 0 {
		return "", errors.New("No line with content found")
	} else if len(brfLines) > 1 {
		return "", errors.New("More than one line with content found")
	} else {
		return strings.TrimSpace(brfLines[0]), nil
	}
}

// trimSurroundingEmptyLines returns a slice of the given BrfLines with
// leading and trailing empty lines removed.
// Empty lines in between remain as is.
func trimSurroundingEmptyLines(brfLines parser.BrfLines) parser.BrfLines {
	firstNonEmpty := -1
	lastNonEmpty := -1
	for idx, line := range brfLines {
		if strings.TrimSpace(line) != "" {
			lastNonEmpty = idx
			if firstNonEmpty == -1 {
				firstNonEmpty = idx
			}
		}
	}

	return brfLines[firstNonEmpty : lastNonEmpty+1]
}

// merge puts all entries in the given maps into a new map.
// The original maps will not be altered
// Keys that exist in both given maps get the value of the second one.
func merge(m1, m2 map[string]string) map[string]string {
	m := make(map[string]string, len(m1)+len(m2))

	for k, v := range m1 {
		m[k] = v
	}

	for k, v := range m2 {
		m[k] = v
	}

	return m
}
