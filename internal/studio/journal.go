package studio

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const maxJournalRecords = 256

type Journal struct{ dir, documentPath, journalPath string }
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
	return j.bound()
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
	dir, err := os.Open(filepath.Dir(path))
	if err == nil {
		err = dir.Sync()
		_ = dir.Close()
	}
	return quarantine, err
}

func MigrateAndWriteSeed(path string, seed Document) error {
	migrated, err := MigrateDocument(seed)
	if err != nil {
		return err
	}
	return writeDocumentAtomic(path, migrated)
}

func (j *Journal) latestRecord() (Document, bool, error) {
	file, err := os.Open(j.journalPath)
	if errors.Is(err, os.ErrNotExist) {
		return Document{}, false, nil
	}
	if err != nil {
		return Document{}, false, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	var latest Document
	found := false
	for scanner.Scan() {
		var record journalRecord
		if json.Unmarshal(scanner.Bytes(), &record) != nil {
			continue
		}
		sum := sha256.Sum256(record.Payload)
		if hex.EncodeToString(sum[:]) != record.Checksum {
			continue
		}
		var payload journalPayload
		if json.Unmarshal(record.Payload, &payload) != nil {
			continue
		}
		migrated, err := MigrateDocument(payload.Document)
		if err != nil {
			continue
		}
		latest = migrated
		found = true
	}
	if err := scanner.Err(); err != nil {
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
	dir, err := os.Open(filepath.Dir(path))
	if err == nil {
		err = dir.Sync()
		_ = dir.Close()
	}
	return err
}

func (j *Journal) bound() error {
	file, err := os.Open(j.journalPath)
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	lines := make([][]byte, 0, maxJournalRecords+1)
	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)
		lines = append(lines, line)
	}
	scanErr := scanner.Err()
	_ = file.Close()
	if scanErr != nil {
		return scanErr
	}
	if len(lines) <= maxJournalRecords {
		return nil
	}
	temp, err := os.CreateTemp(j.dir, ".journal-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	writer := bufio.NewWriter(temp)
	for _, line := range lines[len(lines)-maxJournalRecords:] {
		if _, err = writer.Write(append(line, '\n')); err != nil {
			break
		}
	}
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
	return os.Rename(temp.Name(), j.journalPath)
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
