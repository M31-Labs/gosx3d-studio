package studio

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

var ErrRevisionConflict = errors.New("studio revision conflict")

type TransactionMode string

const (
	ModePropose TransactionMode = "propose"
	ModeDirect  TransactionMode = "direct"
)

type OperationKind string

const (
	OpSetTransform   OperationKind = "set-transform"
	OpAssignMaterial OperationKind = "assign-material"
	OpRenameEntity   OperationKind = "rename-entity"
)

type Transaction struct {
	ID               string          `json:"id"`
	Actor            string          `json:"actor"`
	Mode             TransactionMode `json:"mode"`
	ExpectedRevision uint64          `json:"expectedRevision"`
	Operations       []Operation     `json:"operations"`
}

type Operation struct {
	Kind      OperationKind `json:"kind"`
	Target    ID            `json:"target"`
	Transform *Transform    `json:"transform,omitempty"`
	Material  ID            `json:"material,omitempty"`
	Name      string        `json:"name,omitempty"`
}

type Receipt struct {
	TransactionID     string          `json:"transactionId"`
	Actor             string          `json:"actor"`
	Mode              TransactionMode `json:"mode"`
	Applied           bool            `json:"applied"`
	BeforeRevision    uint64          `json:"beforeRevision"`
	AfterRevision     uint64          `json:"afterRevision"`
	BeforeFingerprint string          `json:"beforeFingerprint"`
	AfterFingerprint  string          `json:"afterFingerprint"`
	Operations        int             `json:"operations"`
}

type Workspace struct {
	mu      sync.RWMutex
	doc     Document
	history []Document
}

func NewWorkspace(document Document) (*Workspace, error) {
	if err := document.Validate(); err != nil {
		return nil, err
	}
	clone, err := document.Clone()
	if err != nil {
		return nil, err
	}
	return &Workspace{doc: clone}, nil
}

func (w *Workspace) Snapshot() (Document, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.doc.Clone()
}

func (w *Workspace) Execute(transaction Transaction) (Receipt, Document, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if transaction.ExpectedRevision != w.doc.Revision {
		return Receipt{}, Document{}, fmt.Errorf("%w: have %d, expected %d", ErrRevisionConflict, w.doc.Revision, transaction.ExpectedRevision)
	}
	if transaction.Mode != ModePropose && transaction.Mode != ModeDirect {
		return Receipt{}, Document{}, fmt.Errorf("unsupported transaction mode %q", transaction.Mode)
	}
	if strings.TrimSpace(transaction.ID) == "" || strings.TrimSpace(transaction.Actor) == "" {
		return Receipt{}, Document{}, fmt.Errorf("transaction id and actor are required")
	}
	if len(transaction.Operations) == 0 {
		return Receipt{}, Document{}, fmt.Errorf("transaction requires at least one operation")
	}
	before, err := w.doc.Clone()
	if err != nil {
		return Receipt{}, Document{}, err
	}
	working, err := w.doc.Clone()
	if err != nil {
		return Receipt{}, Document{}, err
	}
	for index, operation := range transaction.Operations {
		if err := applyOperation(&working, operation); err != nil {
			return Receipt{}, Document{}, fmt.Errorf("operation %d: %w", index, err)
		}
	}
	working.Revision++
	if err := working.Validate(); err != nil {
		return Receipt{}, Document{}, fmt.Errorf("transaction result: %w", err)
	}
	beforeFingerprint, err := before.Fingerprint()
	if err != nil {
		return Receipt{}, Document{}, err
	}
	afterFingerprint, err := working.Fingerprint()
	if err != nil {
		return Receipt{}, Document{}, err
	}
	receipt := Receipt{TransactionID: transaction.ID, Actor: transaction.Actor, Mode: transaction.Mode, Applied: transaction.Mode == ModeDirect, BeforeRevision: before.Revision, AfterRevision: working.Revision, BeforeFingerprint: beforeFingerprint, AfterFingerprint: afterFingerprint, Operations: len(transaction.Operations)}
	preview, err := working.Clone()
	if err != nil {
		return Receipt{}, Document{}, err
	}
	if transaction.Mode == ModeDirect {
		w.history = append(w.history, before)
		w.doc = working
	}
	return receipt, preview, nil
}

