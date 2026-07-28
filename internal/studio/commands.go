package studio

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	ErrRevisionConflict = errors.New("studio revision conflict")
	ErrUnsavedChanges   = errors.New("studio project has unsaved changes")
)

type TransactionMode string

const (
	ModePropose TransactionMode = "propose"
	ModeDirect  TransactionMode = "direct"
)

type OperationKind string

const (
	OpSetField              OperationKind = "set-field"
	OpSetTransform          OperationKind = "set-transform"
	OpAssignMaterial        OperationKind = "assign-material"
	OpRenameEntity          OperationKind = "rename-entity"
	OpCreateEntity          OperationKind = "create-entity"
	OpDeleteEntity          OperationKind = "delete-entity"
	OpReparentEntity        OperationKind = "reparent-entity"
	OpDuplicateEntity       OperationKind = "duplicate-entity"
	OpExtrudeFaces          OperationKind = "extrude-faces"
	OpInsetFaces            OperationKind = "inset-faces"
	OpTriangulateFaces      OperationKind = "triangulate-faces"
	OpWeldVertices          OperationKind = "weld-vertices"
	OpFillFace              OperationKind = "fill-face"
	OpRecalculateNormals    OperationKind = "recalculate-normals"
	OpProjectPlanarUV       OperationKind = "project-planar-uv"
	OpDissolveEdges         OperationKind = "dissolve-edges"
	OpBevelEdges            OperationKind = "bevel-edges"
	OpBridgeLoops           OperationKind = "bridge-loops"
	OpLoopCut               OperationKind = "loop-cut"
	OpSetCurveControlPoint  OperationKind = "set-curve-control-point"
	OpSetModifier           OperationKind = "set-modifier"
	OpRemoveModifier        OperationKind = "remove-modifier"
	OpReorderModifier       OperationKind = "reorder-modifier"
	OpApplyModifier         OperationKind = "apply-modifier"
	OpCSGBoolean            OperationKind = "csg-boolean"
	OpSetMaterial           OperationKind = "set-material"
	OpDeleteMaterial        OperationKind = "delete-material"
	OpCapturePrefab         OperationKind = "capture-prefab"
	OpInstantiatePrefab     OperationKind = "instantiate-prefab"
	OpSetPrefabOverride     OperationKind = "set-prefab-override"
	OpDeletePrefab          OperationKind = "delete-prefab"
	OpRegisterAsset         OperationKind = "register-asset"
	OpDeleteAsset           OperationKind = "delete-asset"
	OpReimportAsset         OperationKind = "reimport-asset"
	OpSetBonePose           OperationKind = "set-bone-pose"
	OpSetAnimationKey       OperationKind = "set-animation-key"
	OpSolveIK               OperationKind = "solve-ik"
	OpSetPhysicsBody        OperationKind = "set-physics-body"
	OpSimulateTicks         OperationKind = "simulate-ticks"
	OpSetSimulationJoint    OperationKind = "set-simulation-joint"
	OpRetargetAnimation     OperationKind = "retarget-animation"
	OpSetAnimationParameter OperationKind = "set-animation-parameter"
	OpStepAnimationMachine  OperationKind = "step-animation-machine"
	OpSetRenderGraph        OperationKind = "set-render-graph"
	OpCollectUnusedAssets   OperationKind = "collect-unused-assets"
)

type Transaction struct {
	ID               string          `json:"id"`
	Actor            string          `json:"actor"`
	Mode             TransactionMode `json:"mode"`
	ExpectedRevision uint64          `json:"expectedRevision"`
	Operations       []Operation     `json:"operations"`
}

