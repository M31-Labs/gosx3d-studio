package studio

import (
	"bytes"
	"encoding/json"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"
)

func TestExportEmbeddedGoIsValidDeterministicAndLossless(t *testing.T) {
	document := SampleDocument()
	payload, report, err := ExportEmbeddedGo(document, "checkers")
	if err != nil {
		t.Fatal(err)
	}
	source := string(payload)
	if !strings.Contains(source, "DO NOT EDIT") || !strings.Contains(source, "package checkers") {
		t.Fatalf("generated header wrong:\n%s", source[:200])
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "generated.go", payload, 0); err != nil {
		t.Fatalf("generated file is not valid Go: %v", err)
	}
	if report.Kind != "go-embed" || len(report.Losses) != 0 {
		t.Fatalf("embedded Go export must be lossless: %+v", report)
	}
	// The embedded JSON must round-trip to the same document fingerprint.
	start := strings.Index(source, "const DocumentJSON = ")
	if start < 0 {
		t.Fatal("DocumentJSON constant missing")
	}
	quoted := strings.TrimSpace(strings.TrimPrefix(source[start:], "const DocumentJSON = "))
	if end := strings.IndexByte(quoted, '\n'); end > 0 {
		quoted = quoted[:end]
	}
	unquoted, err := strconv.Unquote(quoted)
	if err != nil {
		t.Fatalf("DocumentJSON is not a valid Go string literal: %v", err)
	}
	var decoded Document
	if err := json.Unmarshal([]byte(unquoted), &decoded); err != nil {
		t.Fatal(err)
	}
	source1, _ := document.Fingerprint()
	source2, _ := decoded.Fingerprint()
	if source1 != source2 {
		t.Fatal("embedded document changed fingerprint")
	}
	if !strings.Contains(source, "EntityBoard") || !strings.Contains(source, `= "board"`) {
		t.Fatal("stable entity ID constants missing")
	}
	second, _, err := ExportEmbeddedGo(document, "checkers")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(payload, second) {
		t.Fatal("embedded Go export is not byte-deterministic")
	}
}
