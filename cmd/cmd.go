package cmd

import "gopkg.in/alecthomas/kingpin.v2"

// BriefCmd is the base interface for all top level commands of the 'brief'
// application.
//
// BriefCmds are specified as the first argument to the application.
type BriefCmd interface {
	// Configure configures the command line parser for this BriefCmd
	//
	// kingpin is used for commandline parsing and therefore the only
	// accepted parameter is a pointer to a kingpin.Application.
	Configure(app *kingpin.Application)

	// Run executes this BriefCmd
	//
	// kingpin is used for commandline parsing and therefore the only
	// accepted parameter is a pointer to a kingpin.ParseContext.
	run(ctx *kingpin.ParseContext) error
}