type Operation struct {
	Kind            OperationKind         `json:"kind"`
	Target          ID                    `json:"target,omitempty"`
	Transform       *Transform            `json:"transform,omitempty"`
	Material        ID                    `json:"material,omitempty"`
	Name            string                `json:"name,omitempty"`
	Field           string                `json:"field,omitempty"`
	Value           json.RawMessage       `json:"value,omitempty"`
	Entity          *Entity               `json:"entity,omitempty"`
	Parent          ID                    `json:"parent,omitempty"`
	NewID           ID                    `json:"newId,omitempty"`
	Faces           []ID                  `json:"faces,omitempty"`
	Vertices        []ID                  `json:"vertices,omitempty"`
	Edges           []ID                  `json:"edges,omitempty"`
	Loops           [][]ID                `json:"loops,omitempty"`
	Distance        float64               `json:"distance,omitempty"`
	Amount          float64               `json:"amount,omitempty"`
	Tolerance       float64               `json:"tolerance,omitempty"`
	CoordinateSpace string                `json:"coordinateSpace,omitempty"`
	Projection      string                `json:"projection,omitempty"`
	Closed          bool                  `json:"closed,omitempty"`
	ControlPoint    ID                    `json:"controlPoint,omitempty"`
	Position        *Vec3                 `json:"position,omitempty"`
	Weight          *float64              `json:"weight,omitempty"`
	Modifier        *Modifier             `json:"modifier,omitempty"`
	ModifierID      ID                    `json:"modifierId,omitempty"`
	ModifierIndex   int                   `json:"modifierIndex,omitempty"`
	Operand         ID                    `json:"operand,omitempty"`
	Boolean         string                `json:"boolean,omitempty"`
	VoxelSize       float64               `json:"voxelSize,omitempty"`
	MaterialRecord  *Material             `json:"materialRecord,omitempty"`
	MaterialID      ID                    `json:"materialId,omitempty"`
	PrefabID        ID                    `json:"prefabId,omitempty"`
	PrefabEntity    ID                    `json:"prefabEntity,omitempty"`
	PrefabOverride  *PrefabEntityOverride `json:"prefabOverride,omitempty"`
	Asset           *AssetRecord          `json:"asset,omitempty"`
	AssetID         ID                    `json:"assetId,omitempty"`
	ArmatureID      ID                    `json:"armatureId,omitempty"`
	BoneID          ID                    `json:"boneId,omitempty"`
	ClipID          ID                    `json:"clipId,omitempty"`
	TrackID         ID                    `json:"trackId,omitempty"`
	ConstraintID    ID                    `json:"constraintId,omitempty"`
	Physics         *PhysicsBody          `json:"physics,omitempty"`
	Joint           *PhysicsJoint         `json:"joint,omitempty"`
	JointID         ID                    `json:"jointId,omitempty"`
	RenderGraph     *RenderGraph          `json:"renderGraph,omitempty"`
	SimulationID    ID                    `json:"simulationId,omitempty"`
	Ticks           uint64                `json:"ticks,omitempty"`
	Inputs          []SimulationInput     `json:"inputs,omitempty"`
	RetargetMapID   ID                    `json:"retargetMapId,omitempty"`
	SourceClipID    ID                    `json:"sourceClipId,omitempty"`
	MachineID       ID                    `json:"machineId,omitempty"`
	Parameter       string                `json:"parameter,omitempty"`
	Number          float64               `json:"number,omitempty"`
	DeltaTime       float64               `json:"deltaTime,omitempty"`
	Key             *TransformKey         `json:"key,omitempty"`
}

type Receipt struct {
	TransactionID          string             `json:"transactionId"`
	Actor                  string             `json:"actor"`
	Mode                   TransactionMode    `json:"mode"`
	Applied                bool               `json:"applied"`
	BeforeRevision         uint64             `json:"beforeRevision"`
	AfterRevision          uint64             `json:"afterRevision"`
	BeforeFingerprint      string             `json:"beforeFingerprint"`
	AfterFingerprint       string             `json:"afterFingerprint"`
	Operations             int                `json:"operations"`
	Affected               []ID               `json:"affected,omitempty"`
	InverseTransactionID   string             `json:"inverseTransactionId,omitempty"`
	TelemetryCorrelationID string             `json:"telemetryCorrelationId"`
	Changes                []SemanticChange   `json:"changes,omitempty"`
	OperatorRecords        []OperatorRecord   `json:"operatorRecords,omitempty"`
	MaterialChanges        []MaterialChange   `json:"materialChanges,omitempty"`
	PrefabChanges          []PrefabChange     `json:"prefabChanges,omitempty"`
	AssetChanges           []AssetChange      `json:"assetChanges,omitempty"`
	RigChanges             []RigChange        `json:"rigChanges,omitempty"`
	AnimationChanges       []AnimationChange  `json:"animationChanges,omitempty"`
	SimulationChanges      []SimulationChange `json:"simulationChanges,omitempty"`
	RetargetChanges        []RetargetChange   `json:"retargetChanges,omitempty"`
	MachineChanges         []MachineChange    `json:"machineChanges,omitempty"`
}

type OperatorRecord struct {
	Kind            OperationKind `json:"kind"`
	Target          ID            `json:"target"`
	SelectionMode   SelectionMode `json:"selectionMode,omitempty"`
	Selection       []ID          `json:"selection,omitempty"`
	Distance        float64       `json:"distance,omitempty"`
	Amount          float64       `json:"amount,omitempty"`
	Tolerance       float64       `json:"tolerance,omitempty"`
	CoordinateSpace string        `json:"coordinateSpace"`
	Projection      string        `json:"projection,omitempty"`
	Closed          bool          `json:"closed,omitempty"`
	Operand         ID            `json:"operand,omitempty"`
	Boolean         string        `json:"boolean,omitempty"`
	VoxelSize       float64       `json:"voxelSize,omitempty"`
	Modifier        *Modifier     `json:"modifier,omitempty"`
	ModifierID      ID            `json:"modifierId,omitempty"`
	ModifierIndex   int           `json:"modifierIndex,omitempty"`
	Result          []ID          `json:"result"`
	UndoPolicy      string        `json:"undoPolicy"`
}

type SemanticChange struct {
	Kind   OperationKind `json:"kind"`
	Target ID            `json:"target"`
	Before *Entity       `json:"before,omitempty"`
	After  *Entity       `json:"after,omitempty"`
}

type MaterialChange struct {
	Kind   OperationKind `json:"kind"`
	ID     ID            `json:"id"`
	Before *Material     `json:"before,omitempty"`
	After  *Material     `json:"after,omitempty"`
}

type PrefabChange struct {
	Kind   OperationKind     `json:"kind"`
	ID     ID                `json:"id"`
	Before *PrefabDefinition `json:"before,omitempty"`
	After  *PrefabDefinition `json:"after,omitempty"`
}

