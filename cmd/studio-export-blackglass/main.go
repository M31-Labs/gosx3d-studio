// studio-export-blackglass writes the canonical Studio SceneDoc used by the
// GoSX Blackglass Coast showcase. It is intentionally tiny: all validation,
// determinism, and loss accounting live in internal/studio.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"m31labs.dev/gosx3d-studio/internal/studio"
)

func main() {
	output := flag.String("out", "", "required output .scene3d path")
	flag.Parse()
	if *output == "" {
		fatalf("-out is required")
	}
	payload, report, err := studio.ExportSceneDoc(studio.BlackglassCoastDocument())
	if err != nil {
		fatalf("export Blackglass Coast: %v", err)
	}
	if err := os.WriteFile(*output, payload, 0o644); err != nil {
		fatalf("write %s: %v", *output, err)
	}
	encoder := json.NewEncoder(os.Stderr)
	if err := encoder.Encode(report); err != nil {
		fatalf("report: %v", err)
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "studio-export-blackglass: "+format+"\n", args...)
	os.Exit(1)
}
