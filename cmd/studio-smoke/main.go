package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"m31labs.dev/gosx/scene"
	"m31labs.dev/gosx/scene/harness"
	"m31labs.dev/gosx/scene/preview"
	"m31labs.dev/gosx3d-studio/internal/studio"
)

func main() {
	pngPath := flag.String("png", "", "optional deterministic preview PNG output")
	reportPath := flag.String("report", "", "optional harness JSON output; stdout when empty")
	flag.Parse()
	document := studio.SampleDocument()
	props, err := studio.Compile(document)
	fatalIf(err)
	session := harness.New(props, preview.Options{Width: 720, Height: 480, Background: document.Environment.Background, DisablePostFX: true, MaxSegments: 24})
	result, err := session.Render(0)
	fatalIf(err)
	report := session.Report()
	if len(report.Events) == 0 || report.Events[0].Frame == nil || report.Events[0].Frame.Coverage <= 0.001 {
		fatalIf(fmt.Errorf("native frame is blank or missing evidence"))
	}
	trace := session.Trace("center-piece", scene.Ray{Origin: scene.Vec3(0, 4, 0), Direction: scene.Vec3(0, -1, 0)}, scene.PickableOnly())
	if trace.Closest == nil || trace.Closest.ID != "piece-jade-01" {
		fatalIf(fmt.Errorf("exact center trace hit %+v", trace.Closest))
	}
	fatalIf(session.Validate())
	if *pngPath != "" {
		file, err := os.Create(*pngPath)
		fatalIf(err)
		fatalIf(preview.WritePNG(file, result))
		fatalIf(file.Close())
	}
	if *reportPath == "" {
		fatalIf(session.WriteJSON(os.Stdout))
		return
	}
	file, err := os.Create(*reportPath)
	fatalIf(err)
	encoder := json.NewEncoder(file)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	fatalIf(encoder.Encode(session.Report()))
	fatalIf(file.Close())
}

func fatalIf(err error) {
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, "studio-smoke:", err)
	os.Exit(1)
}