type AssetChange struct {
	Kind   OperationKind `json:"kind"`
	ID     ID            `json:"id"`
	Before *AssetRecord  `json:"before,omitempty"`
	After  *AssetRecord  `json:"after,omitempty"`
}

type RigChange struct {
	Armature ID         `json:"armature"`
	Bone     ID         `json:"bone"`
	Before   *Transform `json:"before,omitempty"`
	After    *Transform `json:"after,omitempty"`
}

type AnimationChange struct {
	Clip   ID            `json:"clip"`
	Track  ID            `json:"track"`
	Before *TransformKey `json:"before,omitempty"`
	After  *TransformKey `json:"after,omitempty"`
}

type SimulationChange struct {
	Profile    ID     `json:"profile"`
	Ticks      uint64 `json:"ticks"`
	BeforeHash string `json:"beforeHash"`
	AfterHash  string `json:"afterHash"`
}

type RetargetChange struct {
	Map     ID `json:"map"`
	Source  ID `json:"source"`
	Created ID `json:"created"`
}
type MachineChange struct {
	Machine     ID      `json:"machine"`
	BeforeState ID      `json:"beforeState"`
	AfterState  ID      `json:"afterState"`
	BeforeTime  float64 `json:"beforeTime"`
	AfterTime   float64 `json:"afterTime"`
}

type historyEntry struct {
	before, after Document
	transaction   Transaction
}
type workspaceJournal interface {
	Commit(Transaction, Receipt, Document) error
	Save(Document) error
}

type Workspace struct {
	mu sync.RWMutex
	doc Document
	// docFingerprint is the fingerprint of doc, or "" when it has not been
	// computed since doc last changed. Fingerprint marshals the entire
	// SceneDoc, and every transaction needs the fingerprint of the document
	// it started from — which is the one the previous transaction just
	// produced. Carrying it forward removes one full marshal per command.
	docFingerprint       string
	undo                 []historyEntry
	redo                 []historyEntry
	journal              workspaceJournal
	selection            []ID
	selectionState       SelectionState
	viewportConfirmation *SelectionConfirmation
	play                 *playSession
	receipts             []Receipt
	savedRevision        uint64
	recovered            bool
	projectDir           string
}

func (w *Workspace) Select(ids ...ID) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.selectLocked(SelectionRequest{ExpectedRevision: w.doc.Revision, Mode: SelectionObject, IDs: ids})
}

func (w *Workspace) Selection() []ID {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return append([]ID(nil), w.selection...)
}

func NewWorkspace(document Document) (*Workspace, error) { return newWorkspace(document, nil) }
func newWorkspace(document Document, journal workspaceJournal) (*Workspace, error) {
	var err error
	document, err = MigrateDocument(document)
	if err != nil {
		return nil, err
	}
	clone, err := document.Clone()
	if err != nil {
		return nil, err
	}
	return &Workspace{doc: clone, journal: journal, savedRevision: clone.Revision, selectionState: SelectionState{Revision: clone.Revision, Mode: SelectionObject, IDs: []ID{}}}, nil
}

type ProjectStatus struct {
	Directory     string `json:"directory,omitempty"`
	DocumentPath  string `json:"documentPath,omitempty"`
	JournalPath   string `json:"journalPath,omitempty"`
	Revision      uint64 `json:"revision"`
	SavedRevision uint64 `json:"savedRevision"`
	Dirty         bool   `json:"dirty"`
	Recovered     bool   `json:"recovered"`
	// CompactionWarning names a failed journal trim. Commands stay durable
	// when it is set; the journal keeps more history than its bound.
	CompactionWarning string `json:"compactionWarning,omitempty"`
	// UndoDepth and UndoLimit make the bounded history observable, so a user
	// who cannot undo further can tell that the step was dropped rather than
	// that undo is broken.
	UndoDepth int `json:"undoDepth"`
	UndoLimit int `json:"undoLimit"`
}

// journalCompactor is implemented by journals that trim their own tail.
type journalCompactor interface{ CompactionWarning() string }

func (w *Workspace) ProjectStatus() ProjectStatus {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return projectStatusLocked(w)
}

func (w *Workspace) Save() (ProjectStatus, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.journal == nil {
		return ProjectStatus{}, fmt.Errorf("workspace has no project persistence")
	}
	if err := w.journal.Save(w.doc); err != nil {
		return ProjectStatus{}, err
	}
	w.savedRevision = w.doc.Revision
	w.recovered = false
	return ProjectStatus{Directory: w.projectDir, DocumentPath: filepath.Join(w.projectDir, "scene.scene3d"), JournalPath: filepath.Join(w.projectDir, "commands.jsonl"), Revision: w.doc.Revision, SavedRevision: w.savedRevision}, nil
}

