package studio

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

// journalSnapshotInterval bounds how many operations-only records recovery
// ever replays after a snapshot. A crash-time reopen replays at most
// journalSnapshotInterval-1 small edits, and the journal writes a whole
// SceneDoc on only one record in every journalSnapshotInterval. 32 keeps
// both costs small: a bounded replay, and rare full-document writes.
const journalSnapshotInterval = 32

// maxJournalRecords bounds retained journal history. bound trims toward this
// count, but never past the newest snapshot at or before the cut: a journal
// that starts with an operations-only record has no document to replay it
// onto. The kept file can therefore hold up to journalSnapshotInterval - 1
// records more than this bound.
const maxJournalRecords = 256

type Journal struct {
	dir, documentPath, journalPath string
	// compactionWarning records a failed tail trim. Compaction is an
	// optimization; the journal stays correct when it does not run, so a
	// compaction failure must never reject a command whose record is
	// already durable. The warning surfaces in ProjectStatus instead.
	compactionWarning string
	// recordsSinceSnapshot counts operations-only records committed after
	// the journal's last snapshot. recover sets it once at open; Commit
	// keeps it current afterward. It decides when the next record must
	// carry a full Document instead of just a diff.
	recordsSinceSnapshot int
}

// CompactionWarning reports the last failed journal trim, or the empty string.
func (j *Journal) CompactionWarning() string { return j.compactionWarning }

// eachJournalLine calls visit for every newline-terminated record in the
// journal, oldest first. It uses a growing reader rather than bufio.Scanner:
// a Scanner caps one token, and a snapshot record holds a whole SceneDoc, so
// a large authored scene produced a record the reader could never scan
// again. That turned an ordinary edit into a project that would not reopen.
func eachJournalLine(path string, visit func(line []byte) error) error {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()
	reader := bufio.NewReaderSize(file, 64*1024)
	for {
		line, readErr := reader.ReadBytes('\n')
		line = bytes.TrimRight(line, "\n")
		if len(line) > 0 {
			if err := visit(line); err != nil {
				return err
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return nil
			}
			return readErr
		}
	}
}

type journalPayload struct {
	Transaction Transaction `json:"transaction"`
	Receipt     Receipt     `json:"receipt"`
	// Document is present only on a snapshot record. Its absence means the
	// record's Transaction is the whole change: recovery must replay
	// Transaction.Operations onto the document the record before it in the
	// journal produced. A record written before this format always carries
	// a Document, so it is always its own snapshot — that is what keeps an
	// old journal recoverable unchanged.
	Document *Document `json:"document,omitempty"`
}
type journalRecord struct {
	Payload  json.RawMessage `json:"payload"`
	Checksum string          `json:"checksum"`
}

func OpenWorkspace(dir string, seed Document) (*Workspace, error) {
	journal := &Journal{dir: dir, documentPath: filepath.Join(dir, "scene.scene3d"), journalPath: filepath.Join(dir, "commands.jsonl")}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	document, savedRevision, recovered, err := journal.recover(seed)
	if err != nil {
		return nil, err
	}
	workspace, err := newWorkspace(document, journal)
	if err != nil {
		return nil, err
	}
	workspace.savedRevision = savedRevision
	workspace.recovered = recovered
	workspace.projectDir = dir
	return workspace, nil
}

// Commit appends one record for transaction and receipt. document is written
// in full only when forceSnapshot asks for it or the journal's own interval
// is due; otherwise the record carries transaction.Operations alone, and
// recovery replays them onto the document the previous record produced.
//
// forceSnapshot exists because not every commit's Operations are a valid
// forward diff from the record before them — see the undo/redo call site in
// moveHistory. A caller whose Operations are such a diff may pass false and
// let the interval decide.
func (j *Journal) Commit(transaction Transaction, receipt Receipt, document Document, forceSnapshot bool) error {
	snapshot := forceSnapshot || j.recordsSinceSnapshot >= journalSnapshotInterval-1
	payload := journalPayload{Transaction: transaction, Receipt: receipt}
	if snapshot {
		payload.Document = &document
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	record := journalRecord{Payload: data, Checksum: hex.EncodeToString(sum[:])}
	line, err := json.Marshal(record)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(j.journalPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("open command journal: %w", err)
	}
	if _, err = file.Write(append(line, '\n')); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		return fmt.Errorf("append command journal: %w", err)
	}
	if closeErr != nil {
		return closeErr
	}
	// The record is durable from here, so the cadence it represents must be
	// too: a later crash has to find the same snapshot boundary this call
	// chose, or replay will look for an anchor that was never written.
	if snapshot {
		j.recordsSinceSnapshot = 0
	} else {
		j.recordsSinceSnapshot++
	}
	// Trimming the tail is an optimization, so a failure is reported through
	// ProjectStatus rather than returned: returning it rejected a command
	// that the journal had already accepted, which lost the edit while
	// keeping its record on disk.
	if boundErr := j.bound(); boundErr != nil {
		j.compactionWarning = boundErr.Error()
	} else {
		j.compactionWarning = ""
	}
	return nil
}