func (w *Workspace) Undo(expectedRevision uint64, actor string) (Receipt, Document, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if expectedRevision != w.doc.Revision {
		return Receipt{}, Document{}, fmt.Errorf("%w: have %d, expected %d", ErrRevisionConflict, w.doc.Revision, expectedRevision)
	}
	if len(w.history) == 0 {
		return Receipt{}, Document{}, fmt.Errorf("nothing to undo")
	}
	before, err := w.doc.Clone()
	if err != nil {
		return Receipt{}, Document{}, err
	}
	restored := w.history[len(w.history)-1]
	w.history = w.history[:len(w.history)-1]
	restored.Revision = before.Revision + 1
	if err := restored.Validate(); err != nil {
		return Receipt{}, Document{}, err
	}
	beforeFingerprint, err := before.Fingerprint()
	if err != nil {
		return Receipt{}, Document{}, err
	}
	afterFingerprint, err := restored.Fingerprint()
	if err != nil {
		return Receipt{}, Document{}, err
	}
	w.doc = restored
	preview, err := restored.Clone()
	if err != nil {
		return Receipt{}, Document{}, err
	}
	return Receipt{TransactionID: "undo", Actor: actor, Mode: ModeDirect, Applied: true, BeforeRevision: before.Revision, AfterRevision: restored.Revision, BeforeFingerprint: beforeFingerprint, AfterFingerprint: afterFingerprint, Operations: 1}, preview, nil
}

func applyOperation(document *Document, operation Operation) error {
	entity, ok := document.Entities[operation.Target]
	if !ok {
		return fmt.Errorf("entity %q does not exist", operation.Target)
	}
	if entity.Locked {
		return fmt.Errorf("entity %q is locked", operation.Target)
	}
	switch operation.Kind {
	case OpSetTransform:
		if operation.Transform == nil {
			return fmt.Errorf("set-transform requires transform")
		}
		entity.Transform = *operation.Transform
	case OpAssignMaterial:
		if entity.Mesh == nil {
			return fmt.Errorf("entity %q has no mesh", operation.Target)
		}
		if _, ok := document.Materials[operation.Material]; !ok {
			return fmt.Errorf("material %q does not exist", operation.Material)
		}
		entity.Mesh.Material = operation.Material
	case OpRenameEntity:
		if strings.TrimSpace(operation.Name) == "" {
			return fmt.Errorf("rename-entity requires name")
		}
		entity.Name = strings.TrimSpace(operation.Name)
	default:
		return fmt.Errorf("unsupported operation %q", operation.Kind)
	}
	document.Entities[operation.Target] = entity
	return nil
}

type ActionCapability struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Atomic bool   `json:"atomic"`
	Undo   bool   `json:"undo"`
}

func ActionCatalog() []ActionCapability {
	return []ActionCapability{
		{ID: string(OpSetTransform), Status: "available", Atomic: true, Undo: true},
		{ID: string(OpAssignMaterial), Status: "available", Atomic: true, Undo: true},
		{ID: string(OpRenameEntity), Status: "available", Atomic: true, Undo: true},
		{ID: "create-entity", Status: "planned", Atomic: true, Undo: true},
		{ID: "delete-entity", Status: "planned", Atomic: true, Undo: true},
		{ID: "reparent-entity", Status: "planned", Atomic: true, Undo: true},
		{ID: "duplicate-entity", Status: "planned", Atomic: true, Undo: true},
		{ID: "extrude-faces", Status: "planned", Atomic: true, Undo: true},
	}
}
