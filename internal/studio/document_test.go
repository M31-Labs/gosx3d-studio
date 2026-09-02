package studio

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"m31labs.dev/gosx/scene"
	"m31labs.dev/gosx/scene/harness"
	"m31labs.dev/gosx/scene/preview"
)

func TestSampleDocumentCompilesToSharedSceneIR(t *testing.T) {
	document := SampleDocument()
	if err := document.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	props, err := Compile(document)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if props.PreferWebGPU == nil || !*props.PreferWebGPU {
		t.Fatal("Studio scene must prefer WebGPU on capable browsers")
	}
	if props.ForceWebGL != nil && *props.ForceWebGL {
		t.Fatal("Studio scene must not force the WebGL fallback")
	}
	if props.MaxFrameRate != 60 || props.AdaptiveQuality == nil || !*props.AdaptiveQuality ||
		props.AdaptiveTargetFrameMS != 16.7 || props.AdaptiveWarmupFrames != 12 ||
		props.AdaptivePostFX == nil || !*props.AdaptivePostFX || props.WebGPUPowerPreference != "high-performance" {
		t.Fatalf("Studio interactive frame policy = %+v", props)
	}
	encodedProps, err := json.Marshal(props)
	if err != nil {
		t.Fatalf("marshal Studio Scene3D props: %v", err)
	}
	for _, field := range []string{
		`"preferWebGPU":true`, `"maxFrameRate":60`, `"adaptiveQuality":true`,
		`"adaptiveTargetFrameMS":16.7`, `"webgpuPowerPreference":"high-performance"`,
	} {
		if !bytes.Contains(encodedProps, []byte(field)) {
			t.Fatalf("Studio Scene3D props omit %s: %s", field, encodedProps)
		}
	}
	ir := props.SceneIR()
	if ir.BackendCaps == nil {
		t.Fatal("Studio SceneIR is missing backend capability evidence")
	}
	webGPUCapable := false
	for _, backend := range ir.BackendCaps.Capable {
		if backend == "webgpu" {
			webGPUCapable = true
			break
		}
	}
	if !webGPUCapable {
		t.Fatalf("Studio SceneIR does not admit WebGPU: %+v", ir.BackendCaps)
	}
	if len(ir.Objects) != 145 {
		t.Fatalf("objects = %d, want 145", len(ir.Objects))
	}
	if len(ir.Lights) != 4 {
		t.Fatalf("lights = %d, want 4", len(ir.Lights))
	}
	portableIR, err := CompileIR(document)
	if err != nil {
		t.Fatalf("compile portable IR: %v", err)
	}
	if portableIR.Schema != scene.SceneIRSchema {
		t.Fatalf("portable schema = %q, want %q", portableIR.Schema, scene.SceneIRSchema)
	}
	target, position := FirstPickTarget(document)
	trace := scene.TraceGraph(props.Graph, scene.Ray{Origin: scene.Vec3(position.X, 4, position.Z), Direction: scene.Vec3(0, -1, 0)}, scene.PickableOnly())
	if trace.Closest == nil {
		t.Fatal("center ray missed sample document")
	}
	if trace.Closest.ID != string(target) {
		t.Fatalf("closest id = %q, want %q", trace.Closest.ID, target)
	}
	if trace.PrimitivesTested == 0 || trace.NodesVisited == 0 {
		t.Fatalf("trace telemetry = %+v", trace)
	}
}

func TestCompileArtifactMapsEveryAuthoredEntity(t *testing.T) {
	document := SampleDocument()
	artifact, err := CompileWithSourceMap(document)
	if err != nil {
		t.Fatal(err)
	}
	if len(artifact.SourceMap) != len(document.Entities) {
		t.Fatalf("source map = %d, entities = %d", len(artifact.SourceMap), len(document.Entities))
	}
	if artifact.SourceMap["board"].Kind != "object" || artifact.SourceMap["key-light"].Kind != "light" {
		t.Fatalf("source map records = %#v", artifact.SourceMap)
	}
	fingerprint, _ := document.Fingerprint()
	if artifact.DocumentFingerprint != fingerprint {
		t.Fatalf("fingerprint = %q, want %q", artifact.DocumentFingerprint, fingerprint)
	}
}

