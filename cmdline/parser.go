/*
 * Package cmdline contains structs and functions for calling external
 * commands.
 */
package cmdline

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
)

type Parser struct {
}

// Cmd specifies an external command that can be called via
// os.exec.Command.
type Cmd struct {
	CmdName string
	Args    []string
}

// newCmd creates a new Cmd from the given slice of tokens.
//
// The first token is used as the command name, all other tokens as
// arguments to that command.
func newCmd(tokens []string) *Cmd {
	cmd := new(Cmd)
	if len(tokens) > 0 {
		cmd.CmdName = tokens[0]
	}
	if len(tokens) > 1 {
		cmd.Args = tokens[1:]
	}
	return cmd
}

type parser interface {
	Parse(s string) []string
}

// NewParser creates a new parser for parsing executable commands +
// arguments.
func NewParser() *Parser {
	return new(Parser)
}

// Parse tokenizes the given string with a command line into
// the command and its arguments.
//
// It splits the tokens by whitespace. To include whitespace inside a
// token enclose the token in either single (') or double (") quotes.
//
// The resulting slice should be usable to feed it into os/exec.Command
// like:
//   p := cmdline.NewParser()
//   cmdLineTokens := p.Parse("mycmd arg1 'arg two'")
//   exec.Command(cmdLineTokens...)
func (p *Parser) Parse(s string) []Cmd {
	cmds := make([]Cmd, 0)
	tokens := make([]string, 0)
	word := make([]rune, 0)

	var quoteChar rune
	for _, c := range s {
		switch c {
		case '\'', '"':
			if quoteChar == c {
				quoteChar = 0
			} else {
				quoteChar = c
			}
		case ' ', '\t':
			if quoteChar != 0 {
				word = append(word, c)
			} else if len(word) > 0 {
				tokens = append(tokens, string(word))
				word = make([]rune, 0)
			} else {
				// ignore multiple consecutive whitespace characters
			}
		case '|':
			if quoteChar != 0 {
				word = append(word, c)
			} else {
				//TODO: Start new Cmd
				cmd := newCmd(tokens)
				cmds = append(cmds, *cmd)
				tokens = make([]string, 0)
			}
		default:
			word = append(word, c)
		}
	}

	if len(word) > 0 {
		tokens = append(tokens, string(word))
	}

	if len(tokens) > 0 {
		cmd := newCmd(tokens)
		cmds = append(cmds, *cmd)
	}

	return cmds
}

// Execute executes the given cmdLine.
//
// The cmdLine is a string containing a command and arguments to that
// command, separated by whitespace.
// If an argument contains whitespace it needs to be enclosed in single or
// double quotes, for example: `ls -lha "my file with whitespace"`.
//
// Execute supports pipes between commands. Use the pipe symbol to indicate
// that the stdout of one commands needs to be piped to stdin of another
// command, for example: `cat myFile | wc -l`
//
// If necessary a working dir may be given. This directory will be switched
// to before executing the first command. After the last command finished
// the working directory will be reverted to the one before this function
// was called.
// If it is an empty string, the working directory will not be changed.
//
// If necessary, stdin, stdout and stderr may be given to attach to the
// command. They may be nil.
// When multiple piped commands are given in the cmdLine, stdin will be
// attached to the first, stdout to the last and stderr to all commands.
func Execute(cmdLine string, workingDir string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	// change working directory if necessary
	if workingDir != "" {
		curWorkingDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("Cannot determince current working directory. Not switching to %s: %w", workingDir, err)
		}

		defer func() {
			err := os.Chdir(curWorkingDir)
			if err != nil {
				//FIXME: Only log a warning message? Or return a specific error type as Warning?
				log.Println(fmt.Errorf("Cannot change working directory back from %s to %s: %w", workingDir, curWorkingDir, err))
			}
		}()

		err = os.Chdir(workingDir)
		if err != nil {
			return fmt.Errorf("Cannot change working directory to %s: %w", workingDir, err)
		}
	}

	p := NewParser()
	cmds := p.Parse(cmdLine)

	commands := make([]*exec.Cmd, len(cmds))

	// prepare the actual Command objects to execute
	for i, cmd := range cmds {
		commands[i] = exec.Command(cmd.CmdName, cmd.Args...)
		// write stderr to each command
		commands[i].Stderr = stderr
	}
	// wire stdin to the first and stdout to the last Command
	commands[0].Stdin = stdin
	commands[len(commands)-1].Stdout = stdout

	// connect multiple commands via pipe
	for i := 1; i < len(commands); i++ {
		stdoutPipe, err := commands[i-1].StdoutPipe()
		if err != nil {
			return fmt.Errorf("Error wiring multiple commands via pipe: %w", err)
		}
		commands[i].Stdin = stdoutPipe
	}

	// now execute all the commands
	for _, command := range commands {
		err := command.Start()
		if err != nil {
			return fmt.Errorf("Error executing commands %s: %w", cmdLine, err)
		}
	}
	for i := len(commands) - 1; i >= 0; i-- {
		err := commands[i].Wait()
		if err != nil {
			return fmt.Errorf("Error executing commands %s: %w", cmdLine, err)
		}
	}

	return nil
}
