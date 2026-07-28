package studio

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const AssetGCPlanSchema = "gosx3d.studio.asset-gc-plan/v1"

type AssetGCRequest struct {
	Actor            string          `json:"actor"`
	Mode             TransactionMode `json:"mode"`
	ExpectedRevision uint64          `json:"expectedRevision"`
	ConfirmPlan      string          `json:"confirmPlan,omitempty"`
}

type AssetGCPlan struct {
	Schema   string `json:"schema"`
	Revision uint64 `json:"revision"`
	Assets   []ID   `json:"assets"`
	Bytes    int64  `json:"bytes"`
	// Orphans are stored payloads no asset record names. They cannot be
	// reached from the document, so the walk above can never find them: only
	// reading the store does. Undoing an import leaves one, because undo
	// restores the document and the document is not what owns the bytes.
	Orphans     []AssetOrphan `json:"orphans,omitempty"`
	OrphanBytes int64         `json:"orphanBytes,omitempty"`
	Fingerprint string        `json:"fingerprint"`
}

type AssetGCResult struct {
	Plan            AssetGCPlan `json:"plan"`
	Receipt         Receipt     `json:"receipt"`
	Document        Document    `json:"document"`
	Checkpointed    bool        `json:"checkpointed"`
	DeletedPayloads int         `json:"deletedPayloads"`
	DeletedBytes    int64       `json:"deletedBytes"`
	// ReclaimedOrphans counts payloads removed that no record named.
	ReclaimedOrphans int      `json:"reclaimedOrphans,omitempty"`
	ReclaimedBytes   int64    `json:"reclaimedBytes,omitempty"`
	PayloadFindings  []string `json:"payloadFindings,omitempty"`
}

func PlanAssetGarbage(document Document) (AssetGCPlan, error) {
	if err := document.Validate(); err != nil {
		return AssetGCPlan{}, err
	}
	reachable := map[ID]bool{}
	var visit func(ID)
	visit = func(id ID) {
		if reachable[id] {
			return
		}
		asset, ok := document.Assets[id]
		if !ok {
			return
		}
		reachable[id] = true
		for _, dependency := range asset.Dependencies {
			visit(dependency)
		}
	}
	for _, entity := range document.Entities {
		if entity.Model != nil {
			visit(entity.Model.Asset)
		}
	}
	for _, prefab := range document.Prefabs {
		for _, entity := range prefab.Entities {
			if entity.Model != nil {
				visit(entity.Model.Asset)
			}
		}
	}
	for _, material := range document.Materials {
		for _, slot := range material.Textures {
			visit(slot.Asset)
		}
	}
	unused := map[ID]bool{}
	for id := range document.Assets {
		if !reachable[id] {
			unused[id] = true
		}
	}
	ids := make([]ID, 0, len(unused))
	remaining := map[ID]bool{}
	for id := range unused {
		remaining[id] = true
	}
	for len(remaining) > 0 {
		incoming := map[ID]bool{}
		for id := range remaining {
			for _, dependency := range document.Assets[id].Dependencies {
				if remaining[dependency] {
					incoming[dependency] = true
				}
			}
		}
		round := make([]ID, 0)
		for id := range remaining {
			if !incoming[id] {
				round = append(round, id)
			}
		}
		sortIDs(round)
		if len(round) == 0 {
			return AssetGCPlan{}, fmt.Errorf("unused asset dependency graph cannot be ordered")
		}
		for _, id := range round {
			ids = append(ids, id)
			delete(remaining, id)
		}
	}
	plan := AssetGCPlan{Schema: AssetGCPlanSchema, Revision: document.Revision, Assets: ids}
	for _, id := range ids {
		plan.Bytes += document.Assets[id].Bytes
	}
	fingerprintAssetGCPlan(&plan)
	return plan, nil
}

// fingerprintAssetGCPlan stamps the plan with a hash of everything it would
// reclaim. Direct collection requires the caller to echo it back, so the hash
// has to cover the orphan list too: otherwise a confirmed plan could delete a
// set of files the caller never saw.
func fingerprintAssetGCPlan(plan *AssetGCPlan) {
	canonical, _ := json.Marshal(struct {
		Schema      string        `json:"schema"`
		Revision    uint64        `json:"revision"`
		Assets      []ID          `json:"assets"`
		Bytes       int64         `json:"bytes"`
		Orphans     []AssetOrphan `json:"orphans"`
		OrphanBytes int64         `json:"orphanBytes"`
	}{plan.Schema, plan.Revision, plan.Assets, plan.Bytes, plan.Orphans, plan.OrphanBytes})
	sum := sha256.Sum256(canonical)
	plan.Fingerprint = hex.EncodeToString(sum[:])
}

// PlanAssetGarbageForProject returns the plan this workspace would act on:
// the document-derived plan plus the payloads only the filesystem knows about.
// Human and agent surfaces must both read the plan from here, because direct
// collection confirms against this fingerprint.
func (w *Workspace) PlanAssetGarbageForProject() (AssetGCPlan, error) {
	w.mu.RLock()
	projectDir := w.projectDir
	w.mu.RUnlock()
	document, err := w.Snapshot()
	if err != nil {
		return AssetGCPlan{}, err
	}
	return planWithOrphans(document, projectDir, document.Assets)
}