// SwitchProjectFile replaces the current workspace with an existing canonical
// project selected by the trusted desktop host. ExpectedRevision prevents a
// concurrent command from being discarded while the native dialog is open.
func (w *Workspace) SwitchProjectFile(path string, expectedRevision uint64, discardUnsaved bool) (ProjectStatus, error) {
	resolved, err := canonicalProjectFile(path)
	if err != nil {
		return ProjectStatus{}, err
	}
	candidate, err := OpenWorkspace(filepath.Dir(resolved), Document{})
	if err != nil {
		return ProjectStatus{}, err
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	if w.doc.Revision != expectedRevision {
		return ProjectStatus{}, fmt.Errorf("%w: have %d, expected %d", ErrRevisionConflict, w.doc.Revision, expectedRevision)
	}
	if w.doc.Revision != w.savedRevision && !discardUnsaved {
		return ProjectStatus{}, ErrUnsavedChanges
	}
	candidate.mu.RLock()
	// Empty fingerprint: the newly opened document has not been hashed yet.
	w.setDocumentLocked(candidate.doc, candidate.docFingerprint)
	w.journal = candidate.journal
	w.savedRevision = candidate.savedRevision
	w.recovered = candidate.recovered
	w.projectDir = candidate.projectDir
	candidate.mu.RUnlock()
	// Every field describing the previous project has to go. Receipts, the
	// play session, and the viewport confirmation used to survive a switch,
	// so the command console and Agent Actions panel attributed the old
	// project's history to the newly opened one.
	w.undo = nil
	w.redo = nil
	w.selection = nil
	w.receipts = nil
	w.play = nil
	w.viewportConfirmation = nil
	w.selectionState = SelectionState{Revision: w.doc.Revision, Mode: SelectionObject, IDs: []ID{}}
	return projectStatusLocked(w), nil
}

func canonicalProjectFile(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("project file path is required")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve project file: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("resolve project file links: %w", err)
	}
	if !strings.EqualFold(filepath.Base(resolved), "scene.scene3d") {
		return "", fmt.Errorf("project entry must be named scene.scene3d")
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("project entry is not a regular file")
	}
	return resolved, nil
}

func projectStatusLocked(w *Workspace) ProjectStatus {
	status := ProjectStatus{Directory: w.projectDir, Revision: w.doc.Revision, SavedRevision: w.savedRevision, Dirty: w.doc.Revision != w.savedRevision, Recovered: w.recovered, UndoDepth: len(w.undo), UndoLimit: undoHistoryLimit}
	if compactor, ok := w.journal.(journalCompactor); ok {
		status.CompactionWarning = compactor.CompactionWarning()
	}
	if w.projectDir != "" {
		status.DocumentPath = filepath.Join(w.projectDir, "scene.scene3d")
		status.JournalPath = filepath.Join(w.projectDir, "commands.jsonl")
	}
	return status
}

func (w *Workspace) Snapshot() (Document, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.doc.Clone()
}

// Read runs fn against the live document under a read lock.
//
// It exists because Snapshot deep-clones through JSON: marshal plus unmarshal
// of the whole SceneDoc. Readers that wanted one entity's transform paid the
// cost of the entire scene, which on a large authored mesh is hundreds of
// milliseconds per viewport interaction.
//
// fn must treat the document as immutable and must not retain it or anything
// reachable from it past the call. fn must not call back into the workspace:
// this holds the read lock, and a write would deadlock. Use Snapshot when the
// value has to outlive the call.
func (w *Workspace) Read(fn func(document *Document) error) error {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return fn(&w.doc)
}

// setDocumentLocked installs a new canonical document and the fingerprint that
// describes it. Pass an empty fingerprint to have it computed on next use.
// Callers must hold w.mu for writing.
func (w *Workspace) setDocumentLocked(document Document, fingerprint string) {
	w.doc = document
	w.docFingerprint = fingerprint
}

// documentFingerprintForRead returns the current document's fingerprint
// without writing the cache. Callers hold w.mu for reading, and filling the
// cache under a read lock would be a data race. Every commit records a
// fingerprint, so the uncached path only runs before the first commit against
// a freshly opened project.
func (w *Workspace) documentFingerprintForRead() (string, error) {
	if w.docFingerprint != "" {
		return w.docFingerprint, nil
	}
	return w.doc.Fingerprint()
}

// documentFingerprintLocked returns the current document's fingerprint,
// computing and caching it if a previous write did not supply one. Callers
// must hold w.mu for writing.
func (w *Workspace) documentFingerprintLocked() (string, error) {
	if w.docFingerprint != "" {
		return w.docFingerprint, nil
	}
	fingerprint, err := w.doc.Fingerprint()
	if err != nil {
		return "", err
	}
	w.docFingerprint = fingerprint
	return fingerprint, nil
}