func (j *Journal) Save(document Document) error {
	return writeDocumentAtomic(j.documentPath, document)
}

func (j *Journal) recover(seed Document) (Document, uint64, bool, error) {
	var saved Document
	var savedOK bool
	data, readErr := os.ReadFile(j.documentPath)
	if readErr == nil {
		decodeErr := json.Unmarshal(data, &saved)
		if decodeErr == nil {
			if migrated, migrateErr := MigrateDocument(saved); migrateErr == nil {
				saved = migrated
				savedOK = true
			} else {
				decodeErr = migrateErr
			}
		}
		if decodeErr != nil {
			if _, err := quarantineCorruptDocument(j.documentPath, data); err != nil {
				return Document{}, 0, false, fmt.Errorf("preserve corrupt SceneDoc: %w", err)
			}
		}
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return Document{}, 0, false, readErr
	}
	latest, found, sinceSnapshot, err := j.replayJournal()
	if err != nil {
		return Document{}, 0, false, err
	}
	if found && (!savedOK || latest.Revision > saved.Revision) {
		savedRevision := uint64(0)
		if savedOK {
			savedRevision = saved.Revision
		}
		// The workspace starts exactly where replay left off, so the next
		// commit may keep counting from here.
		j.recordsSinceSnapshot = sinceSnapshot
		return latest, savedRevision, true, nil
	}
	// Every other branch starts the workspace from a document replay did not
	// just produce — the explicit save, or a fresh seed. The journal's next
	// record cannot assume it continues that replay chain, so it must anchor
	// a new one.
	j.recordsSinceSnapshot = journalSnapshotInterval - 1
	if savedOK {
		return saved, saved.Revision, false, nil
	}
	if err := MigrateAndWriteSeed(j.documentPath, seed); err != nil {
		return Document{}, 0, false, err
	}
	return seed, seed.Revision, false, nil
}

func quarantineCorruptDocument(path string, data []byte) (string, error) {
	sum := sha256.Sum256(data)
	quarantine := path + ".corrupt-" + hex.EncodeToString(sum[:6])
	if _, err := os.Stat(quarantine); err == nil {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		return quarantine, nil
	}
	if err := os.Rename(path, quarantine); err != nil {
		return "", err
	}
	return quarantine, syncDir(filepath.Dir(path))
}

func MigrateAndWriteSeed(path string, seed Document) error {
	migrated, err := MigrateDocument(seed)
	if err != nil {
		return err
	}
	return writeDocumentAtomic(path, migrated)
}

// parseJournalRecord validates one record's checksum and decodes its JSON
// payload, without migrating an embedded Document. It only answers whether
// the bytes are durable and well-formed; a caller that reads the document's
// content still has to migrate it.
func parseJournalRecord(line []byte) (journalPayload, bool) {
	var record journalRecord
	if json.Unmarshal(line, &record) != nil {
		return journalPayload{}, false
	}
	sum := sha256.Sum256(record.Payload)
	if hex.EncodeToString(sum[:]) != record.Checksum {
		return journalPayload{}, false
	}
	var payload journalPayload
	if json.Unmarshal(record.Payload, &payload) != nil {
		return journalPayload{}, false
	}
	return payload, true
}

