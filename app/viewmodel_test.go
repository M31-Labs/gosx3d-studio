package app

import (
	"strings"
	"testing"
	"time"

	"m31labs.dev/gosx3d-studio/internal/studio"
)

func TestHierarchyAndInspectorReflectCanonicalDocument(t *testing.T) {
	document := studio.SampleDocument()
	selected, _ := studio.FirstPickTarget(document)
	hierarchy := hierarchyView(document, selected)
	if len(hierarchy) != len(document.Entities) {
		t.Fatalf("hierarchy = %d, entities = %d", len(hierarchy), len(document.Entities))
	}
	found := false
	for _, item := range hierarchy {
		if item["id"] == string(selected) {
			found = true
			if !strings.Contains(item["class"].(string), "selected") {
				t.Fatalf("selected class = %q", item["class"])
			}
		}
	}
	if !found {
		t.Fatalf("selected entity %q missing", selected)
	}
	inspector := inspectorView(document, selected)
	if inspector["id"] != string(selected) || inspector["material"] != "Coral Pieces" {
		t.Fatalf("inspector = %#v", inspector)
	}
}

// The evidence suite renders frames and builds workspaces. Measured on the
// sample document it costs about 2.3 seconds, and every edit changes the
// revision, so running it inline stalled the editor once per edit. The render
// path must return without waiting for it, and the card must say which
// document the evidence it shows describes.
func TestCertificationViewDoesNotBlockTheRenderPath(t *testing.T) {
	document := studio.SampleDocument()

	liveCertCache.Lock()
	liveCertCache.view, liveCertCache.running, liveCertCache.fingerprint, liveCertCache.revision = nil, "", "", 0
	liveCertCache.Unlock()

	start := time.Now()
	view := liveCertificationView(document)
	elapsed := time.Since(start)

	// The suite itself takes seconds. A render that waited for it would not
	// come back inside this budget.
	if elapsed > 500*time.Millisecond {
		t.Fatalf("render path took %v; the evidence suite is still running inline", elapsed)
	}
	if view["certState"] != "pending" {
		t.Fatalf("first render certState = %v, want pending", view["certState"])
	}

	deadline := time.Now().Add(60 * time.Second)
	for {
		view = liveCertificationView(document)
		if view["certState"] == "current" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("background evidence never became current; last state = %v", view["certState"])
		}
		time.Sleep(20 * time.Millisecond)
	}
	if view["certRevision"] == "" || view["liveChecksTotal"] == "0" {
		t.Fatalf("current evidence view = %#v", view)
	}
}

// The evidence suite runs on its own goroutine, and an unrecovered panic on
// any goroutine takes the whole process with it. The suite drives edge cases
// and indexes results it did not produce, so it is exactly the code most
// likely to panic. Losing the editor because a check could not be taken is a
// worse outcome than reporting that the check failed.
func TestCertificationPanicIsReportedRatherThanFatal(t *testing.T) {
	original := certifyCurrent
	t.Cleanup(func() {
		certifyCurrent = original
		liveCertCache.Lock()
		liveCertCache.view, liveCertCache.running, liveCertCache.fingerprint, liveCertCache.revision = nil, "", "", 0
		liveCertCache.Unlock()
	})
	certifyCurrent = func(studio.Document) (studio.EvidenceReport, error) {
		var records []int
		_ = records[3] // index out of range, exactly like the real defects fixed alongside this
		return studio.EvidenceReport{}, nil
	}

	liveCertCache.Lock()
	liveCertCache.view, liveCertCache.running, liveCertCache.fingerprint, liveCertCache.revision = nil, "", "", 0
	liveCertCache.Unlock()

	document := studio.SampleDocument()
	fingerprint, err := document.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}

	// Without the recover this call ends the test binary rather than
	// returning, so reaching the next line is the assertion.
	runCertification(document, fingerprint)

	liveCertCache.Lock()
	view, running := liveCertCache.view, liveCertCache.running
	liveCertCache.Unlock()
	if view == nil {
		t.Fatal("a panicking run published no view")
	}
	if running != "" {
		t.Fatalf("a panicking run left the card wedged as recomputing (running = %q)", running)
	}
	status, _ := view["releaseStatus"].(string)
	if !strings.Contains(status, "panicked") {
		t.Fatalf("releaseStatus = %q, want it to name the panic", status)
	}
	// The card must report the panic rather than a passing evidence count.
	if view["liveChecksPass"] != "0" || view["liveChecksTotal"] != "0" {
		t.Fatalf("panicking run reported checks %v/%v", view["liveChecksPass"], view["liveChecksTotal"])
	}
}

func TestTimelineViewReflectsCanonicalRigClipAndSimulation(t *testing.T) {
	view := timelineView(studio.ArticulatedProofDocument())
	if view["armatureId"] != "arm" || view["clipId"] != "idle" || view["trackId"] != "idle-lower" || view["simulationId"] != "articulated-physics" || view["tickRate"] != "60" || view["retargetMapId"] != "arm-to-tall" || view["machineId"] != "locomotion" || view["machineParameter"] != "speed" {
		t.Fatalf("timeline view = %#v", view)
	}
}

// The card reads "recomputing" after every edit and had no way back to
// "current" until something else caused a render. The state a client polls has
// to answer honestly at each stage, and must never claim "current" for a
// document the published evidence does not describe.
func TestCertificationEvidenceStateTracksTheDocument(t *testing.T) {
	original := certifyCurrent
	t.Cleanup(func() {
		certifyCurrent = original
		liveCertCache.Lock()
		liveCertCache.view, liveCertCache.running, liveCertCache.fingerprint, liveCertCache.revision = nil, "", "", 0
		liveCertCache.Unlock()
	})

	liveCertCache.Lock()
	liveCertCache.view, liveCertCache.running, liveCertCache.fingerprint, liveCertCache.revision = nil, "", "", 0
	liveCertCache.Unlock()

	document := studio.SampleDocument()
	if state := CertificationEvidenceStateFor(document.Revision); state.State != "pending" {
		t.Fatalf("state before any run = %q, want pending", state.State)
	}

	fingerprint, err := document.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	runCertification(document, fingerprint)

	state := CertificationEvidenceStateFor(document.Revision)
	if state.State != "current" {
		t.Fatalf("state after a completed run = %q, want current", state.State)
	}
	if state.Revision != document.Revision || state.DocumentRevision != document.Revision {
		t.Fatalf("state revisions = %d/%d, want both %d", state.Revision, state.DocumentRevision, document.Revision)
	}
	if state.Schema == "" {
		t.Fatal("state carries no schema")
	}

	// A newer document must read as recomputing, and must report the older
	// revision its evidence actually describes rather than the current one.
	ahead := CertificationEvidenceStateFor(document.Revision + 1)
	if ahead.State != "recomputing" {
		t.Fatalf("state for a newer document = %q, want recomputing", ahead.State)
	}
	if ahead.Revision != document.Revision {
		t.Fatalf("evidence revision = %d, want the %d it describes", ahead.Revision, document.Revision)
	}
	if ahead.DocumentRevision != document.Revision+1 {
		t.Fatalf("document revision = %d, want %d", ahead.DocumentRevision, document.Revision+1)
	}
}
