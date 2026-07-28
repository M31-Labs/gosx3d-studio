package studio

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// countJournalRecords reports how many records a journal holds and how many
// of those are snapshots (carry a Document). It reuses parseJournalRecord
// rather than a second decoder, so a test failure here means the production
// reader disagrees with itself, not that the test drifted from it.
func countJournalRecords(t *testing.T, path string) (total, snapshots int) {
	t.Helper()
	if err := eachJournalLine(path, func(line []byte) error {
		payload, ok := parseJournalRecord(line)
		if !ok {
			t.Fatalf("unparseable record in %s: %q", path, line)
		}
		total++
		if payload.Document != nil {
			snapshots++
		}
		return nil
	}); err != nil {
		t.Fatalf("read journal: %v", err)
	}
	return total, snapshots
}

// appendLegacyJournalRecord writes one record the way the journal wrote
// every record before this change: a Document on it regardless of position.
// It builds the same journalRecord/journalPayload wire shape production code
// uses, so this test proves compatibility with the format, not with a copy
// of it that could quietly drift.
func appendLegacyJournalRecord(t *testing.T, path string, transaction Transaction, receipt Receipt, document Document) {
	t.Helper()
	payload := journalPayload{Transaction: transaction, Receipt: receipt, Document: &document}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	record := journalRecord{Payload: data, Checksum: hex.EncodeToString(sum[:])}
	line, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err := file.Write(append(line, '\n')); err != nil {
		t.Fatal(err)
	}
}

// TestJournalReplaysOperationsAcrossMultipleSnapshotIntervals is the replay
// equivalence proof: a run long enough to cross journalSnapshotInterval more
// than once must reopen to the exact document the live workspace holds,
// which only happens if every operations-only record replayed correctly and
// in order onto the snapshot before it.
func TestJournalReplaysOperationsAcrossMultipleSnapshotIntervals(t *testing.T) {
	dir := t.TempDir()
	workspace, err := OpenWorkspace(dir, SampleDocument())
	if err != nil {
		t.Fatal(err)
	}
	target, _ := FirstPickTarget(SampleDocument())

	const edits = journalSnapshotInterval*2 + 6
	expected := uint64(1)
	for i := 0; i < edits; i++ {
		transaction := Transaction{
			ID: fmt.Sprintf("edit-%03d", i), Actor: "agent://test", Mode: ModeDirect,
			ExpectedRevision: expected,
			Operations:       []Operation{{Kind: OpRenameEntity, Target: target, Name: fmt.Sprintf("Renamed %03d", i)}},
		}
		_, committed, err := workspace.Execute(transaction)
		if err != nil {
			t.Fatalf("edit %d: %v", i, err)
		}
		expected = committed.Revision
	}

	live, err := workspace.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	liveFingerprint, err := live.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}

	journalPath := filepath.Join(dir, "commands.jsonl")
	total, snapshots := countJournalRecords(t, journalPath)
	if total != edits {
		t.Fatalf("records = %d, want %d (one per edit)", total, edits)
	}
	if snapshots < 2 || snapshots >= total {
		t.Fatalf("snapshots = %d of %d records; this run should cross the snapshot interval more than once and still write mostly operations-only records", snapshots, total)
	}

	reopened, err := OpenWorkspace(dir, SampleDocument())
	if err != nil {
		t.Fatal(err)
	}
	recovered, err := reopened.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	recoveredFingerprint, err := recovered.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	if recoveredFingerprint != liveFingerprint {
		t.Fatalf("recovered fingerprint = %s, want %s", recoveredFingerprint, liveFingerprint)
	}
	if recovered.Revision != live.Revision {
		t.Fatalf("recovered revision = %d, want %d", recovered.Revision, live.Revision)
	}
	if recovered.Entities[target].Name != live.Entities[target].Name {
		t.Fatalf("recovered name = %q, want %q", recovered.Entities[target].Name, live.Entities[target].Name)
	}
}