func (w *Workspace) Execute(transaction Transaction) (Receipt, Document, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := validateTransaction(transaction, w.doc.Revision); err != nil {
		return Receipt{}, Document{}, err
	}
	// `before` is the outgoing document, not a copy of it. Nothing mutates a
	// document in place: operations run against `working`, and a commit
	// rebinds w.doc rather than editing it. Cloning here bought nothing and
	// cost a full marshal-and-unmarshal of the whole scene per command.
	before := w.doc
	beforeFingerprint, err := w.documentFingerprintLocked()
	if err != nil {
		return Receipt{}, Document{}, err
	}
	working, err := w.doc.Clone()
	if err != nil {
		return Receipt{}, Document{}, err
	}
	affected := make([]ID, 0, len(transaction.Operations))
	for index, operation := range transaction.Operations {
		ids, applyErr := applyOperation(&working, operation)
		if applyErr != nil {
			return Receipt{}, Document{}, fmt.Errorf("operation %d: %w", index, applyErr)
		}
		affected = append(affected, ids...)
	}
	working.Revision++
	if err := working.Validate(); err != nil {
		return Receipt{}, Document{}, fmt.Errorf("transaction result: %w", err)
	}
	afterFingerprint, err := working.Fingerprint()
	if err != nil {
		return Receipt{}, Document{}, err
	}
	receipt, err := makeReceipt(transaction, before, working, beforeFingerprint, afterFingerprint, affected)
	if err != nil {
		return Receipt{}, Document{}, err
	}
	if transaction.Mode != ModeDirect {
		// A proposal never becomes canonical state, so `working` is already a
		// private copy the caller may keep. Cloning it again only to discard
		// the original doubled the cost of every preview.
		w.recordReceiptLocked(receipt)
		return receipt, working, nil
	}
	// A direct commit keeps `working` as canonical state, so the caller gets
	// its own copy. Handing back the live document would let a caller mutate
	// the workspace by editing a value it was merely shown.
	preview, err := working.Clone()
	if err != nil {
		return Receipt{}, Document{}, err
	}
	if w.journal != nil {
		if err := w.journal.Commit(transaction, receipt, working); err != nil {
			return Receipt{}, Document{}, err
		}
	}
	w.undo = append(w.undo, historyEntry{before: before, after: working, transaction: transaction})
	w.trimUndoLocked()
	w.redo = nil
	w.setDocumentLocked(working, afterFingerprint)
	w.selectionState = reconcileSelection(working, w.selectionState)
	w.syncLegacySelectionLocked()
	w.recordReceiptLocked(receipt)
	return receipt, preview, nil
}

// undoHistoryLimit bounds the undo stack. Each entry retains the complete
// document before and after its transaction, so the stack grows by twice the
// document size per edit. Unbounded, a long session on a large authored scene
// consumed gigabytes. The oldest steps are dropped, never the newest.
const undoHistoryLimit = 128

// trimUndoLocked enforces undoHistoryLimit. Callers must hold w.mu.
func (w *Workspace) trimUndoLocked() {
	if len(w.undo) <= undoHistoryLimit {
		return
	}
	dropped := len(w.undo) - undoHistoryLimit
	// Re-slice into a fresh backing array so the dropped documents become
	// unreachable instead of being pinned by the slice header.
	kept := make([]historyEntry, undoHistoryLimit)
	copy(kept, w.undo[dropped:])
	w.undo = kept
}

// recordReceiptLocked keeps a bounded newest-last receipt history so command
// attribution stays visible to the UI and agents without replaying the
// journal. Callers must hold w.mu.
func (w *Workspace) recordReceiptLocked(receipt Receipt) {
	const receiptHistoryLimit = 32
	w.receipts = append(w.receipts, receipt)
	if len(w.receipts) > receiptHistoryLimit {
		w.receipts = w.receipts[len(w.receipts)-receiptHistoryLimit:]
	}
}

// RecentReceipts returns up to limit receipts, newest first.
func (w *Workspace) RecentReceipts(limit int) []Receipt {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if limit <= 0 || limit > len(w.receipts) {
		limit = len(w.receipts)
	}
	out := make([]Receipt, 0, limit)
	for i := len(w.receipts) - 1; i >= len(w.receipts)-limit; i-- {
		out = append(out, w.receipts[i])
	}
	return out
}

func validateTransaction(transaction Transaction, revision uint64) error {
	if transaction.ExpectedRevision != revision {
		return fmt.Errorf("%w: have %d, expected %d", ErrRevisionConflict, revision, transaction.ExpectedRevision)
	}
	if transaction.Mode != ModePropose && transaction.Mode != ModeDirect {
		return fmt.Errorf("unsupported transaction mode %q", transaction.Mode)
	}
	if strings.TrimSpace(transaction.ID) == "" || strings.TrimSpace(transaction.Actor) == "" {
		return fmt.Errorf("transaction id and actor are required")
	}
	if len(transaction.Operations) == 0 {
		return fmt.Errorf("transaction requires at least one operation")
	}
	return nil
}

// makeReceipt takes both fingerprints already computed. Each one marshals a
// whole SceneDoc, and callers can supply them from a cache: the fingerprint of
// the document a transaction starts from is the one the previous transaction
// produced.
func makeReceipt(transaction Transaction, before, after Document, beforeFingerprint, afterFingerprint string, affected []ID) (Receipt, error) {
	correlation := afterFingerprint
	if len(correlation) > 12 {
		correlation = correlation[:12]
	}
	return Receipt{TransactionID: transaction.ID, Actor: transaction.Actor, Mode: transaction.Mode, Applied: transaction.Mode == ModeDirect, BeforeRevision: before.Revision, AfterRevision: after.Revision, BeforeFingerprint: beforeFingerprint, AfterFingerprint: afterFingerprint, Operations: len(transaction.Operations), Affected: uniqueIDs(affected), InverseTransactionID: "inverse:" + transaction.ID, TelemetryCorrelationID: "studio:" + transaction.ID + ":" + correlation, Changes: semanticChanges(transaction, before, after), OperatorRecords: operatorRecords(transaction), MaterialChanges: materialChanges(transaction, before, after), PrefabChanges: prefabChanges(transaction, before, after), AssetChanges: assetChanges(transaction, before, after), RigChanges: rigChanges(transaction, before, after), AnimationChanges: animationChanges(transaction, before, after), SimulationChanges: simulationChanges(transaction, before, after), RetargetChanges: retargetChanges(transaction), MachineChanges: machineChanges(transaction, before, after)}, nil
}

