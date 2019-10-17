package utils

import (
	"testing"
)

func TestDeriveFilePath(t *testing.T) {
	cases := []struct {
		input  string
		newExt string
		output string
	}{
		{"/my/path/file.brf", "tex", "/my/path/file.tex"},
		{"/my/path/file.brf", ".tex", "/my/path/file.tex"},
		{"file.brf", ".tex", "file.tex"},
	}
	for _, tt := range cases {
		newFilePath := DeriveFilePath(tt.input, tt.newExt)
		if newFilePath != tt.output {
			t.Errorf("Parse(%q, %q) == %q, expected %q", tt.input, tt.newExt, newFilePath, tt.output)
		}
	}
}