// TestOldFormatJournalRecovers hand-writes a journal in the format every
// record used before this change — a Document on each one — and proves it
// still opens correctly. Each record's Transaction names a change the
// record's Document does not reflect, so a recovery that replayed
// Transaction.Operations instead of trusting Document would produce a
// different, wrong result and this test would catch it.
func TestOldFormatJournalRecovers(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	journalPath := filepath.Join(dir, "commands.jsonl")
	target, _ := FirstPickTarget(SampleDocument())

	first := SampleDocument()
	first.Revision = 2
	entity := first.Entities[target]
	entity.Name = "Legacy One"
	first.Entities[target] = entity
	appendLegacyJournalRecord(t, journalPath,
		Transaction{ID: "legacy-1", Actor: "agent://legacy", Mode: ModeDirect, ExpectedRevision: 1, Operations: []Operation{{Kind: OpRenameEntity, Target: target, Name: "decoy, must be ignored"}}},
		Receipt{TransactionID: "legacy-1", Applied: true, BeforeRevision: 1, AfterRevision: 2},
		first)

	second := SampleDocument()
	second.Revision = 3
	entity = second.Entities[target]
	entity.Name = "Legacy Two"
	second.Entities[target] = entity
	appendLegacyJournalRecord(t, journalPath,
		Transaction{ID: "legacy-2", Actor: "agent://legacy", Mode: ModeDirect, ExpectedRevision: 2, Operations: []Operation{{Kind: OpRenameEntity, Target: target, Name: "decoy, must be ignored"}}},
		Receipt{TransactionID: "legacy-2", Applied: true, BeforeRevision: 2, AfterRevision: 3},
		second)

	workspace, err := OpenWorkspace(dir, SampleDocument())
	if err != nil {
		t.Fatalf("open workspace over old-format journal: %v", err)
	}
	recovered, err := workspace.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if recovered.Revision != 3 {
		t.Fatalf("recovered revision = %d, want 3", recovered.Revision)
	}
	if name := recovered.Entities[target].Name; name != "Legacy Two" {
		t.Fatalf("recovered name = %q, want %q (a record with a Document must be treated as a snapshot, not replayed)", name, "Legacy Two")
	}
	status := workspace.ProjectStatus()
	if !status.Recovered {
		t.Fatalf("status = %+v, want Recovered", status)
	}

	// The journal must stay usable going forward: a new commit against the
	// recovered revision should succeed and, since every prior record
	// carried its own Document, land as an operations-only record.
	_, committed, err := workspace.Execute(Transaction{ID: "after-legacy", Actor: "agent://test", Mode: ModeDirect, ExpectedRevision: 3, Operations: []Operation{{Kind: OpRenameEntity, Target: target, Name: "Post-Legacy"}}})
	if err != nil {
		t.Fatalf("commit after recovering an old-format journal: %v", err)
	}
	if committed.Revision != 4 {
		t.Fatalf("committed revision = %d, want 4", committed.Revision)
	}
	total, snapshots := countJournalRecords(t, journalPath)
	if total != 3 || snapshots != 2 {
		t.Fatalf("records = %d (snapshots %d), want 3 records (2 legacy snapshots, 1 new operations-only record)", total, snapshots)
	}
}

// TestJournalTornTailAfterOperationsRecordsSkipsAndRecoversPriorState proves
// that a crash mid-append past a run of operations-only records loses only
// the torn record. The prior state has to come back by replaying a snapshot
// plus several operations-only records, not by reading one record in
// isolation.
func TestJournalTornTailAfterOperationsRecordsSkipsAndRecoversPriorState(t *testing.T) {
	dir := t.TempDir()
	workspace, err := OpenWorkspace(dir, SampleDocument())
	if err != nil {
		t.Fatal(err)
	}
	target, _ := FirstPickTarget(SampleDocument())

	var priorState Document
	expected := uint64(1)
	const edits = 5
	for i := 0; i < edits; i++ {
		_, committed, err := workspace.Execute(Transaction{
			ID: fmt.Sprintf("edit-%d", i), Actor: "agent://test", Mode: ModeDirect,
			ExpectedRevision: expected,
			Operations:       []Operation{{Kind: OpRenameEntity, Target: target, Name: fmt.Sprintf("State %d", i)}},
		})
		if err != nil {
			t.Fatalf("edit %d: %v", i, err)
		}
		expected = committed.Revision
		if i == edits-2 {
			priorState = committed
		}
	}

	journalPath := filepath.Join(dir, "commands.jsonl")
	totalBefore, snapshotsBefore := countJournalRecords(t, journalPath)
	if totalBefore != edits || snapshotsBefore != 1 {
		t.Fatalf("setup records = %d (snapshots %d), want %d records with exactly 1 snapshot", totalBefore, snapshotsBefore, edits)
	}

	data, err := os.ReadFile(journalPath)
	if err != nil {
		t.Fatal(err)
	}
	lines := bytes.Split(bytes.TrimRight(data, "\n"), []byte("\n"))
	if len(lines) != edits {
		t.Fatalf("raw journal lines = %d, want %d", len(lines), edits)
	}
	last := lines[len(lines)-1]
	torn := append(bytes.Join(lines[:len(lines)-1], []byte("\n")), '\n')
	torn = append(torn, last[:len(last)/2]...) // No trailing newline: an in-progress write, not a complete one.
	if err := os.WriteFile(journalPath, torn, 0o600); err != nil {
		t.Fatal(err)
	}

	reopened, err := OpenWorkspace(dir, SampleDocument())
	if err != nil {
		t.Fatalf("reopen past a torn tail: %v", err)
	}
	recovered, err := reopened.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	want, err := priorState.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	got, err := recovered.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("recovered fingerprint = %s, want %s (state before the torn record)", got, want)
	}
	if recovered.Revision != priorState.Revision {
		t.Fatalf("recovered revision = %d, want %d", recovered.Revision, priorState.Revision)
	}
	if recovered.Entities[target].Name != priorState.Entities[target].Name {
		t.Fatalf("recovered name = %q, want %q", recovered.Entities[target].Name, priorState.Entities[target].Name)
	}
	status := reopened.ProjectStatus()
	if !status.Recovered {
		t.Fatalf("status = %+v, want Recovered", status)
	}
}