func operatorRecords(transaction Transaction) []OperatorRecord {
	records := make([]OperatorRecord, 0, len(transaction.Operations))
	for _, operation := range transaction.Operations {
		target := operation.Target
		if target == "" && operation.Entity != nil {
			target = operation.Entity.ID
		}
		mode := SelectionMode("")
		selection := operation.Faces
		if len(selection) > 0 {
			mode = SelectionFace
		} else if len(operation.Edges) > 0 {
			mode = SelectionEdge
			selection = operation.Edges
		} else if len(operation.Vertices) > 0 {
			mode = SelectionVertex
			selection = operation.Vertices
		}
		space := operation.CoordinateSpace
		if space == "" {
			space = "object"
		}
		policy := "inverse-or-checkpoint"
		if isMeshOperation(operation.Kind) {
			policy = "geometry-checkpoint"
		}
		if len(operation.Loops) > 0 {
			mode = SelectionVertex
			selection = nil
			for _, loop := range operation.Loops {
				selection = append(selection, loop...)
			}
		}
		if operation.ControlPoint != "" {
			mode = SelectionCurveControlPoint
			selection = []ID{operation.ControlPoint}
		}
		result := []ID{target}
		if operation.Kind == OpCSGBoolean {
			mode = SelectionObject
			selection = []ID{target, operation.Operand}
			result = []ID{operation.NewID}
		}
		if operation.Kind == OpSetMaterial && operation.MaterialRecord != nil {
			target = operation.MaterialRecord.ID
			result = []ID{target}
		}
		if operation.Kind == OpDeleteMaterial {
			target = operation.MaterialID
			result = []ID{target}
		}
		if operation.Kind == OpCapturePrefab || operation.Kind == OpDeletePrefab {
			target = operation.PrefabID
			result = []ID{target}
		}
		if operation.Kind == OpInstantiatePrefab {
			target = operation.PrefabID
			result = []ID{operation.NewID}
		}
		if operation.Kind == OpRegisterAsset && operation.Asset != nil {
			target = operation.Asset.ID
			result = []ID{target}
		}
		if operation.Kind == OpDeleteAsset {
			target = operation.AssetID
			result = []ID{target}
		}
		if operation.Kind == OpReimportAsset && operation.Asset != nil {
			target = operation.AssetID
			result = []ID{operation.Asset.ID}
		}
		records = append(records, OperatorRecord{Kind: operation.Kind, Target: target, SelectionMode: mode, Selection: append([]ID(nil), selection...), Distance: operation.Distance, Amount: operation.Amount, Tolerance: operation.Tolerance, CoordinateSpace: space, Projection: operation.Projection, Closed: operation.Closed, Operand: operation.Operand, Boolean: operation.Boolean, VoxelSize: operation.VoxelSize, Modifier: operation.Modifier, ModifierID: operation.ModifierID, ModifierIndex: operation.ModifierIndex, Result: result, UndoPolicy: policy})
	}
	return records
}

func materialChanges(transaction Transaction, before, after Document) []MaterialChange {
	out := []MaterialChange{}
	for _, operation := range transaction.Operations {
		var id ID
		switch operation.Kind {
		case OpSetMaterial:
			if operation.MaterialRecord != nil {
				id = operation.MaterialRecord.ID
			}
		case OpDeleteMaterial:
			id = operation.MaterialID
		default:
			continue
		}
		change := MaterialChange{Kind: operation.Kind, ID: id}
		if material, ok := before.Materials[id]; ok {
			copy := material
			change.Before = &copy
		}
		if material, ok := after.Materials[id]; ok {
			copy := material
			change.After = &copy
		}
		out = append(out, change)
	}
	return out
}

func prefabChanges(transaction Transaction, before, after Document) []PrefabChange {
	out := []PrefabChange{}
	for _, operation := range transaction.Operations {
		if operation.Kind != OpCapturePrefab && operation.Kind != OpDeletePrefab {
			continue
		}
		change := PrefabChange{Kind: operation.Kind, ID: operation.PrefabID}
		if value, ok := before.Prefabs[operation.PrefabID]; ok {
			copy := value
			change.Before = &copy
		}
		if value, ok := after.Prefabs[operation.PrefabID]; ok {
			copy := value
			change.After = &copy
		}
		out = append(out, change)
	}
	return out
}

func assetChanges(transaction Transaction, before, after Document) []AssetChange {
	out := []AssetChange{}
	for _, operation := range transaction.Operations {
		var id ID
		switch operation.Kind {
		case OpRegisterAsset:
			if operation.Asset != nil {
				id = operation.Asset.ID
			}
		case OpDeleteAsset:
			id = operation.AssetID
		case OpReimportAsset:
			if operation.Asset == nil {
				continue
			}
			oldID := operation.AssetID
			change := AssetChange{Kind: operation.Kind, ID: oldID}
			if value, ok := before.Assets[oldID]; ok {
				copy := value
				change.Before = &copy
			}
			if oldID == operation.Asset.ID {
				copy := *operation.Asset
				change.After = &copy
				out = append(out, change)
				continue
			}
			out = append(out, change)
			id = operation.Asset.ID
		default:
			continue
		}
		change := AssetChange{Kind: operation.Kind, ID: id}
		if value, ok := before.Assets[id]; ok {
			copy := value
			change.Before = &copy
		}
		if value, ok := after.Assets[id]; ok {
			copy := value
			change.After = &copy
		}
		out = append(out, change)
	}
	return out
}

