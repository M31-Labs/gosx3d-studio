package studio

import (
	"strings"
	"testing"
)

func TestActionDescriptorsCarryOutputSchemaAndReadSurface(t *testing.T) {
	descriptors := ActionDescriptors()
	reads, mutates := 0, 0
	for _, descriptor := range descriptors {
		if descriptor.OutputSchema == nil {
			t.Fatalf("descriptor %q has no output schema", descriptor.Name)
		}
		switch descriptor.Access {
		case "read":
			reads++
			if len(descriptor.AuthorityModes) != 0 {
				t.Fatalf("read descriptor %q must not require transaction authority", descriptor.Name)
			}
		case "mutate":
			mutates++
			properties, ok := descriptor.OutputSchema["properties"].(map[string]any)
			if !ok || properties["afterRevision"] == nil || properties["transactionId"] == nil {
				t.Fatalf("mutating descriptor %q output schema must describe the receipt, got %+v", descriptor.Name, descriptor.OutputSchema)
			}
		default:
			t.Fatalf("descriptor %q has unknown access %q", descriptor.Name, descriptor.Access)
		}
	}
	if mutates == 0 || reads < 5 {
		t.Fatalf("descriptor surface incomplete: %d mutate, %d read", mutates, reads)
	}
}

func TestInitializeReportDescribesProtocolAndDocument(t *testing.T) {
	document := SampleDocument()
	report, err := BuildInitializeReport(document)
	if err != nil {
		t.Fatal(err)
	}
	if report.Protocol != "gosx.studio3d.actions/v1" || report.SchemaVersion == "" {
		t.Fatalf("report=%+v", report)
	}
	if report.Document.ID != string(document.ID) || report.Document.Revision != document.Revision || len(report.Document.Fingerprint) != 64 {
		t.Fatalf("document identity=%+v", report.Document)
	}
	if report.Actions.Mutating == 0 || report.Actions.Read == 0 {
		t.Fatalf("actions=%+v", report.Actions)
	}
	if !strings.Contains(strings.Join(report.AuthorityModes, ","), "propose") {
		t.Fatalf("authority modes=%v", report.AuthorityModes)
	}
}
