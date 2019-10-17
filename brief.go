package main

import (
	"os"

	"gopkg.in/alecthomas/kingpin.v2"
	"poiu.de/brief/cmd"
	"poiu.de/brief/config"
)

func prepareCommands() {
	app := kingpin.New("brief", "User friendly creation of high quality letters.")

	cfg := config.NewConfig()

	//TODO: Put into cmd/tex.go#init()?
	//      Doesn't work. We would still need the reference to 'app'
	texCmd := &cmd.TexCommand{Config: *cfg}
	texCmd.Configure(app)

	pdfCmd := &cmd.PdfCommand{Config: *cfg}
	pdfCmd.Configure(app)

	previewCmd := &cmd.PreviewCommand{Config: *cfg}
	previewCmd.Configure(app)

	kingpin.MustParse(app.Parse(os.Args[1:]))
}

func main() {
	prepareCommands()
}