func isMeshOperation(kind OperationKind) bool {
	switch kind {
	case OpExtrudeFaces, OpInsetFaces, OpTriangulateFaces, OpWeldVertices, OpFillFace, OpRecalculateNormals, OpProjectPlanarUV, OpDissolveEdges, OpBevelEdges, OpBridgeLoops, OpLoopCut, OpSetCurveControlPoint, OpSetModifier, OpRemoveModifier, OpApplyModifier, OpCSGBoolean:
		return true
	default:
		return false
	}
}

func semanticChanges(transaction Transaction, before, after Document) []SemanticChange {
	changes := make([]SemanticChange, 0, len(transaction.Operations))
	for _, operation := range transaction.Operations {
		target := operation.Target
		if target == "" && operation.Entity != nil {
			target = operation.Entity.ID
		}
		change := SemanticChange{Kind: operation.Kind, Target: target}
		if entity, ok := before.Entities[target]; ok {
			copy := entity
			change.Before = &copy
		}
		if entity, ok := after.Entities[target]; ok {
			copy := entity
			change.After = &copy
		}
		changes = append(changes, change)
	}
	return changes
}

func (w *Workspace) Undo(expectedRevision uint64, actor string) (Receipt, Document, error) {
	return w.moveHistory(expectedRevision, actor, true)
}
func (w *Workspace) Redo(expectedRevision uint64, actor string) (Receipt, Document, error) {
	return w.moveHistory(expectedRevision, actor, false)
}
func (w *Workspace) moveHistory(expectedRevision uint64, actor string, undo bool) (Receipt, Document, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if expectedRevision != w.doc.Revision {
		return Receipt{}, Document{}, fmt.Errorf("%w: have %d, expected %d", ErrRevisionConflict, w.doc.Revision, expectedRevision)
	}
	stack := &w.redo
	label := "redo"
	if undo {
		stack = &w.undo
		label = "undo"
	}
	if len(*stack) == 0 {
		return Receipt{}, Document{}, fmt.Errorf("nothing to %s", label)
	}
	entry := (*stack)[len(*stack)-1]
	*stack = (*stack)[:len(*stack)-1]
	// The outgoing document, not a copy: see the note in Execute.
	before := w.doc
	beforeFingerprint, err := w.documentFingerprintLocked()
	if err != nil {
		return Receipt{}, Document{}, err
	}
	target := entry.after
	if undo {
		target = entry.before
	}
	// The history entry stays on the opposite stack, so the document that
	// becomes canonical has to be its own copy.
	target, err = target.Clone()
	if err != nil {
		return Receipt{}, Document{}, err
	}
	target.Revision = before.Revision + 1
	afterFingerprint, err := target.Fingerprint()
	if err != nil {
		return Receipt{}, Document{}, err
	}
	transaction := Transaction{ID: label + ":" + entry.transaction.ID, Actor: actor, Mode: ModeDirect, ExpectedRevision: before.Revision, Operations: entry.transaction.Operations}
	receipt, err := makeReceipt(transaction, before, target, beforeFingerprint, afterFingerprint, nil)
	if err != nil {
		return Receipt{}, Document{}, err
	}
	if w.journal != nil {
		if err := w.journal.Commit(transaction, receipt, target); err != nil {
			return Receipt{}, Document{}, err
		}
	}
	w.setDocumentLocked(target, afterFingerprint)
	w.selectionState = reconcileSelection(target, w.selectionState)
	w.syncLegacySelectionLocked()
	if undo {
		w.redo = append(w.redo, entry)
	} else {
		w.undo = append(w.undo, entry)
	}
	w.recordReceiptLocked(receipt)
	// The caller gets its own copy: target is now canonical state. A discarded
	// error here returned an empty document as the result of an undo that had
	// already succeeded.
	preview, err := target.Clone()
	if err != nil {
		return Receipt{}, Document{}, err
	}
	return receipt, preview, nil
}