// replayJournal reconstructs the newest document the journal describes, and
// counts how many operations-only records were replayed after the snapshot
// it anchored to. The count seeds Journal.recordsSinceSnapshot, so a
// reopened project keeps the snapshot cadence a continuously running one
// would have had, instead of writing one on the very next commit regardless.
//
// A record without a Document holds only a diff against the document the
// record before it in the journal produced, so replay needs an anchor
// before it can apply one. An old-format journal has a Document on every
// record, so every record is its own anchor; this degrades to "use the
// newest valid document," which is the recovery behavior the format
// replaces, and is why an old journal opens unchanged.
//
// A record that cannot be replayed breaks the chain rather than being
// skipped. Skipping one and continuing would apply later diffs to a document
// missing an edit, and recover a state the project never actually held —
// worse than losing the tail, because it looks correct. Replay therefore
// stops applying diffs until the next valid snapshot re-anchors it, which
// bounds the loss to one snapshot interval.
func (j *Journal) replayJournal() (document Document, found bool, recordsSinceSnapshot int, err error) {
	chainBroken := false
	walkErr := eachJournalLine(j.journalPath, func(line []byte) error {
		payload, ok := parseJournalRecord(line)
		if !ok {
			// Torn or corrupt. A tail record is simply absent; a record with
			// valid records after it leaves a hole no later diff may cross.
			chainBroken = true
			return nil
		}
		if payload.Document != nil {
			migrated, migrateErr := MigrateDocument(*payload.Document)
			if migrateErr != nil {
				chainBroken = true
				return nil
			}
			// A snapshot stands on its own, so it re-anchors a broken chain.
			document = migrated
			found = true
			recordsSinceSnapshot = 0
			chainBroken = false
			return nil
		}
		if !found || chainBroken {
			// Either nothing to replay onto, or an earlier record in this
			// chain did not survive. Wait for a snapshot.
			chainBroken = true
			return nil
		}
		next, cloneErr := document.Clone()
		if cloneErr != nil {
			chainBroken = true
			return nil
		}
		for _, operation := range payload.Transaction.Operations {
			if _, applyErr := applyOperation(&next, operation); applyErr != nil {
				chainBroken = true
				return nil
			}
		}
		// Execute increments the revision once, after every operation in the
		// transaction has applied. Replay has to do the same in the same
		// place, or a reopened project reports a revision the live one never
		// held.
		next.Revision++
		if next.Validate() != nil {
			chainBroken = true
			return nil
		}
		document = next
		recordsSinceSnapshot++
		return nil
	})
	if walkErr != nil {
		return Document{}, false, 0, walkErr
	}
	return document, found, recordsSinceSnapshot, nil
}

func writeDocumentAtomic(path string, document Document) error {
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".scene-*.tmp")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err = temp.Chmod(0o600); err == nil {
		_, err = temp.Write(data)
	}
	if err == nil {
		err = temp.Sync()
	}
	closeErr := temp.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err = os.Rename(tempPath, path); err != nil {
		return err
	}
	return syncDir(filepath.Dir(path))
}

// bound trims the journal toward maxJournalRecords, but never past the
// newest snapshot at or before the cut: an operations-only record has
// nothing to replay onto without one. It streams in three single-record
// passes rather than buffering anything, because one record can still carry
// a whole SceneDoc — holding more than one at a time defeats the point of
// writing most records without one.
func (j *Journal) bound() error {
	total := 0
	if err := eachJournalLine(j.journalPath, func([]byte) error { total++; return nil }); err != nil {
		return err
	}
	if total <= maxJournalRecords {
		return nil
	}
	skip := total - maxJournalRecords

	// anchor is the highest-index snapshot at or before skip. Cutting there
	// instead of exactly at skip can keep a few more records than
	// maxJournalRecords; Commit guarantees a snapshot at least every
	// journalSnapshotInterval records, so the excess stays bounded even
	// though this search is not.
	anchor := 0
	index := 0
	if err := eachJournalLine(j.journalPath, func(line []byte) error {
		if index <= skip {
			if payload, ok := parseJournalRecord(line); ok && payload.Document != nil {
				anchor = index
			}
		}
		index++
		return nil
	}); err != nil {
		return err
	}
	skip = anchor

	temp, err := os.CreateTemp(j.dir, ".journal-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	writer := bufio.NewWriter(temp)
	index = 0
	err = eachJournalLine(j.journalPath, func(line []byte) error {
		keep := index >= skip
		index++
		if !keep {
			return nil
		}
		if _, writeErr := writer.Write(line); writeErr != nil {
			return writeErr
		}
		_, writeErr := writer.Write([]byte{'\n'})
		return writeErr
	})
	if err == nil {
		err = writer.Flush()
	}
	if err == nil {
		err = temp.Sync()
	}
	closeErr := temp.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err := os.Rename(temp.Name(), j.journalPath); err != nil {
		return err
	}
	return syncDir(j.dir)
}

func DecodeTransaction(reader io.Reader) (Transaction, error) {
	var transaction Transaction
	decoder := json.NewDecoder(io.LimitReader(reader, 4<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&transaction); err != nil {
		return Transaction{}, err
	}
	return transaction, nil
}

// syncDir makes a rename durable by fsyncing the containing directory on
// platforms that support it. Windows cannot fsync a directory handle
// ("Access is denied"); NTFS journals rename metadata, so the sync is
// skipped there — matching standard Go durability practice.
func syncDir(path string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	err = dir.Sync()
	if closeErr := dir.Close(); err == nil {
		err = closeErr
	}
	return err
}
