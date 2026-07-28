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

const maxJournalRecords = 256

type Journal struct {
	dir, documentPath, journalPath string
	// compactionWarning records a failed tail trim. Compaction is an
	// optimization; the journal stays correct when it does not run, so a
	// compaction failure must never reject a command whose record is
	// already durable. The warning surfaces in ProjectStatus instead.
	compactionWarning string
}

// CompactionWarning reports the last failed journal trim, or the empty string.
func (j *Journal) CompactionWarning() string { return j.compactionWarning }

// eachJournalLine calls visit for every newline-terminated record in the
// journal, oldest first. It uses a growing reader rather than bufio.Scanner:
// a Scanner caps one token, and one record holds a whole SceneDoc, so a large
// authored scene produced a record the reader could never scan again. That
// turned an ordinary edit into a project that would not reopen.
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
	Document    Document    `json:"document"`
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

func (j *Journal) Commit(transaction Transaction, receipt Receipt, document Document) error {
	payload, err := json.Marshal(journalPayload{Transaction: transaction, Receipt: receipt, Document: document})
	if err != nil {
		return err
	}
	sum := sha256.Sum256(payload)
	record := journalRecord{Payload: payload, Checksum: hex.EncodeToString(sum[:])}
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
	// The record is durable from here. Trimming the tail is an optimization,
	// so a failure is reported through ProjectStatus rather than returned:
	// returning it rejected a command that the journal had already accepted,
	// which lost the edit while keeping its record on disk.
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
	latest, found, err := j.latestRecord()
	if err != nil {
		return Document{}, 0, false, err
	}
	if found && (!savedOK || latest.Revision > saved.Revision) {
		savedRevision := uint64(0)
		if savedOK {
			savedRevision = saved.Revision
		}
		return latest, savedRevision, true, nil
	}
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

func (j *Journal) latestRecord() (Document, bool, error) {
	var latest Document
	found := false
	err := eachJournalLine(j.journalPath, func(line []byte) error {
		var record journalRecord
		if json.Unmarshal(line, &record) != nil {
			return nil
		}
		sum := sha256.Sum256(record.Payload)
		if hex.EncodeToString(sum[:]) != record.Checksum {
			return nil
		}
		var payload journalPayload
		if json.Unmarshal(record.Payload, &payload) != nil {
			return nil
		}
		migrated, migrateErr := MigrateDocument(payload.Document)
		if migrateErr != nil {
			return nil
		}
		latest = migrated
		found = true
		return nil
	})
	if err != nil {
		return Document{}, false, err
	}
	return latest, found, nil
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

// bound trims the journal to its newest maxJournalRecords records. It reads
// twice and holds one record at a time: one record carries a whole SceneDoc,
// so buffering every line put the entire retained history in memory at once.
func (j *Journal) bound() error {
	total := 0
	if err := eachJournalLine(j.journalPath, func([]byte) error { total++; return nil }); err != nil {
		return err
	}
	if total <= maxJournalRecords {
		return nil
	}
	skip := total - maxJournalRecords

	temp, err := os.CreateTemp(j.dir, ".journal-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	writer := bufio.NewWriter(temp)
	index := 0
	err = eachJournalLine(j.journalPath, func(line []byte) error {
		index++
		if index <= skip {
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