// TestJournalWriteAmplificationDropsWellBelowWholeDocumentPerRecord is the
// evidence for the change this file exists to make: journaling operations
// with periodic snapshots instead of a whole SceneDoc on every record. It
// commits enough edits to cross the snapshot interval twice and measures the
// journal against what the old whole-document-per-record format would have
// cost for the same run.
//
// The edited entity is a small piece, not the large mesh sharing the
// document: a Receipt carries the full before/after of whatever entity an
// operation touches (semantic-diff evidence, unrelated to this change), so
// editing the large entity would size every record by it regardless of
// format and hide the exact saving this test exists to measure.
func TestJournalWriteAmplificationDropsWellBelowWholeDocumentPerRecord(t *testing.T) {
	if testing.Short() {
		t.Skip("mesh-document journal sizing skipped in short mode")
	}
	document := SampleDocument()
	root := document.Entities["scene-root"]
	mesh := meshEntity("amplification-mesh", "Amplification mesh", Vec3{}, gridMeshGeometry(50), "board-material", true)
	mesh.Parent = root.ID
	root.Children = append(root.Children, mesh.ID)
	document.Entities[mesh.ID] = mesh
	document.Entities[root.ID] = root
	if err := document.Validate(); err != nil {
		t.Fatal(err)
	}
	documentJSON, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	target, _ := FirstPickTarget(document)

	dir := t.TempDir()
	workspace, err := OpenWorkspace(dir, document)
	if err != nil {
		t.Fatal(err)
	}

	const edits = journalSnapshotInterval * 2
	expected := uint64(1)
	for i := 0; i < edits; i++ {
		_, committed, err := workspace.Execute(Transaction{
			ID: fmt.Sprintf("mesh-edit-%03d", i), Actor: "agent://test", Mode: ModeDirect,
			ExpectedRevision: expected,
			Operations:       []Operation{{Kind: OpRenameEntity, Target: target, Name: fmt.Sprintf("Amp %03d", i)}},
		})
		if err != nil {
			t.Fatalf("edit %d: %v", i, err)
		}
		expected = committed.Revision
	}
	if warning := workspace.ProjectStatus().CompactionWarning; warning != "" {
		t.Fatalf("compaction warning = %q", warning)
	}

	journalPath := filepath.Join(dir, "commands.jsonl")
	info, err := os.Stat(journalPath)
	if err != nil {
		t.Fatal(err)
	}
	total, snapshots := countJournalRecords(t, journalPath)
	if total != edits {
		t.Fatalf("records = %d, want %d", total, edits)
	}
	oldFormatCost := int64(edits) * int64(len(documentJSON))
	// The theoretical new cost is about 2 snapshots plus edits-2 small
	// records — roughly 1/32 of oldFormatCost. Assert an eighth as a margin
	// against per-record JSON overhead without letting a regression hide.
	if info.Size() >= oldFormatCost/8 {
		t.Fatalf("journal size = %d bytes (%d records, %d snapshots); old whole-document-per-record cost for %d edits would be %d bytes; want well below oldFormatCost/8 = %d", info.Size(), total, snapshots, edits, oldFormatCost, oldFormatCost/8)
	}
	if snapshots < 2 || snapshots > edits/journalSnapshotInterval+1 {
		t.Fatalf("snapshots = %d across %d edits spanning %d snapshot intervals", snapshots, edits, edits/journalSnapshotInterval)
	}
	t.Logf("journal = %d bytes for %d edits (%d snapshots); old format would have cost %d bytes (%.1fx)", info.Size(), edits, snapshots, oldFormatCost, float64(oldFormatCost)/float64(info.Size()))

	reopened, err := OpenWorkspace(dir, document)
	if err != nil {
		t.Fatalf("reopen after mesh edits: %v", err)
	}
	recovered, err := reopened.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if recovered.Revision != expected {
		t.Fatalf("recovered revision = %d, want %d", recovered.Revision, expected)
	}
	if name := recovered.Entities[target].Name; name != fmt.Sprintf("Amp %03d", edits-1) {
		t.Fatalf("recovered name = %q, want %q", name, fmt.Sprintf("Amp %03d", edits-1))
	}
	if got, want := len(recovered.Entities["amplification-mesh"].Mesh.Geometry.Vertices), len(document.Entities["amplification-mesh"].Mesh.Geometry.Vertices); got != want {
		t.Fatalf("recovered mesh vertices = %d, want %d (the large entity must survive snapshot and replay unchanged)", got, want)
	}
}

