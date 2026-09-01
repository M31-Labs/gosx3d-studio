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

	"m31labs.dev/gosx3d-studio/internal/studio"
)

const (
	maxWebMCPProposalOperations = 12
	maxWebMCPProposals          = 64
	webMCPProposalTTL           = 15 * time.Minute
)

var (
	errWebMCPProposalNotFound = errors.New("WebMCP proposal not found")
	errWebMCPProposalExpired  = errors.New("WebMCP proposal expired")
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
	ExpiresAt   time.Time
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
	receipt, preview, err := store.workspace.Execute(transaction)
	if err != nil {
		return nil, err
	}
	expiresAt := store.now().Add(webMCPProposalTTL)
	previewSummary := webMCPDocumentSummary(preview)
	proposal := webMCPProposal{
		ID: proposalID, Owner: owner, Title: request.Title, Rationale: request.Rationale,
		Transaction: transaction, Receipt: receipt, Governance: governance,
		Preview: previewSummary, ExpiresAt: expiresAt,
	}
	store.pruneLocked(store.now())
	for len(store.order) >= maxWebMCPProposals {
		delete(store.proposals, store.order[0])
		store.order = store.order[1:]
	}
	store.proposals[proposalID] = proposal
	store.order = append(store.order, proposalID)
	return webMCPProposalView(proposal), nil
}

// Current returns the newest unexpired proposal owned by this exact browser
// session. Keeping the opaque proposal ID server-backed lets a human reload or
// restore the tab without losing the review boundary, while a different
// session learns nothing about proposals it does not own.
func (store *webMCPProposalStore) Current(owner string) map[string]any {
	if owner == "" {
		return nil
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.pruneLocked(store.now())
	for index := len(store.order) - 1; index >= 0; index-- {
		proposal, ok := store.proposals[store.order[index]]
		if ok && webMCPProposalOwnerMatches(proposal.Owner, owner) {
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
	if !ok || !webMCPProposalOwnerMatches(proposal.Owner, owner) {
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
		"proposalId": proposal.ID,
		"title":      proposal.Title,
		"rationale":  proposal.Rationale,
		"expiresAt":  proposal.ExpiresAt.UTC().Format(time.RFC3339),
		"receipt":    proposal.Receipt,
		"governance": proposal.Governance,
		"preview":    proposal.Preview,
	}
}

func (store *webMCPProposalStore) Commit(proposalID, owner string) (map[string]any, error) {
	now := store.now()
	store.mu.Lock()
	proposal, ok := store.proposals[proposalID]
	if !ok || !webMCPProposalOwnerMatches(proposal.Owner, owner) {
		store.mu.Unlock()
		return nil, errWebMCPProposalNotFound
	}
	if !proposal.ExpiresAt.After(now) {
		delete(store.proposals, proposalID)
		store.removeOrderLocked(proposalID)
		store.mu.Unlock()
		return nil, errWebMCPProposalExpired
	}
	store.mu.Unlock()

	transaction := proposal.Transaction
	transaction.ID = "webmcp-commit:" + proposalID
	transaction.Actor = "human://webmcp-review"
	transaction.Mode = studio.ModeDirect
	receipt, document, err := store.workspace.Execute(transaction)
	if err != nil {
		if errors.Is(err, studio.ErrRevisionConflict) {
			store.mu.Lock()
			delete(store.proposals, proposalID)
			store.removeOrderLocked(proposalID)
			store.mu.Unlock()
		}
		return nil, err
	}
	store.mu.Lock()
	delete(store.proposals, proposalID)
	store.removeOrderLocked(proposalID)
	store.mu.Unlock()
	return map[string]any{
		"proposalId": proposalID,
		"title":      proposal.Title,
		"receipt":    receipt,
		"document":   webMCPDocumentSummary(document),
	}, nil
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
		if !ok || !proposal.ExpiresAt.After(now) {
			delete(store.proposals, id)
			continue
		}
		kept = append(kept, id)
	}
	store.order = kept
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
