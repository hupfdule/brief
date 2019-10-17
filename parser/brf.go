package parser

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
)

var (
	// Regex for a line indicating the start of a new section in a .brf file
	patternSectionId *regexp.Regexp = regexp.MustCompile(`^\.([A-Z]+)\s*$`)
)

// BrfLines is a slice of strings where each string represents a single
// line in the source .brf file.
type BrfLines []string

// Brf contains the content of a .brf file as a BrfLines object and a map
// with all sections in the .brf file referring to their respective lines.
type Brf struct {
	// Lines represent all the lines in a .brf file.
	Lines BrfLines
	// Sections contains all sections defined in a .brf file as keys and the
	// lines corresponding to the sections as the value.
	Sections map[string]BrfLines
}

// NewBrf creates a new Brf object for the given lines.
// The sections of the Brf object will not be filled automatically.
//
// Use ReadBrfFile(string) to read a fully initialized
func NewBrf(lines BrfLines) Brf {
	brf := Brf{}
	brf.Lines = lines
	brf.Sections = map[string]BrfLines{}
	return brf
}

// ReadBrfFile reads the given .brf file and converts its content into a
// Brf object.
func ReadBrfFile(brfFilePath string) (Brf, error) {
	var brf Brf

	file, err := os.Open(brfFilePath)
	if err != nil {
		return brf, fmt.Errorf("Error opening %s: %w", brfFilePath, err)
	}
	defer file.Close()

	var brfLines BrfLines

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		brfLines = append(brfLines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return brf, fmt.Errorf("Error reading %s: %w", brfFilePath, err)
	}

	brf = CreateBrf(brfLines)
	return brf, nil
}

// CreateBrf creates a Brf object from the given lines of a .brf file.
func CreateBrf(brfLines BrfLines) Brf {
	brf := NewBrf(brfLines)

	var currentSection *string = nil
	var currentSectionStart int = -1
	for idx, line := range brfLines {
		sectionIdMatch := patternSectionId.FindStringSubmatch(line)
		if sectionIdMatch != nil {
			// if a new section starts, everything up to here is the content of
			// the previous section
			if currentSection != nil {
				brf.Sections[*currentSection] = brfLines[currentSectionStart+1 : idx]
			} else {
				//TODO: Warn if there were content lines that don't belong to any
				//section?
			}
			currentSection = &sectionIdMatch[1]
			currentSectionStart = idx
		}
	}
	// the remainder of the content is the content of the last section
	if currentSection != nil {
		brf.Sections[*currentSection] = brfLines[currentSectionStart+1:]
	}

	return brf
}