// planWithOrphans is the plan a project actually acts on: the document-derived
// plan plus the payloads only the filesystem knows about. PlanAssetGarbage
// itself stays a pure function of the document, because its determinism is
// what the certification evidence rests on.
func planWithOrphans(document Document, projectDir string, assets map[ID]AssetRecord) (AssetGCPlan, error) {
	plan, err := PlanAssetGarbage(document)
	if err != nil {
		return AssetGCPlan{}, err
	}
	if projectDir == "" {
		return plan, nil
	}
	// An asset this plan is about to delete is still referenced right now, so
	// its payload must not also be listed as an orphan.
	plan.Orphans = findOrphanPayloads(projectDir, assets)
	for _, orphan := range plan.Orphans {
		plan.OrphanBytes += orphan.Bytes
	}
	fingerprintAssetGCPlan(&plan)
	return plan, nil
}

func validateAssetDependencyGraph(assets map[ID]AssetRecord) error {
	state := map[ID]uint8{}
	var visit func(ID) error
	visit = func(id ID) error {
		if state[id] == 1 {
			return fmt.Errorf("asset dependency cycle includes %q", id)
		}
		if state[id] == 2 {
			return nil
		}
		state[id] = 1
		dependencies := append([]ID{}, assets[id].Dependencies...)
		sortIDs(dependencies)
		for _, dependency := range dependencies {
			if err := visit(dependency); err != nil {
				return err
			}
		}
		state[id] = 2
		return nil
	}
	ids := make([]ID, 0, len(assets))
	for id := range assets {
		ids = append(ids, id)
	}
	sortIDs(ids)
	for _, id := range ids {
		if err := visit(id); err != nil {
			return err
		}
	}
	return nil
}

func (w *Workspace) CollectAssetGarbage(request AssetGCRequest) (AssetGCResult, error) {
	document, err := w.Snapshot()
	if err != nil {
		return AssetGCResult{}, err
	}
	if request.ExpectedRevision != document.Revision {
		return AssetGCResult{}, fmt.Errorf("%w: have %d, expected %d", ErrRevisionConflict, document.Revision, request.ExpectedRevision)
	}
	w.mu.RLock()
	planProjectDir := w.projectDir
	w.mu.RUnlock()
	plan, err := planWithOrphans(document, planProjectDir, document.Assets)
	if err != nil {
		return AssetGCResult{}, err
	}
	if request.Mode == ModeDirect && request.ConfirmPlan != plan.Fingerprint {
		return AssetGCResult{}, fmt.Errorf("direct asset collection requires exact confirmPlan fingerprint")
	}
	if request.Mode != ModePropose && request.Mode != ModeDirect {
		return AssetGCResult{}, fmt.Errorf("asset collection mode must be propose or direct")
	}
	if request.Mode == ModeDirect {
		w.mu.RLock()
		persisted := w.projectDir != ""
		w.mu.RUnlock()
		if !persisted {
			return AssetGCResult{}, fmt.Errorf("direct asset collection requires a persisted project")
		}
	}
	operations := make([]Operation, 0, len(plan.Assets))
	for _, id := range plan.Assets {
		operations = append(operations, Operation{Kind: OpDeleteAsset, AssetID: id})
	}
	// A plan may hold nothing but orphans. There is no document change to
	// make for those, so there is no transaction and no checkpoint — deleting
	// a file the document does not reference cannot alter the document.
	if len(operations) == 0 {
		result := AssetGCResult{Plan: plan, Document: document}
		if request.Mode == ModeDirect {
			reclaimOrphans(planProjectDir, plan, &result)
		}
		return result, nil
	}
	receipt, next, err := w.Execute(Transaction{ID: "asset-gc:" + plan.Fingerprint[:16], Actor: request.Actor, Mode: request.Mode, ExpectedRevision: document.Revision, Operations: operations})
	if err != nil {
		return AssetGCResult{}, err
	}
	result := AssetGCResult{Plan: plan, Receipt: receipt, Document: next}
	if request.Mode == ModePropose {
		return result, nil
	}
	w.mu.Lock()
	projectDir := w.projectDir
	w.undo = nil
	w.redo = nil
	w.mu.Unlock()
	result.Checkpointed = true
	for _, id := range plan.Assets {
		asset := document.Assets[id]
		path := filepath.Join(projectDir, filepath.FromSlash(asset.StorePath))
		if err := os.Remove(path); err == nil {
			result.DeletedPayloads++
			result.DeletedBytes += asset.Bytes
		} else if !os.IsNotExist(err) {
			result.PayloadFindings = append(result.PayloadFindings, fmt.Sprintf("%s: %v", id, err))
		}
	}
	reclaimOrphans(projectDir, plan, &result)
	sort.Strings(result.PayloadFindings)
	return result, nil
}

// reclaimOrphans removes the payloads the plan listed as unreferenced. It
// re-derives each path under the project directory rather than trusting the
// plan's string, so a plan can only ever name a file inside the store.
func reclaimOrphans(projectDir string, plan AssetGCPlan, result *AssetGCResult) {
	if projectDir == "" {
		return
	}
	for _, orphan := range plan.Orphans {
		clean := filepath.ToSlash(filepath.Clean(orphan.Path))
		if !strings.HasPrefix(clean, "assets/sha256/") || strings.Contains(clean, "../") {
			result.PayloadFindings = append(result.PayloadFindings, fmt.Sprintf("%s: refused, outside the asset store", orphan.Path))
			continue
		}
		path := filepath.Join(projectDir, filepath.FromSlash(clean))
		if err := os.Remove(path); err == nil {
			result.ReclaimedOrphans++
			result.ReclaimedBytes += orphan.Bytes
		} else if !os.IsNotExist(err) {
			result.PayloadFindings = append(result.PayloadFindings, fmt.Sprintf("%s: %v", orphan.Path, err))
		}
	}
}
