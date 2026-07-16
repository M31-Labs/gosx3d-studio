package studio

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// ExportLoss is one machine-readable semantic-loss entry in an export report.
// Every export names what the target format cannot carry before bytes are
// written, per the spec's export contract.
type ExportLoss struct {
	Domain string `json:"domain"`
	Reason string `json:"reason"`
	Count  int    `json:"count"`
}

// ExportReport describes an export artifact: identity, size, payload
// fingerprint, and the complete semantic-loss accounting.
type ExportReport struct {
	Kind        string       `json:"kind"`
	Schema      string       `json:"schema"`
	Revision    uint64       `json:"revision"`
	Fingerprint string       `json:"fingerprint"`
	Bytes       int          `json:"bytes"`
	Losses      []ExportLoss `json:"losses"`
}

func exportReport(kind, schema string, document Document, payload []byte, losses []ExportLoss) ExportReport {
	sum := sha256.Sum256(payload)
	if losses == nil {
		losses = []ExportLoss{}
	}
	return ExportReport{Kind: kind, Schema: schema, Revision: document.Revision, Fingerprint: hex.EncodeToString(sum[:]), Bytes: len(payload), Losses: losses}
}

// ExportSceneDoc emits the canonical `.scene3d` document bytes. The canonical
// format is the authoring source of truth, so the loss report is empty by
// definition; a non-empty report here would be a serialization bug.
func ExportSceneDoc(document Document) ([]byte, ExportReport, error) {
	if err := document.Validate(); err != nil {
		return nil, ExportReport{}, err
	}
	payload, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, ExportReport{}, err
	}
	return payload, exportReport("scene3d", SceneDocSchema, document, payload, nil), nil
}

// ExportSceneIR emits the compiled renderer IR plus source map and reports
// every authoring concept the renderer contract does not carry.
func ExportSceneIR(document Document) ([]byte, ExportReport, error) {
	artifact, err := CompileWithSourceMap(document)
	if err != nil {
		return nil, ExportReport{}, err
	}
	payload, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		return nil, ExportReport{}, err
	}
	losses := make([]ExportLoss, 0, 8)
	addLoss := func(domain, reason string, count int) {
		if count > 0 {
			losses = append(losses, ExportLoss{Domain: domain, Reason: reason, Count: count})
		}
	}
	addLoss("prefabs", "prefab definitions and overrides are flattened into resolved runtime nodes", len(document.Prefabs))
	addLoss("rigs", "armatures, bones, constraints, and poses are not part of renderer IR", len(document.Armatures))
	addLoss("animations", "authoring clips and tracks are sampled, not carried", len(document.Animations))
	addLoss("simulations", "simulation profiles and recordings are authoring state", len(document.Simulations))
	addLoss("retargetMaps", "retarget maps are authoring state", len(document.RetargetMaps))
	addLoss("stateMachines", "animation state machines are authoring state", len(document.AnimationMachines))
	renderGraphs := 0
	if document.RenderGraph != nil {
		renderGraphs = 1
	}
	addLoss("renderGraphs", "retained render-graph records lower separately via CompileRenderGraph", renderGraphs)
	addLoss("editorMetadata", "document metadata keys are editor-only", len(document.Metadata))
	modifierCount := 0
	for _, entity := range document.Entities {
		if entity.Mesh != nil {
			modifierCount += len(entity.Mesh.Modifiers)
		}
	}
	addLoss("modifiers", "non-destructive modifier stacks are baked into evaluated geometry", modifierCount)
	return payload, exportReport("scene-ir", "gosx.scene3d.artifact/v1", document, payload, losses), nil
}

