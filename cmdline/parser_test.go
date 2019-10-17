package cmdline

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	parser := NewParser()

	cases := []struct {
		input string
		cmd   []Cmd
	}{
		{"cmd param1 param2", []Cmd{Cmd{"cmd", []string{"param1", "param2"}}}},
		{"cmd  param1 \t param2", []Cmd{Cmd{"cmd", []string{"param1", "param2"}}}},
		{"cmd  \"param 1\" \t 'param two'", []Cmd{Cmd{"cmd", []string{"param 1", "param two"}}}},
		{"cmd1 param1 param2 | cmd2 paramA", []Cmd{Cmd{"cmd1", []string{"param1", "param2"}}, Cmd{"cmd2", []string{"paramA"}}}},
		{"", []Cmd{}},
	}
	for _, tt := range cases {
		cmd := parser.Parse(tt.input)
		if !reflect.DeepEqual(cmd, tt.cmd) {
			t.Errorf("Parse(%q) == %q, expected %q", tt.input, cmd, tt.cmd)
		}
	}
}