// A record that cannot be replayed breaks the chain: replay must not skip it
// and keep applying later diffs, because that reconstructs a document the
// project never held — every edit except one. Losing the tail is honest;
// fabricating a state is not. A later snapshot re-anchors replay, so the loss
// is bounded by the snapshot interval.
func TestCorruptMiddleRecordDoesNotFabricateAState(t *testing.T) {
	dir := t.TempDir()
	workspace, err := OpenWorkspace(dir, SampleDocument())
	if err != nil {
		t.Fatal(err)
	}
	document, err := workspace.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	target, _ := FirstPickTarget(document)
	revision := document.Revision

	// Six edits: the first commit anchors a snapshot, the rest are diffs.
	const edits = 6
	states := make([]string, 0, edits+1)
	for i := 0; i < edits; i++ {
		transform := TransformFromEuler(Vec3{X: float64(i + 1)}, Vec3{}, Vec3{X: 1, Y: 1, Z: 1})
		receipt, committed, err := workspace.Execute(Transaction{
			ID: fmt.Sprintf("hole-%d", i), Actor: "human://local-ui", Mode: ModeDirect,
			ExpectedRevision: revision,
			Operations:       []Operation{{Kind: OpSetTransform, Target: target, Transform: &transform}},
		})
		if err != nil {
			t.Fatalf("edit %d: %v", i, err)
		}
		revision = receipt.AfterRevision
		fingerprint, err := committed.Fingerprint()
		if err != nil {
			t.Fatal(err)
		}
		states = append(states, fingerprint)
	}

	// Corrupt a record in the middle, leaving valid records after it.
	journalPath := filepath.Join(dir, "commands.jsonl")
	raw, err := os.ReadFile(journalPath)
	if err != nil {
		t.Fatal(err)
	}
	lines := bytes.Split(bytes.TrimRight(raw, "\n"), []byte("\n"))
	if len(lines) < 5 {
		t.Fatalf("expected several journal records, got %d", len(lines))
	}
	const holeIndex = 2
	lines[holeIndex] = []byte(`{"payload":{"transaction":{}},"checksum":"deadbeef"}`)
	if err := os.WriteFile(journalPath, append(bytes.Join(lines, []byte("\n")), '\n'), 0o600); err != nil {
		t.Fatal(err)
	}

	reopened, err := OpenWorkspace(dir, SampleDocument())
	if err != nil {
		t.Fatal(err)
	}
	recovered, err := reopened.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	recoveredFingerprint, err := recovered.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}

	// The recovered document must be one the project actually held: either a
	// state from before the hole, or the seed. It must never be a later state,
	// which would mean later diffs were applied across the gap.
	for index, fingerprint := range states {
		if fingerprint == recoveredFingerprint {
			if index >= holeIndex {
				t.Fatalf("recovered state %d, which is at or after the corrupt record %d: later diffs were applied across the hole", index, holeIndex)
			}
			return
		}
	}
	seed, err := SampleDocument().Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	if recoveredFingerprint != seed {
		t.Fatalf("recovered a document that never existed: %s", recoveredFingerprint)
	}
}