func TestSampleDocumentProducesVisibleNativeEvidence(t *testing.T) {
	document := SampleDocument()
	props, err := Compile(document)
	if err != nil {
		t.Fatal(err)
	}
	session := harness.New(props, preview.Options{Width: 320, Height: 200, DisableShadows: true, DisablePostFX: true, MaxSegments: 16})
	if _, err := session.Render(0); err != nil {
		t.Fatal(err)
	}
	report := session.Report()
	if len(report.Events) != 1 || report.Events[0].Frame == nil {
		t.Fatalf("frame evidence = %+v", report.Events)
	}
	if report.Events[0].Frame.Coverage <= 0.001 || report.Events[0].Frame.UniqueColors < 4 {
		t.Fatalf("blank native evidence: %+v", *report.Events[0].Frame)
	}
	if len(report.Materials) == 0 || !report.Materials[0].Valid || report.Materials[0].WGSLFragmentHash == "" || report.Materials[0].GLSLFragmentHash == "" {
		t.Fatalf("Selena artifact evidence = %+v", report.Materials)
	}
	if err := session.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestWorkspaceProposalDirectAndUndo(t *testing.T) {
	workspace, err := NewWorkspace(SampleDocument())
	if err != nil {
		t.Fatal(err)
	}
	target, _ := FirstPickTarget(SampleDocument())
	transaction := Transaction{ID: "tx-material", Actor: "agent://test", Mode: ModePropose, ExpectedRevision: 1, Operations: []Operation{{Kind: OpAssignMaterial, Target: target, Material: "player-4-material"}}}
	receipt, preview, err := workspace.Execute(transaction)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Applied || preview.Revision != 2 || preview.Entities[target].Mesh.Material != "player-4-material" {
		t.Fatalf("proposal receipt=%+v preview=%+v", receipt, preview.Entities[target])
	}
	if receipt.InverseTransactionID == "" || receipt.TelemetryCorrelationID == "" || len(receipt.Changes) != 1 || receipt.Changes[0].Before == nil || receipt.Changes[0].After == nil {
		t.Fatalf("semantic receipt = %+v", receipt)
	}
	snapshot, _ := workspace.Snapshot()
	if snapshot.Revision != 1 || snapshot.Entities[target].Mesh.Material != "player-1-material" {
		t.Fatal("proposal mutated workspace")
	}
	transaction.Mode = ModeDirect
	receipt, _, err = workspace.Execute(transaction)
	if err != nil || !receipt.Applied {
		t.Fatalf("direct receipt=%+v err=%v", receipt, err)
	}
	if _, _, err := workspace.Execute(transaction); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("stale transaction error = %v", err)
	}
	_, restored, err := workspace.Undo(2, "identity://test")
	if err != nil {
		t.Fatal(err)
	}
	if restored.Revision != 3 || restored.Entities[target].Mesh.Material != "player-1-material" {
		t.Fatalf("restored revision=%d material=%q", restored.Revision, restored.Entities[target].Mesh.Material)
	}
	_, redone, err := workspace.Redo(3, "identity://test")
	if err != nil || redone.Revision != 4 || redone.Entities[target].Mesh.Material != "player-4-material" {
		t.Fatalf("redo revision=%d err=%v", redone.Revision, err)
	}
}