func applyOperation(document *Document, operation Operation) ([]ID, error) {
	switch operation.Kind {
	case OpCreateEntity:
		return applyCreate(document, operation)
	case OpDeleteEntity:
		return applyDelete(document, operation)
	case OpReparentEntity:
		return applyReparent(document, operation)
	case OpDuplicateEntity:
		return applyDuplicate(document, operation)
	case OpExtrudeFaces, OpInsetFaces, OpTriangulateFaces, OpWeldVertices, OpFillFace, OpRecalculateNormals, OpProjectPlanarUV, OpDissolveEdges, OpBevelEdges, OpBridgeLoops, OpLoopCut:
		return applyMeshOperation(document, operation)
	case OpSetCurveControlPoint:
		return applySetCurveControlPoint(document, operation)
	case OpSetModifier:
		return applySetModifier(document, operation)
	case OpRemoveModifier:
		return applyRemoveModifier(document, operation)
	case OpReorderModifier:
		return applyReorderModifier(document, operation)
	case OpApplyModifier:
		return applyApplyModifier(document, operation)
	case OpCSGBoolean:
		return applyCSGBoolean(document, operation)
	case OpSetMaterial:
		return applySetMaterial(document, operation)
	case OpDeleteMaterial:
		return applyDeleteMaterial(document, operation)
	case OpCapturePrefab:
		return applyCapturePrefab(document, operation)
	case OpInstantiatePrefab:
		return applyInstantiatePrefab(document, operation)
	case OpSetPrefabOverride:
		return applySetPrefabOverride(document, operation)
	case OpDeletePrefab:
		return applyDeletePrefab(document, operation)
	case OpRegisterAsset:
		return applyRegisterAsset(document, operation)
	case OpDeleteAsset:
		return applyDeleteAsset(document, operation)
	case OpReimportAsset:
		return applyReimportAsset(document, operation)
	case OpSetBonePose:
		return applySetBonePose(document, operation)
	case OpSetAnimationKey:
		return applySetAnimationKey(document, operation)
	case OpSolveIK:
		return applySolveIK(document, operation)
	case OpSetPhysicsBody:
		return applySetPhysicsBody(document, operation)
	case OpSimulateTicks:
		return applySimulateTicks(document, operation)
	case OpSetSimulationJoint:
		return applySetSimulationJoint(document, operation)
	case OpRetargetAnimation:
		return applyRetargetAnimation(document, operation)
	case OpSetAnimationParameter:
		return applySetAnimationParameter(document, operation)
	case OpStepAnimationMachine:
		return applyStepAnimationMachine(document, operation)
	case OpSetRenderGraph:
		return applySetRenderGraph(document, operation)
	}
	entity, ok := document.Entities[operation.Target]
	if !ok {
		return nil, fmt.Errorf("entity %q does not exist", operation.Target)
	}
	if entity.Locked {
		return nil, fmt.Errorf("entity %q is locked", operation.Target)
	}
	switch operation.Kind {
	case OpSetTransform:
		if operation.Transform == nil {
			return nil, fmt.Errorf("set-transform requires transform")
		}
		entity.Transform = *operation.Transform
	case OpAssignMaterial:
		if entity.Mesh == nil {
			return nil, fmt.Errorf("entity %q has no mesh", operation.Target)
		}
		if _, ok := document.Materials[operation.Material]; !ok {
			return nil, fmt.Errorf("material %q does not exist", operation.Material)
		}
		entity.Mesh.Material = operation.Material
	case OpRenameEntity:
		if strings.TrimSpace(operation.Name) == "" {
			return nil, fmt.Errorf("rename-entity requires name")
		}
		entity.Name = strings.TrimSpace(operation.Name)
	case OpSetField:
		if err := setEntityField(&entity, operation.Field, operation.Value); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported operation %q", operation.Kind)
	}
	document.Entities[operation.Target] = entity
	return []ID{operation.Target}, nil
}

func setEntityField(entity *Entity, field string, value json.RawMessage) error {
	switch field {
	case "visible":
		return json.Unmarshal(value, &entity.Visible)
	case "locked":
		return json.Unmarshal(value, &entity.Locked)
	case "name":
		var name string
		if err := json.Unmarshal(value, &name); err != nil {
			return err
		}
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("name is required")
		}
		entity.Name = strings.TrimSpace(name)
		return nil
	default:
		return fmt.Errorf("field %q is not settable", field)
	}
}

type ActionCapability struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Atomic  bool   `json:"atomic"`
	Undo    bool   `json:"undo"`
	Preview bool   `json:"preview"`
}

func ActionCatalog() []ActionCapability {
	kinds := []OperationKind{OpSetField, OpSetTransform, OpAssignMaterial, OpRenameEntity, OpCreateEntity, OpDeleteEntity, OpReparentEntity, OpDuplicateEntity, OpExtrudeFaces, OpInsetFaces, OpTriangulateFaces, OpWeldVertices, OpFillFace, OpRecalculateNormals, OpProjectPlanarUV, OpDissolveEdges, OpBevelEdges, OpBridgeLoops, OpLoopCut, OpSetCurveControlPoint, OpSetModifier, OpRemoveModifier, OpReorderModifier, OpApplyModifier, OpCSGBoolean, OpSetMaterial, OpDeleteMaterial, OpCapturePrefab, OpInstantiatePrefab, OpSetPrefabOverride, OpDeletePrefab, OpRegisterAsset, OpDeleteAsset, OpReimportAsset, OpSetBonePose, OpSetAnimationKey, OpSolveIK, OpSetPhysicsBody, OpSimulateTicks, OpSetSimulationJoint, OpRetargetAnimation, OpSetAnimationParameter, OpStepAnimationMachine, OpSetRenderGraph, OpCollectUnusedAssets}
	out := make([]ActionCapability, 0, len(kinds))
	for _, kind := range kinds {
		undo := kind != OpCollectUnusedAssets
		out = append(out, ActionCapability{ID: string(kind), Status: "available", Atomic: true, Undo: undo, Preview: true})
	}
	return out
}

func uniqueIDs(values []ID) []ID {
	seen := map[ID]bool{}
	out := make([]ID, 0, len(values))
	for _, value := range values {
		if value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}
