package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"m31labs.dev/gosx/scene"
	"m31labs.dev/gosx3d-studio/internal/studio"
)

const (
	maxWebMCPProposalOperations = 12
	maxWebMCPProposals          = 64
	webMCPProposalTTL           = 15 * time.Minute
)

var (
	errWebMCPProposalNotFound   = errors.New("WebMCP proposal not found")
	errWebMCPProposalExpired    = errors.New("WebMCP proposal expired")
	errWebMCPProposalNoChanges  = errors.New("WebMCP proposal does not change the scene")
	errWebMCPProposalCommitting = errors.New("a WebMCP proposal is already being committed")
)

// webMCPProposalRequest is intentionally narrower than studio.Transaction.
// The browser supplies the scene revision and reversible operations; the
// server owns transaction identity, actor, and authority mode.
type webMCPProposalRequest struct {
	ExpectedRevision uint64             `json:"expectedRevision"`
	Title            string             `json:"title"`
	Rationale        string             `json:"rationale"`
	Operations       []studio.Operation `json:"operations"`
}

type webMCPCommitRequest struct {
	ProposalID string `json:"proposalId"`
}

type webMCPProposal struct {
	ID          string
	Owner       string
	Title       string
	Rationale   string
	Transaction studio.Transaction
	Receipt     studio.Receipt
	Governance  []webMCPOperationDecision
	Preview     map[string]any
	Materials   map[string]string
	// SceneCommands render the proposal as a reversible, client-local preview.
	// Canonical authority remains unchanged until Commit executes the stored
	// transaction; Discard applies ReverseSceneCommands to the same live mount.
	SceneCommands        []scene.Command
	ReverseSceneCommands []scene.Command
	ExpiresAt            time.Time
	Claimed              bool
}

// webMCPProposalStore keeps staged edits exact and bounded. The browser only
// receives an opaque identifier, so the human commit applies the operations
// the agent actually previewed rather than a client-rewritten transaction.
type webMCPProposalStore struct {
	mu        sync.Mutex
	workspace *studio.Workspace
	policy    *webMCPOperationPolicy
	proposals map[string]webMCPProposal
	order     []string
	now       func() time.Time
}

func newWebMCPProposalStore(workspace *studio.Workspace, policy *webMCPOperationPolicy) *webMCPProposalStore {
	return &webMCPProposalStore{
		workspace: workspace,
		policy:    policy,
		proposals: map[string]webMCPProposal{},
		now:       time.Now,
	}
}

func decodeWebMCPProposal(reader io.Reader) (webMCPProposalRequest, error) {
	var request webMCPProposalRequest
	if err := decodeWebMCPJSON(reader, &request); err != nil {
		return webMCPProposalRequest{}, err
	}
	if err := validateWebMCPOperationCount(request.Operations); err != nil {
		return webMCPProposalRequest{}, err
	}
	request.Title = strings.TrimSpace(request.Title)
	request.Rationale = strings.TrimSpace(request.Rationale)
	if request.Title == "" {
		return webMCPProposalRequest{}, fmt.Errorf("title is required")
	}
	if len(request.Title) > 80 {
		return webMCPProposalRequest{}, fmt.Errorf("title exceeds 80 characters")
	}
	if len(request.Rationale) > 400 {
		return webMCPProposalRequest{}, fmt.Errorf("rationale exceeds 400 characters")
	}
	return request, nil
}

func decodeWebMCPCommit(reader io.Reader) (webMCPCommitRequest, error) {
	var request webMCPCommitRequest
	if err := decodeWebMCPJSON(reader, &request); err != nil {
		return webMCPCommitRequest{}, err
	}
	request.ProposalID = strings.TrimSpace(request.ProposalID)
	if request.ProposalID == "" {
		return webMCPCommitRequest{}, fmt.Errorf("proposalId is required")
	}
	return request, nil
}

func decodeWebMCPJSON(reader io.Reader, target any) error {
	decoder := json.NewDecoder(io.LimitReader(reader, 64<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("request body must contain exactly one JSON value")
		}
		return err
	}
	return nil
}