func TestExactPickAndWorkspaceSelectionAgree(t *testing.T) {
	document := SampleDocument()
	target, position := FirstPickTarget(document)
	result, err := ExactPick(document, PickRequest{Origin: Vec3{X: position.X, Y: 4, Z: position.Z}, Direction: Vec3{Y: -1}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Selected != target {
		t.Fatalf("selected = %q, want %q", result.Selected, target)
	}
	workspace, err := NewWorkspace(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := workspace.Select(result.Selected); err != nil {
		t.Fatal(err)
	}
	selection := workspace.Selection()
	if len(selection) != 1 || selection[0] != target {
		t.Fatalf("selection = %v", selection)
	}
}

func TestValidateRejectsCycles(t *testing.T) {
	document := SampleDocument()
	root := document.Entities["scene-root"]
	piece := document.Entities["piece-jade-01"]
	root.Parent = piece.ID
	piece.Children = append(piece.Children, root.ID)
	document.Entities[root.ID] = root
	document.Entities[piece.ID] = piece
	if err := document.Validate(); err == nil {
		t.Fatal("expected invalid root/cycle")
	}
}

func TestJournalRecoversLastCommittedDocument(t *testing.T) {
	dir := t.TempDir()
	workspace, err := OpenWorkspace(dir, SampleDocument())
	if err != nil {
		t.Fatal(err)
	}
	target, _ := FirstPickTarget(SampleDocument())
	_, committed, err := workspace.Execute(Transaction{ID: "persist", Actor: "agent://test", Mode: ModeDirect, ExpectedRevision: 1, Operations: []Operation{{Kind: OpRenameEntity, Target: target, Name: "Recovered Piece"}}})
	if err != nil {
		t.Fatal(err)
	}
	var explicitSave Document
	data, err := os.ReadFile(filepath.Join(dir, "scene.scene3d"))
	if err != nil || json.Unmarshal(data, &explicitSave) != nil {
		t.Fatalf("read explicit save: %v", err)
	}
	if explicitSave.Revision != 1 {
		t.Fatalf("command rewrote explicit save to revision %d", explicitSave.Revision)
	}
	reopened, err := OpenWorkspace(dir, SampleDocument())
	if err != nil {
		t.Fatal(err)
	}
	recovered, _ := reopened.Snapshot()
	if recovered.Entities[target].Name != "Recovered Piece" {
		t.Fatalf("recovered name = %q", recovered.Entities[target].Name)
	}
	want, _ := committed.Fingerprint()
	got, _ := recovered.Fingerprint()
	if got != want {
		t.Fatalf("fingerprint mismatch: %s != %s", got, want)
	}
	status := reopened.ProjectStatus()
	if !status.Recovered || !status.Dirty || status.SavedRevision != 1 || status.Revision != 2 {
		t.Fatalf("recovery status = %+v", status)
	}
	if _, err := reopened.Save(); err != nil {
		t.Fatal(err)
	}
	saved, err := OpenWorkspace(dir, SampleDocument())
	if err != nil {
		t.Fatal(err)
	}
	if status = saved.ProjectStatus(); status.Recovered || status.Dirty || status.SavedRevision != 2 {
		t.Fatalf("saved status = %+v", status)
	}
}

func TestRecoverySkipsTornJournalTailAndQuarantinesCorruptSave(t *testing.T) {
	dir := t.TempDir()
	workspace, err := OpenWorkspace(dir, SampleDocument())
	if err != nil {
		t.Fatal(err)
	}
	target, _ := FirstPickTarget(SampleDocument())
	_, first, err := workspace.Execute(Transaction{ID: "first", Actor: "agent://test", Mode: ModeDirect, ExpectedRevision: 1, Operations: []Operation{{Kind: OpRenameEntity, Target: target, Name: "Last Valid"}}})
	if err != nil {
		t.Fatal(err)
	}
	journalPath := filepath.Join(dir, "commands.jsonl")
	journal, err := os.OpenFile(journalPath, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := journal.WriteString("{torn-json\n"); err != nil {
		t.Fatal(err)
	}
	_ = journal.Close()
	corrupt := []byte("not a scene document")
	if err := os.WriteFile(filepath.Join(dir, "scene.scene3d"), corrupt, 0o600); err != nil {
		t.Fatal(err)
	}

	reopened, err := OpenWorkspace(dir, SampleDocument())
	if err != nil {
		t.Fatal(err)
	}
	recovered, _ := reopened.Snapshot()
	want, _ := first.Fingerprint()
	got, _ := recovered.Fingerprint()
	if got != want || recovered.Entities[target].Name != "Last Valid" {
		t.Fatalf("recovered fingerprint/name = %s/%q want %s", got, recovered.Entities[target].Name, want)
	}
	matches, err := filepath.Glob(filepath.Join(dir, "scene.scene3d.corrupt-*"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("corrupt quarantine = %v, %v", matches, err)
	}
	preserved, err := os.ReadFile(matches[0])
	if err != nil || string(preserved) != string(corrupt) {
		t.Fatalf("quarantined bytes = %q, %v", preserved, err)
	}
}

// gridMeshGeometry builds an n by n quad grid as an authored indexed mesh.
func gridMeshGeometry(n int) Geometry {
	geometry := Geometry{Kind: "indexed-mesh"}
	vertexID := func(row, column int) ID { return ID(fmt.Sprintf("v-%04d-%04d", row, column)) }
	for row := 0; row <= n; row++ {
		for column := 0; column <= n; column++ {
			geometry.Vertices = append(geometry.Vertices, Vertex{
				ID:       vertexID(row, column),
				Position: Vec3{X: float64(column) * 0.01, Z: float64(row) * 0.01},
				Normal:   Vec3{Y: 1},
				UV:       &Vec2{X: float64(column) / float64(n), Y: float64(row) / float64(n)},
			})
		}
	}
	for row := 0; row < n; row++ {
		for column := 0; column < n; column++ {
			geometry.Faces = append(geometry.Faces, Face{
				ID:       ID(fmt.Sprintf("f-%04d-%04d", row, column)),
				Vertices: []ID{vertexID(row, column), vertexID(row, column+1), vertexID(row+1, column+1), vertexID(row+1, column)},
			})
		}
	}
	return geometry
}

// A journal record carries a whole SceneDoc, so a large authored mesh produces
// a record larger than any fixed line buffer. The reader used a 16 MiB
// bufio.Scanner cap: the record was fsynced, the trim that followed failed,
// the command was rejected with its record already on disk, and every later
// open of that project failed the same way. One edit on a scene inside the
// documented 100k-triangle import budget made the project unopenable.
func TestLargeDocumentCommitsAndReopensPastTheOldScannerCap(t *testing.T) {
	if testing.Short() {
		t.Skip("large-record journal evidence skipped in short mode")
	}
	document := SampleDocument()
	root := document.Entities["scene-root"]
	entity := meshEntity("large-mesh", "Large mesh", Vec3{}, gridMeshGeometry(200), "board-material", true)
	entity.Parent = root.ID
	root.Children = append(root.Children, entity.ID)
	document.Entities[entity.ID] = entity
	document.Entities[root.ID] = root
	if err := document.Validate(); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	if err := MigrateAndWriteSeed(filepath.Join(dir, "scene.scene3d"), document); err != nil {
		t.Fatal(err)
	}
	workspace, err := OpenWorkspace(dir, document)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := workspace.Snapshot()
	if err != nil {
		t.Fatal(err)
	}

	transform := TransformFromEuler(Vec3{X: 1}, Vec3{}, Vec3{X: 1, Y: 1, Z: 1})
	receipt, _, err := workspace.Execute(Transaction{
		ID: "large-edit", Actor: "human://inspector", Mode: ModeDirect,
		ExpectedRevision: snapshot.Revision,
		Operations:       []Operation{{Kind: OpSetTransform, Target: "large-mesh", Transform: &transform}},
	})
	if err != nil {
		t.Fatalf("edit on a large document: %v", err)
	}
	if !receipt.Applied || receipt.AfterRevision != snapshot.Revision+1 {
		t.Fatalf("receipt = applied %t, revision %d, want applied at %d", receipt.Applied, receipt.AfterRevision, snapshot.Revision+1)
	}

	info, err := os.Stat(filepath.Join(dir, "commands.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() <= 16<<20 {
		t.Fatalf("journal is %d bytes; this test must exceed the old 16 MiB cap to prove anything", info.Size())
	}
	if warning := workspace.ProjectStatus().CompactionWarning; warning != "" {
		t.Fatalf("compaction warning = %q", warning)
	}

	reopened, err := OpenWorkspace(dir, document)
	if err != nil {
		t.Fatalf("reopen after a large edit: %v", err)
	}
	recovered, err := reopened.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if recovered.Revision != receipt.AfterRevision {
		t.Fatalf("recovered revision = %d, want %d", recovered.Revision, receipt.AfterRevision)
	}
	if got := recovered.Entities["large-mesh"].Transform.Position.X; got != 1 {
		t.Fatalf("recovered position.x = %v, want 1", got)
	}
}

func TestProjectSwitchIsRevisionSafeAndRequiresExplicitDiscard(t *testing.T) {
	firstDir := t.TempDir()
	secondDir := t.TempDir()
	// The workspace canonicalizes project paths (Abs + EvalSymlinks); resolve
	// the expectation the same way so Windows 8.3 short names and macOS
	// /private symlinked temp dirs compare equal.
	if resolved, err := filepath.EvalSymlinks(secondDir); err == nil {
		secondDir = resolved
	}
	workspace, err := OpenWorkspace(firstDir, SampleDocument())
	if err != nil {
		t.Fatal(err)
	}
	second := SampleDocument()
	second.ID = "second-project"
	second.Name = "Second Project"
	if _, err := OpenWorkspace(secondDir, second); err != nil {
		t.Fatal(err)
	}
	target, _ := FirstPickTarget(SampleDocument())
	_, dirty, err := workspace.Execute(Transaction{ID: "dirty", Actor: "human://test", Mode: ModeDirect, ExpectedRevision: 1, Operations: []Operation{{Kind: OpRenameEntity, Target: target, Name: "Unsaved"}}})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(secondDir, "scene.scene3d")
	if _, err := workspace.SwitchProjectFile(path, dirty.Revision, false); !errors.Is(err, ErrUnsavedChanges) {
		t.Fatalf("dirty switch error = %v", err)
	}
	if _, err := workspace.SwitchProjectFile(path, 1, true); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("stale switch error = %v", err)
	}
	status, err := workspace.SwitchProjectFile(path, dirty.Revision, true)
	if err != nil {
		t.Fatal(err)
	}
	opened, _ := workspace.Snapshot()
	if opened.ID != "second-project" || opened.Name != "Second Project" || status.Directory != secondDir || status.Dirty {
		t.Fatalf("opened=%s/%s status=%+v", opened.ID, opened.Name, status)
	}
	if selection := workspace.Selection(); len(selection) != 0 {
		t.Fatalf("selection leaked across projects: %v", selection)
	}
	// Receipts, the play session, and the viewport confirmation describe the
	// project that produced them. Carrying them across a switch attributed
	// the previous project's history to the newly opened one.
	if receipts := workspace.RecentReceipts(10); len(receipts) != 0 {
		t.Fatalf("receipts leaked across projects: %d retained, first %q", len(receipts), receipts[0].TransactionID)
	}
	if confirmation := workspace.ViewportConfirmation(); confirmation != nil {
		t.Fatalf("viewport confirmation leaked across projects: %+v", confirmation)
	}
	if status.UndoDepth != 0 {
		t.Fatalf("undo history leaked across projects: depth %d", status.UndoDepth)
	}
	if _, err := workspace.SwitchProjectFile(filepath.Join(secondDir, "other.scene3d"), opened.Revision, true); err == nil {
		t.Fatal("non-canonical project entry was accepted")
	}
}

func TestExtrudeFacesIsCheckpointSafe(t *testing.T) {
	document := SampleDocument()
	cube := Entity{ID: "editable", Name: "Editable", Parent: "scene-root", Transform: IdentityTransform(), Visible: true, Mesh: &MeshComponent{
		Material: "board-material", Pickable: true,
		Geometry: Geometry{Kind: "indexed-mesh", Vertices: []Vertex{
			{ID: "a", Position: Vec3{}}, {ID: "b", Position: Vec3{X: 1}}, {ID: "c", Position: Vec3{X: 1, Z: 1}}, {ID: "d", Position: Vec3{Z: 1}},
		}, Faces: []Face{{ID: "top", Vertices: []ID{"a", "b", "c", "d"}}}},
	}}
	root := document.Entities["scene-root"]
	root.Children = append(root.Children, cube.ID)
	document.Entities[root.ID] = root
	document.Entities[cube.ID] = cube
	workspace, err := NewWorkspace(document)
	if err != nil {
		t.Fatal(err)
	}
	_, extruded, err := workspace.Execute(Transaction{ID: "extrude", Actor: "agent://test", Mode: ModeDirect, ExpectedRevision: 1, Operations: []Operation{{Kind: OpExtrudeFaces, Target: "editable", Faces: []ID{"top"}, Distance: 0.5}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(extruded.Entities["editable"].Mesh.Geometry.Faces) != 5 {
		t.Fatalf("faces = %d", len(extruded.Entities["editable"].Mesh.Geometry.Faces))
	}
	_, restored, err := workspace.Undo(2, "agent://test")
	if err != nil {
		t.Fatal(err)
	}
	if len(restored.Entities["editable"].Mesh.Geometry.Faces) != 1 {
		t.Fatalf("undo faces = %d", len(restored.Entities["editable"].Mesh.Geometry.Faces))
	}
}
