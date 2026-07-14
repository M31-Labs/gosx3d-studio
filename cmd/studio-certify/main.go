package main

import (
	"flag"
	"fmt"
	"os"

	"m31labs.dev/gosx3d-studio/internal/studio"
)

func main() {
	output := flag.String("output", "", "optional deterministic certification JSON output; stdout when empty")
	flag.Parse()
	report, err := studio.CertifyCurrent(studio.SampleDocument())
	if err != nil {
		fatal(err)
	}
	data, err := report.MarshalDeterministic()
	if err != nil {
		fatal(err)
	}
	data = append(data, '\n')
	if *output == "" {
		_, err = os.Stdout.Write(data)
	} else {
		err = os.WriteFile(*output, data, 0o600)
	}
	if err != nil {
		fatal(err)
	}
	if !report.Valid {
		fatal(fmt.Errorf("studio certification contains failed checks"))
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "studio-certify:", err)
	os.Exit(1)
}