func validateWebMCPOperations(operations []studio.Operation) error {
	if err := validateWebMCPOperationCount(operations); err != nil {
		return err
	}
	for index, operation := range operations {
		if err := validateWebMCPOperation(operation); err != nil {
			return fmt.Errorf("operation %d: %w", index, err)
		}
	}
	return nil
}

func validateWebMCPOperationCount(operations []studio.Operation) error {
	if len(operations) == 0 {
		return fmt.Errorf("operations must contain at least one edit")
	}
	if len(operations) > maxWebMCPProposalOperations {
		return fmt.Errorf("operations contains %d edits; maximum is %d", len(operations), maxWebMCPProposalOperations)
	}
	return nil
}

func validateWebMCPOperation(operation studio.Operation) error {
	requireTarget := func() error {
		if operation.Target == "" {
			return fmt.Errorf("target is required for %s", operation.Kind)
		}
		return nil
	}
	switch operation.Kind {
	case studio.OpRenameEntity:
		if err := requireTarget(); err != nil {
			return err
		}
		if strings.TrimSpace(operation.Name) == "" {
			return fmt.Errorf("name is required for %s", operation.Kind)
		}
	case studio.OpSetTransform:
		if err := requireTarget(); err != nil {
			return err
		}
		if operation.Transform == nil {
			return fmt.Errorf("transform is required for %s", operation.Kind)
		}
	case studio.OpAssignMaterial:
		if err := requireTarget(); err != nil {
			return err
		}
		if operation.Material == "" {
			return fmt.Errorf("material is required for %s", operation.Kind)
		}
	default:
		return fmt.Errorf("policy allowed kind %q without a request validator", operation.Kind)
	}
	return nil
}

func (store *webMCPProposalStore) Stage(request webMCPProposalRequest, owner string) (map[string]any, error) {
	if owner == "" {
		return nil, fmt.Errorf("WebMCP proposal owner is required")
	}
	if err := validateWebMCPOperationCount(request.Operations); err != nil {
		return nil, err
	}
	governance := make([]webMCPOperationDecision, 0, len(request.Operations))
	for index, operation := range request.Operations {
		decision, err := store.policy.Decide(operation.Kind)
		if err != nil {
			return nil, fmt.Errorf("evaluate WebMCP operation %d policy: %w", index, err)
		}
		governance = append(governance, decision)
		if !decision.Allowed {
			return nil, fmt.Errorf("operation %d kind %q denied by WebMCP policy: %s", index, operation.Kind, decision.Reason)
		}
		if err := validateWebMCPOperation(operation); err != nil {
			return nil, fmt.Errorf("operation %d: %w", index, err)
		}
	}
	proposalID, err := newWebMCPProposalID()
	if err != nil {
		return nil, err
	}
	transaction := studio.Transaction{
		ID:               "webmcp-proposal:" + proposalID,
		Actor:            "agent://webmcp",
		Mode:             studio.ModePropose,
		ExpectedRevision: request.ExpectedRevision,
		Operations:       request.Operations,
	}
	// Serialize preview execution and insertion with Clear. The public demo
	// reset replaces canonical state and then clears staged proposals; holding
	// this lock prevents an in-flight Stage from inserting a pre-reset proposal
	// after that invalidation has completed.
	store.mu.Lock()
	defer store.mu.Unlock()
	store.pruneLocked(store.now())
	if store.hasClaimedOwnerLocked(owner) {
		return nil, errWebMCPProposalCommitting
	}
	var sameRevision, changesScene bool
	var canonicalScene scene.SceneIR
	if err := store.workspace.Read(func(document *studio.Document) error {
		sameRevision = document.Revision == request.ExpectedRevision
		if sameRevision {
			changesScene = webMCPOperationsChangeDocument(document, request.Operations)
			if changesScene {
				props, compileErr := studio.CompileViewport(*document)
				if compileErr != nil {
					return compileErr
				}
				canonicalScene = props.SceneIR()
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	if sameRevision && !changesScene {
		return nil, errWebMCPProposalNoChanges
	}
	receipt, preview, err := store.workspace.Execute(transaction)
	if err != nil {
		return nil, err
	}
	previewProps, err := studio.CompileViewport(preview)
	if err != nil {
		return nil, err
	}
	forward := scene.DiffScene(canonicalScene, previewProps.SceneIR(), scene.DiffOptions{})
	reverse := scene.DiffScene(previewProps.SceneIR(), canonicalScene, scene.DiffOptions{})
	var sceneCommands, reverseSceneCommands []scene.Command
	if len(forward.RemountFields) == 0 && len(reverse.RemountFields) == 0 {
		sceneCommands = forward.Commands
		reverseSceneCommands = reverse.Commands
	}
	expiresAt := store.now().Add(webMCPProposalTTL)
	previewSummary := webMCPDocumentSummary(preview)
	proposal := webMCPProposal{
		ID: proposalID, Owner: owner, Title: request.Title, Rationale: request.Rationale,
		Transaction: transaction, Receipt: receipt, Governance: governance,
		Preview: previewSummary, Materials: webMCPMaterialDisplayNames(receipt, preview),
		SceneCommands: sceneCommands, ReverseSceneCommands: reverseSceneCommands, ExpiresAt: expiresAt,
	}
	// A successful preview supersedes an older review from the same browser.
	// Failed validation or preview execution returns above and preserves it.
	store.removeOwnerProposalsLocked(owner)
	if err := store.makeRoomLocked(); err != nil {
		return nil, err
	}
	store.proposals[proposalID] = proposal
	store.order = append(store.order, proposalID)
	return webMCPProposalView(proposal), nil
}

// Current returns the newest unexpired proposal owned by this exact browser
// session whose review base is still the canonical workspace revision. Keeping
// the opaque proposal ID server-backed lets a human reload or restore the tab
// without losing the review boundary, while a different session learns nothing
// about proposals it does not own. A proposal based on an older revision can no
// longer produce the reviewed result, so Current removes it instead of letting
// the browser restore a stale live preview.
func (store *webMCPProposalStore) Current(owner string) map[string]any {
	if owner == "" {
		return nil
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.pruneLocked(store.now())
	var canonicalRevision uint64
	// Keep the same store -> workspace lock order Stage uses. Only the scalar
	// revision escapes the read callback; the live document remains borrowed.
	if err := store.workspace.Read(func(document *studio.Document) error {
		canonicalRevision = document.Revision
		return nil
	}); err != nil {
		return nil
	}
	store.pruneStaleLocked(canonicalRevision)
	for index := len(store.order) - 1; index >= 0; index-- {
		proposal, ok := store.proposals[store.order[index]]
		if ok && !proposal.Claimed && webMCPProposalOwnerMatches(proposal.Owner, owner) {
			return webMCPProposalView(proposal)
		}
	}
	return nil
}

// Discard revokes a staged proposal without touching the canonical scene.
// This is deliberately session-owned just like Commit: clearing browser DOM
// alone would leave an opaque proposal live on the server until expiry.
func (store *webMCPProposalStore) Discard(proposalID, owner string) (map[string]any, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.pruneLocked(store.now())
	proposal, ok := store.proposals[proposalID]
	if !ok || proposal.Claimed || !webMCPProposalOwnerMatches(proposal.Owner, owner) {
		return nil, errWebMCPProposalNotFound
	}
	delete(store.proposals, proposalID)
	store.removeOrderLocked(proposalID)
	return map[string]any{
		"proposalId":            proposalID,
		"discarded":             true,
		"canonicalSceneChanged": false,
	}, nil
}

func webMCPProposalView(proposal webMCPProposal) map[string]any {
	return map[string]any{
		"proposalId":           proposal.ID,
		"title":                proposal.Title,
		"rationale":            proposal.Rationale,
		"expiresAt":            proposal.ExpiresAt.UTC().Format(time.RFC3339),
		"receipt":              proposal.Receipt,
		"governance":           proposal.Governance,
		"preview":              proposal.Preview,
		"materials":            proposal.Materials,
		"sceneCommands":        proposal.SceneCommands,
		"reverseSceneCommands": proposal.ReverseSceneCommands,
	}
}

func (store *webMCPProposalStore) Commit(proposalID, owner string) (map[string]any, error) {
	proposal, err := store.claimCommit(proposalID, owner)
	if err != nil {
		return nil, err
	}

	transaction := proposal.Transaction
	transaction.ID = "webmcp-commit:" + proposalID
	transaction.Actor = "human://webmcp-review"
	transaction.Mode = studio.ModeDirect
	receipt, document, err := store.workspace.Execute(transaction)
	if err != nil {
		store.finishCommit(proposalID, owner, err)
		return nil, err
	}
	store.finishCommit(proposalID, owner, nil)
	return map[string]any{
		"proposalId": proposalID,
		"title":      proposal.Title,
		"receipt":    receipt,
		"document":   webMCPDocumentSummary(document),
	}, nil
}

// claimCommit atomically makes a proposal unavailable to every other
// lifecycle action before canonical execution begins. In particular, a
// concurrent Discard can no longer report success while this commit applies.
func (store *webMCPProposalStore) claimCommit(proposalID, owner string) (webMCPProposal, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	proposal, ok := store.proposals[proposalID]
	if !ok || proposal.Claimed || !webMCPProposalOwnerMatches(proposal.Owner, owner) {
		return webMCPProposal{}, errWebMCPProposalNotFound
	}
	if !proposal.ExpiresAt.After(store.now()) {
		delete(store.proposals, proposalID)
		store.removeOrderLocked(proposalID)
		return webMCPProposal{}, errWebMCPProposalExpired
	}
	proposal.Claimed = true
	store.proposals[proposalID] = proposal
	return proposal, nil
}

// finishCommit consumes successful and revision-conflicted proposals. A
// transient execution failure releases the claim only if Clear has not
// invalidated the proposal in the meantime, allowing a safe retry without
// resurrecting reset state.
func (store *webMCPProposalStore) finishCommit(proposalID, owner string, executionErr error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	proposal, ok := store.proposals[proposalID]
	if !ok || !proposal.Claimed || !webMCPProposalOwnerMatches(proposal.Owner, owner) {
		return
	}
	if executionErr == nil || errors.Is(executionErr, studio.ErrRevisionConflict) {
		delete(store.proposals, proposalID)
		store.removeOrderLocked(proposalID)
		return
	}
	proposal.Claimed = false
	store.proposals[proposalID] = proposal
}

// Clear invalidates every staged proposal without touching the canonical
// workspace. The store is already capped at maxWebMCPProposals, and replacing
// its map and order makes reset work constant-time regardless of occupancy.
func (store *webMCPProposalStore) Clear() {
	store.mu.Lock()
	store.proposals = map[string]webMCPProposal{}
	store.order = nil
	store.mu.Unlock()
}

func webMCPProposalOwnerMatches(expected, actual string) bool {
	if expected == "" || actual == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(actual)) == 1
}

func (store *webMCPProposalStore) pruneLocked(now time.Time) {
	kept := store.order[:0]
	for _, id := range store.order {
		proposal, ok := store.proposals[id]
		if !ok || (!proposal.Claimed && !proposal.ExpiresAt.After(now)) {
			delete(store.proposals, id)
			continue
		}
		kept = append(kept, id)
	}
	store.order = kept
}

// pruneStaleLocked removes every unclaimed proposal whose exact review base is
// no longer canonical. Claimed proposals belong to an in-flight Commit and are
// left for finishCommit to consume or release; deleting one here would break
// that lifecycle's ownership handshake.
func (store *webMCPProposalStore) pruneStaleLocked(canonicalRevision uint64) {
	kept := store.order[:0]
	for _, id := range store.order {
		proposal, ok := store.proposals[id]
		if !ok || (!proposal.Claimed && proposal.Transaction.ExpectedRevision != canonicalRevision) {
			delete(store.proposals, id)
			continue
		}
		kept = append(kept, id)
	}
	store.order = kept
}

func (store *webMCPProposalStore) hasClaimedOwnerLocked(owner string) bool {
	for _, proposal := range store.proposals {
		if proposal.Claimed && webMCPProposalOwnerMatches(proposal.Owner, owner) {
			return true
		}
	}
	return false
}

func (store *webMCPProposalStore) removeOwnerProposalsLocked(owner string) {
	kept := store.order[:0]
	for _, id := range store.order {
		proposal, ok := store.proposals[id]
		if ok && !proposal.Claimed && webMCPProposalOwnerMatches(proposal.Owner, owner) {
			delete(store.proposals, id)
			continue
		}
		kept = append(kept, id)
	}
	store.order = kept
}

func (store *webMCPProposalStore) makeRoomLocked() error {
	for len(store.order) >= maxWebMCPProposals {
		candidate := ""
		for _, id := range store.order {
			if proposal, ok := store.proposals[id]; ok && !proposal.Claimed {
				candidate = id
				break
			}
		}
		if candidate == "" {
			return fmt.Errorf("WebMCP proposal capacity is busy; try again")
		}
		delete(store.proposals, candidate)
		store.removeOrderLocked(candidate)
	}
	return nil
}

func (store *webMCPProposalStore) removeOrderLocked(id string) {
	for index, candidate := range store.order {
		if candidate != id {
			continue
		}
		store.order = append(store.order[:index], store.order[index+1:]...)
		return
	}
}

func newWebMCPProposalID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generate WebMCP proposal id: %w", err)
	}
	return hex.EncodeToString(value), nil
}

func webMCPDocumentSummary(document studio.Document) map[string]any {
	return map[string]any{
		"id":            document.ID,
		"name":          document.Name,
		"revision":      document.Revision,
		"entityCount":   len(document.Entities),
		"materialCount": len(document.Materials),
	}
}

type webMCPEntityEditState struct {
	Name      string
	Transform studio.Transform
	Material  studio.ID
	HasMesh   bool
}

// webMCPOperationsChangeDocument evaluates the final effect of the deliberately
// narrow WebMCP operation surface. Invalid targets are left to Workspace's
// authoritative validator; this helper only rejects valid requests whose net
// result would be identical to canonical state.
func webMCPOperationsChangeDocument(document *studio.Document, operations []studio.Operation) bool {
	original := make(map[studio.ID]webMCPEntityEditState, len(operations))
	working := make(map[studio.ID]webMCPEntityEditState, len(operations))
	for _, operation := range operations {
		state, seen := working[operation.Target]
		if !seen {
			entity, ok := document.Entities[operation.Target]
			if !ok || entity.Locked {
				return true
			}
			state = webMCPEntityEditState{Name: entity.Name, Transform: entity.Transform, HasMesh: entity.Mesh != nil}
			if entity.Mesh != nil {
				state.Material = entity.Mesh.Material
			}
			original[operation.Target] = state
		}
		switch operation.Kind {
		case studio.OpRenameEntity:
			state.Name = strings.TrimSpace(operation.Name)
		case studio.OpSetTransform:
			if operation.Transform == nil {
				return true
			}
			state.Transform = *operation.Transform
		case studio.OpAssignMaterial:
			if !state.HasMesh {
				return true
			}
			if _, ok := document.Materials[operation.Material]; !ok {
				return true
			}
			state.Material = operation.Material
		default:
			return true
		}
		working[operation.Target] = state
	}
	for id, state := range working {
		if state != original[id] {
			return true
		}
	}
	return false
}

func webMCPMaterialDisplayNames(receipt studio.Receipt, preview studio.Document) map[string]string {
	ids := map[studio.ID]struct{}{}
	for _, change := range receipt.Changes {
		if change.Kind != studio.OpAssignMaterial {
			continue
		}
		if change.Before != nil && change.Before.Mesh != nil {
			ids[change.Before.Mesh.Material] = struct{}{}
		}
		if change.After != nil && change.After.Mesh != nil {
			ids[change.After.Mesh.Material] = struct{}{}
		}
	}
	names := make(map[string]string, len(ids))
	for id := range ids {
		name := string(id)
		if material, ok := preview.Materials[id]; ok && strings.TrimSpace(material.Name) != "" {
			name = material.Name
		}
		names[string(id)] = name
	}
	return names
}
