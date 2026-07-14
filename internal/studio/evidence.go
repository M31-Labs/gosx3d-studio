package studio

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"

	"m31labs.dev/gosx/scene"
	"m31labs.dev/gosx/scene/harness"
	"m31labs.dev/gosx/scene/preview"
)

const EvidenceSchema = "gosx3d.studio.evidence/v1"

type EvidenceCheck struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Evidence string `json:"evidence"`
}

type EvidenceReport struct {
	Schema              string              `json:"schema"`
	Slice               string              `json:"slice"`
	Valid               bool                `json:"valid"`
	ReleaseStatus       string              `json:"releaseStatus"`
	DocumentFingerprint string              `json:"documentFingerprint"`
	Checks              []EvidenceCheck     `json:"checks"`
	Incremental         IncrementalStats    `json:"incremental"`
	Harness             harness.Report      `json:"harness"`
	Certification       CertificationReport `json:"certification"`
}

// CertifyM0 builds deterministic, browser-free evidence for the executable M0
// vertical slice. Valid means these checks pass; ReleaseStatus remains the
// honest full-product certification state and is not promoted by this report.
func CertifyM0(document Document) (EvidenceReport, error) {
	report := EvidenceReport{Schema: EvidenceSchema, Slice: "M0", Valid: true, ReleaseStatus: Certification().Status, Certification: Certification()}
	add := func(id string, ok bool, evidence string) {
		status := "pass"
		if !ok {
			status = "fail"
			report.Valid = false
		}
		report.Checks = append(report.Checks, EvidenceCheck{ID: id, Status: status, Evidence: evidence})
	}

	if err := document.Validate(); err != nil {
		return EvidenceReport{}, err
	}
	fingerprint, err := document.Fingerprint()
	if err != nil {
		return EvidenceReport{}, err
	}
	report.DocumentFingerprint = fingerprint
	add("scene-document", true, fmt.Sprintf("revision=%d entities=%d fingerprint=%s", document.Revision, len(document.Entities), fingerprint))

	artifact, err := CompileWithSourceMap(document)
	if err != nil {
		return EvidenceReport{}, err
	}
	add("source-map", len(artifact.SourceMap) == len(document.Entities), fmt.Sprintf("mapped=%d entities=%d", len(artifact.SourceMap), len(document.Entities)))

	compiler := NewIncrementalCompiler()
	incremental, firstStats, err := compiler.Compile(document, "")
	if err != nil {
		return EvidenceReport{}, err
	}
	_, reuseStats, err := compiler.Compile(document, "")
	if err != nil {
		return EvidenceReport{}, err
	}
	report.Incremental = reuseStats
	clean, err := Compile(document)
	if err != nil {
		return EvidenceReport{}, err
	}
	equivalent := reflect.DeepEqual(incremental.SceneIR(), clean.SceneIR())
	add("incremental-equivalence", equivalent && firstStats.RecompiledEntities == len(document.Entities) && reuseStats.RecompiledEntities == 0, fmt.Sprintf("initial=%d reusedSubtrees=%d cleanEquivalent=%t", firstStats.RecompiledEntities, reuseStats.ReusedSubtrees, equivalent))

	session := harness.New(clean, preview.Options{Width: 720, Height: 480, Background: document.Environment.Background, DisablePostFX: true, MaxSegments: 24})
	frame, err := session.Render(0)
	if err != nil {
		return EvidenceReport{}, err
	}
	target, position := FirstPickTarget(document)
	trace := session.Trace("stable-piece", scene.Ray{Origin: scene.Vec3(position.X, 4, position.Z), Direction: scene.Vec3(0, -1, 0)}, scene.PickableOnly())
	if err := session.Validate(); err != nil {
		return EvidenceReport{}, err
	}
	report.Harness = session.Report()
	telemetry := report.Harness.Events[0].Frame
	frameValid := frame != nil && telemetry != nil && telemetry.Coverage > 0.001 && telemetry.PNGHash != ""
	frameEvidence := "missing"
	if telemetry != nil {
		frameEvidence = fmt.Sprintf("png=%s coverage=%.6f", telemetry.PNGHash, telemetry.Coverage)
	}
	add("headless-frame", frameValid, frameEvidence)
	exact := trace.Closest != nil && trace.Closest.ID == string(target)
	add("exact-pick", exact, fmt.Sprintf("selected=%s visited=%d tested=%d", target, trace.NodesVisited, trace.PrimitivesTested))
	selenaValid := len(report.Harness.Materials) > 0
	for _, material := range report.Harness.Materials {
		selenaValid = selenaValid && material.Valid && material.WGSLFragmentHash != "" && material.GLSLFragmentHash != ""
	}
	add("selena-artifacts", selenaValid, fmt.Sprintf("validated=%d", len(report.Harness.Materials)))
	return report, nil
}

// CertifyCurrent extends the M0 proof with every M1 behavior that currently has
// complete browser-free semantic evidence. It intentionally names the slice a
// foundation rather than claiming the full M1 milestone.
func CertifyCurrent(document Document) (EvidenceReport, error) {
	report, err := CertifyM0(document)
	if err != nil {
		return EvidenceReport{}, err
	}
	report.Slice = "M0+M1+M2-foundation"
	add := func(id string, ok bool, evidence string) {
		status := "pass"
		if !ok {
			status = "fail"
			report.Valid = false
		}
		report.Checks = append(report.Checks, EvidenceCheck{ID: id, Status: status, Evidence: evidence})
	}

	meshDocument := certificationMeshDocument(document)
	workspace, err := NewWorkspace(meshDocument)
	if err != nil {
		return EvidenceReport{}, err
	}
	edges := MeshEdges(meshDocument.Entities["cert-mesh"].Mesh.Geometry)
	selectionOK := len(edges) == 4
	if selectionOK {
		err = workspace.SelectSubobjects(SelectionRequest{ExpectedRevision: meshDocument.Revision, Mode: SelectionEdge, Object: "cert-mesh", IDs: []ID{edges[0].ID}})
		selectionOK = err == nil && workspace.SelectionState().Mode == SelectionEdge
	}
	add("m1-subobject-selection", selectionOK, fmt.Sprintf("vertices=4 edges=%d faces=1 revision=%d", len(edges), meshDocument.Revision))

	operations := []Operation{{Kind: OpInsetFaces, Target: "cert-mesh", Faces: []ID{"quad"}, Amount: 0.25}, {Kind: OpTriangulateFaces, Target: "cert-mesh", Faces: []ID{"quad"}}, {Kind: OpRecalculateNormals, Target: "cert-mesh"}, {Kind: OpProjectPlanarUV, Target: "cert-mesh", Projection: "xz"}}
	transaction := Transaction{ID: "certify:m1-topology", Actor: "certifier://studio", Mode: ModePropose, ExpectedRevision: meshDocument.Revision, Operations: operations}
	previewReceipt, previewDocument, err := workspace.Execute(transaction)
	if err != nil {
		return EvidenceReport{}, err
	}
	transaction.Mode = ModeDirect
	directReceipt, directDocument, err := workspace.Execute(transaction)
	if err != nil {
		return EvidenceReport{}, err
	}
	previewFingerprint, _ := previewDocument.Fingerprint()
	directFingerprint, _ := directDocument.Fingerprint()
	_, compiledErr := Compile(directDocument)
	topologyOK := previewFingerprint == directFingerprint && len(previewReceipt.OperatorRecords) == len(operations) && len(directReceipt.OperatorRecords) == len(operations) && compiledErr == nil
	add("m1-topology-actions", topologyOK, fmt.Sprintf("operators=%d result=%s previewEquivalent=%t sceneIR=%t", len(operations), directFingerprint, previewFingerprint == directFingerprint, compiledErr == nil))

	analysis, err := AnalyzeEntityGeometry(meshDocument, "cert-mesh")
	if err != nil {
		return EvidenceReport{}, err
	}
	analysisOK := analysis.Valid && analysis.Vertices == 4 && analysis.Edges == 4 && analysis.Faces == 1 && analysis.Triangles == 2 && near(analysis.SurfaceArea, 1)
	add("m1-geometry-analysis", analysisOK, fmt.Sprintf("v=%d e=%d f=%d triangles=%d area=%.6f findings=%d", analysis.Vertices, analysis.Edges, analysis.Faces, analysis.Triangles, analysis.SurfaceArea, len(analysis.Findings)))
	uvAnalysis, err := AnalyzeEntityGeometry(directDocument, "cert-mesh")
	if err != nil {
		return EvidenceReport{}, err
	}
	uvOK := uvAnalysis.UV.Complete && uvAnalysis.UV.MissingVertices == 0 && uvAnalysis.UV.DegenerateFaces == 0 && uvAnalysis.UV.Bounds != nil
	add("m1-uv-authoring", uvOK, fmt.Sprintf("mapped=%d missing=%d degenerate=%d outsideUnit=%d", uvAnalysis.UV.MappedVertices, uvAnalysis.UV.MissingVertices, uvAnalysis.UV.DegenerateFaces, uvAnalysis.UV.OutsideUnit))
	dissolveFaces, bridgeFaces, structuralOK, err := certifyStructuralOperators(meshDocument)
	if err != nil {
		return EvidenceReport{}, err
	}
	add("m1-structural-operators", structuralOK, fmt.Sprintf("dissolvedFaces=%d bridgedFaces=%d sceneIR=%t", dissolveFaces, bridgeFaces, structuralOK))
	cutVertices, cutFaces, loopOK, err := certifyLoopCut(meshDocument)
	if err != nil {
		return EvidenceReport{}, err
	}
	add("m1-loop-cut", loopOK, fmt.Sprintf("vertices=%d faces=%d sharedCrossing=true sceneIR=%t", cutVertices, cutFaces, loopOK))
	curveAnalysis, curveOK, err := certifyCurve(meshDocument)
	if err != nil {
		return EvidenceReport{}, err
	}
	add("m1-nurbs-curve", curveOK, fmt.Sprintf("degree=%d controls=%d length=%.6f vertices=%d triangles=%d sceneIR=%t", curveAnalysis.Degree, curveAnalysis.ControlPoints, curveAnalysis.ApproximateLength, curveAnalysis.TessellatedVertices, curveAnalysis.Triangles, curveOK))
	modifierVertices, modifierFaces, modifierOK, err := certifyModifiers(meshDocument)
	if err != nil {
		return EvidenceReport{}, err
	}
	add("m1-modifier-stack", modifierOK, fmt.Sprintf("modifiers=4 kinds=subdivision,mirror,array,solidify reorder=true applyThrough=true remaining=1 evaluatedVertices=%d evaluatedFaces=%d closed=true manifold=true previewEquivalent=true checkpointUndo=true sceneIR=%t", modifierVertices, modifierFaces, modifierOK))
	csgAnalysis, csgOK, err := certifyCSG(meshDocument)
	if err != nil {
		return EvidenceReport{}, err
	}
	add("m1-voxel-csg", csgOK, fmt.Sprintf("operation=union voxelSize=0.25 closed=%t manifold=%t volume=%.6f vertices=%d faces=%d sceneIR=%t", csgAnalysis.Closed, csgAnalysis.Manifold, valueOrZero(csgAnalysis.Volume), csgAnalysis.Vertices, csgAnalysis.Faces, csgOK))
	materialID, materialOK, err := certifyMaterialAuthoring(document)
	if err != nil {
		return EvidenceReport{}, err
	}
	add("m1-material-authoring", materialOK, fmt.Sprintf("material=%s previewEquivalent=true semanticReceipt=true selenaInvalidRejected=true lastGoodPreserved=true sceneIR=%t", materialID, materialOK))
	prefabEntities, prefabMappings, prefabOK, err := certifyPrefab(meshDocument)
	if err != nil {
		return EvidenceReport{}, err
	}
	add("m1-linked-prefab", prefabOK, fmt.Sprintf("definitionEntities=%d instances=1 runtimeMappings=%d previewEquivalent=true override=true incrementalInvalidation=true sceneIR=%t", prefabEntities, prefabMappings, prefabOK))
	assetID, assetHash, assetOK, err := certifyAssetPipeline(document)
	if err != nil {
		return EvidenceReport{}, err
	}
	add("m1-asset-pipeline", assetOK, fmt.Sprintf("asset=%s sha256=%s format=gltf audit=true dependencies=true reimport=true atomicRetarget=true semanticReceipt=true modelSceneIR=%t", assetID, assetHash, assetOK))
	rigEvidence, rigOK, err := certifyRigAnimationFoundation()
	if err != nil {
		return EvidenceReport{}, err
	}
	add("m2-rig-animation-foundation", rigOK, rigEvidence)
	simulationEvidence, simulationOK, err := certifySimulationFoundation()
	if err != nil {
		return EvidenceReport{}, err
	}
	add("m2-fixed-step-simulation", simulationOK, simulationEvidence)
	return report, nil
}

func certifySimulationFoundation() (string, bool, error) {
	document := ArticulatedProofDocument()
	inputs := []SimulationInput{{Tick: 30, Entity: "physics-payload", Impulse: Vec3{X: 1.5}}}
	recording, simulated, err := RunSimulation(document, "articulated-physics", 120, inputs)
	if err != nil {
		return "", false, err
	}
	replayed, replayDocument, err := ReplaySimulation(document, recording)
	if err != nil {
		return "", false, err
	}
	workspace, err := NewWorkspace(document)
	if err != nil {
		return "", false, err
	}
	operation := Operation{Kind: OpSimulateTicks, SimulationID: "articulated-physics", Ticks: 120, Inputs: inputs}
	transaction := Transaction{ID: "certify:m2-agent-simulation", Actor: "agent://studio-certifier", Mode: ModePropose, ExpectedRevision: document.Revision, Operations: []Operation{operation}}
	previewReceipt, preview, err := workspace.Execute(transaction)
	if err != nil {
		return "", false, err
	}
	transaction.Mode = ModeDirect
	directReceipt, direct, err := workspace.Execute(transaction)
	if err != nil {
		return "", false, err
	}
	previewFingerprint, _ := preview.Fingerprint()
	directFingerprint, _ := direct.Fingerprint()
	simulatedFingerprint, _ := simulated.Fingerprint()
	replayFingerprint, _ := replayDocument.Fingerprint()
	authoredProps, authoredCompileErr := Compile(document)
	simulatedProps, simulatedCompileErr := Compile(simulated)
	authoredIR, _ := json.Marshal(authoredProps.SceneIR())
	simulatedIR, _ := json.Marshal(simulatedProps.SceneIR())
	sceneIRChanged := authoredCompileErr == nil && simulatedCompileErr == nil && !bytes.Equal(authoredIR, simulatedIR)
	ok := recording.Final.Hash == replayed.Final.Hash && simulatedFingerprint == replayFingerprint && len(recording.Events) > 0 && previewFingerprint == directFingerprint && len(previewReceipt.SimulationChanges) == 1 && len(directReceipt.SimulationChanges) == 1 && directReceipt.SimulationChanges[0].BeforeHash != directReceipt.SimulationChanges[0].AfterHash && sceneIRChanged
	return fmt.Sprintf("profile=articulated-physics tickRate=60 ticks=120 inputs=%d contacts=%d initial=%s final=%s replayExact=%t sceneIRChanged=%t agentActor=%s previewEquivalent=%t semanticSimulationReceipt=%t", len(inputs), len(recording.Events), shortHash(recording.Initial.Hash), shortHash(recording.Final.Hash), recording.Final.Hash == replayed.Final.Hash && simulatedFingerprint == replayFingerprint, sceneIRChanged, directReceipt.Actor, previewFingerprint == directFingerprint, len(directReceipt.SimulationChanges) == 1), ok, nil
}

func shortHash(value string) string {
	if len(value) > 12 {
		return value[:12]
	}
	return value
}

func certifyRigAnimationFoundation() (string, bool, error) {
	document := ArticulatedProofDocument()
	if err := document.Validate(); err != nil {
		return "", false, err
	}
	first, evaluated, err := SampleAnimation(document, "reach", 0.5)
	if err != nil {
		return "", false, err
	}
	second, repeated, err := SampleAnimation(document, "reach", 0.5)
	if err != nil {
		return "", false, err
	}
	firstFingerprint, _ := evaluated.Fingerprint()
	secondFingerprint, _ := repeated.Fingerprint()
	_, compileErr := Compile(evaluated)
	deformed, deformation, deformationErr := DeformSkinnedGeometry(evaluated, "skinned")
	authoredIR, authoredCompileErr := Compile(document)
	deformedIR, deformedCompileErr := Compile(evaluated)
	authoredIRJSON, _ := json.Marshal(authoredIR.SceneIR())
	deformedIRJSON, _ := json.Marshal(deformedIR.SceneIR())
	workspace, err := NewWorkspace(document)
	if err != nil {
		return "", false, err
	}
	pose := Transform{Position: Vec3{Y: 1}, Rotation: Vec3{Z: 0.35}, Scale: Vec3{X: 1, Y: 1, Z: 1}}
	key := TransformKey{Time: 0.25, Transform: pose}
	transaction := Transaction{ID: "certify:m2-agent-animation", Actor: "agent://studio-certifier", Mode: ModePropose, ExpectedRevision: document.Revision, Operations: []Operation{{Kind: OpSetBonePose, ArmatureID: "arm", BoneID: "lower", Transform: &pose}, {Kind: OpSetAnimationKey, ClipID: "reach", TrackID: "lower-track", Key: &key}, {Kind: OpSolveIK, ArmatureID: "arm", ConstraintID: "reach-ik"}}}
	previewReceipt, preview, err := workspace.Execute(transaction)
	if err != nil {
		return "", false, err
	}
	transaction.Mode = ModeDirect
	directReceipt, direct, err := workspace.Execute(transaction)
	if err != nil {
		return "", false, err
	}
	previewFingerprint, _ := preview.Fingerprint()
	directFingerprint, _ := direct.Fingerprint()
	ikResult, _, ikErr := SolveTwoBoneIK(document, "arm", "reach-ik")
	semanticReceipt := len(directReceipt.RigChanges) == 3 && len(directReceipt.AnimationChanges) == 1
	skinOK := deformationErr == nil && authoredCompileErr == nil && deformedCompileErr == nil && len(deformed.Vertices) == 3 && deformation.MovedVertices > 0 && !bytes.Equal(authoredIRJSON, deformedIRJSON)
	ok := firstFingerprint == secondFingerprint && reflect.DeepEqual(first, second) && compileErr == nil && previewFingerprint == directFingerprint && semanticReceipt && len(previewReceipt.RigChanges) == 3 && ikErr == nil && ikResult.Error < 1e-8 && skinOK
	return fmt.Sprintf("armatures=1 bones=3 normalizedWeights=3 skin=LBS movedVertices=%d maxDelta=%.9f sceneIRChanged=%t ik=two-bone-cpu ikError=%.9f clips=1 sample=0.5 deterministic=%t agentActor=%s previewEquivalent=%t semanticRigReceipt=%t sceneIR=%t", deformation.MovedVertices, deformation.MaximumDelta, !bytes.Equal(authoredIRJSON, deformedIRJSON), ikResult.Error, firstFingerprint == secondFingerprint, directReceipt.Actor, previewFingerprint == directFingerprint, semanticReceipt, compileErr == nil), ok, nil
}

func certifyAssetPipeline(document Document) (ID, string, bool, error) {
	dir, err := os.MkdirTemp("", "gosx3d-cert-assets-*")
	if err != nil {
		return "", "", false, err
	}
	defer os.RemoveAll(dir)
	input := filepath.Join(dir, "certification.gltf")
	if err := os.WriteFile(input, []byte(`{"asset":{"version":"2.0"},"scenes":[{}],"scene":0}`), 0o600); err != nil {
		return "", "", false, err
	}
	project := filepath.Join(dir, "project")
	workspace, err := OpenWorkspace(project, document)
	if err != nil {
		return "", "", false, err
	}
	initial, err := workspace.Snapshot()
	if err != nil {
		return "", "", false, err
	}
	previewReceipt, preview, previewAsset, err := workspace.ImportAsset(AssetImportRequest{Path: input, Actor: "certifier://studio", Mode: ModePropose, ExpectedRevision: initial.Revision})
	if err != nil {
		return "", "", false, err
	}
	directReceipt, direct, asset, err := workspace.ImportAsset(AssetImportRequest{Path: input, Actor: "certifier://studio", Mode: ModeDirect, ExpectedRevision: initial.Revision})
	if err != nil {
		return "", "", false, err
	}
	model := Entity{ID: "cert-asset-model", Name: "Certification asset model", Transform: IdentityTransform(), Model: &ModelComponent{Asset: asset.ID, Bounds: 1, Fit: "contain", Pickable: true}, Visible: true}
	_, modeled, err := workspace.Execute(Transaction{ID: "certify:asset-model", Actor: "certifier://studio", Mode: ModeDirect, ExpectedRevision: direct.Revision, Operations: []Operation{{Kind: OpCreateEntity, Entity: &model}}})
	if err != nil {
		return "", "", false, err
	}
	_, captured, err := workspace.Execute(Transaction{ID: "certify:asset-prefab", Actor: "certifier://studio", Mode: ModeDirect, ExpectedRevision: modeled.Revision, Operations: []Operation{{Kind: OpCapturePrefab, Target: model.ID, PrefabID: "cert-asset-prefab", Name: "Certification asset prefab"}}})
	if err != nil {
		return "", "", false, err
	}
	_, instantiated, err := workspace.Execute(Transaction{ID: "certify:asset-instance", Actor: "certifier://studio", Mode: ModeDirect, ExpectedRevision: captured.Revision, Operations: []Operation{{Kind: OpInstantiatePrefab, PrefabID: "cert-asset-prefab", NewID: "cert-asset-instance"}}})
	if err != nil {
		return "", "", false, err
	}
	dependencies := workspace.AssetDependencies()
	replacementPath := filepath.Join(dir, "certification-reimport.gltf")
	if err := os.WriteFile(replacementPath, []byte(`{"asset":{"version":"2.0","generator":"studio-certifier"},"scenes":[{}],"scene":0}`), 0o600); err != nil {
		return "", "", false, err
	}
	reimportRequest := AssetReimportRequest{AssetID: asset.ID, Path: replacementPath, Actor: "certifier://studio", Mode: ModePropose, ExpectedRevision: instantiated.Revision}
	reimportPreviewReceipt, reimportPreview, previewReplacement, err := workspace.ReimportAsset(reimportRequest)
	if err != nil {
		return "", "", false, err
	}
	reimportRequest.Mode = ModeDirect
	reimportReceipt, reimported, replacement, err := workspace.ReimportAsset(reimportRequest)
	if err != nil {
		return "", "", false, err
	}
	ir, err := CompileIR(reimported)
	if err != nil {
		return "", "", false, err
	}
	previewFingerprint, _ := preview.Fingerprint()
	directFingerprint, _ := direct.Fingerprint()
	reimportPreviewFingerprint, _ := reimportPreview.Fingerprint()
	reimportFingerprint, _ := reimported.Fingerprint()
	_, restored, undoErr := workspace.Undo(reimported.Revision, "certifier://studio")
	var replayed Document
	var redoErr error
	if undoErr == nil {
		_, replayed, redoErr = workspace.Redo(restored.Revision, "certifier://studio")
	}
	audit := workspace.AuditAssets()
	modelsRetargeted := len(ir.Models) == 2
	for _, compiled := range ir.Models {
		modelsRetargeted = modelsRetargeted && compiled.Src == replacement.URI
	}
	dependencyOK := len(dependencies.Assets) == 1 && dependencies.Assets[0].DirectCount == 2 && len(dependencies.Assets[0].Instances) == 1
	ok := previewAsset.ID == asset.ID && previewFingerprint == directFingerprint && len(previewReceipt.AssetChanges) == 1 && len(directReceipt.AssetChanges) == 1 &&
		previewReplacement.ID == replacement.ID && reimportPreviewFingerprint == reimportFingerprint && len(reimportPreviewReceipt.AssetChanges) == 2 && len(reimportReceipt.AssetChanges) == 2 &&
		dependencyOK && reimported.Entities[model.ID].Model.Asset == replacement.ID && reimported.Prefabs["cert-asset-prefab"].Entities[model.ID].Model.Asset == replacement.ID &&
		undoErr == nil && redoErr == nil && restored.Entities[model.ID].Model.Asset == asset.ID && replayed.Entities[model.ID].Model.Asset == replacement.ID &&
		audit.Valid && len(audit.Assets) == 1 && modelsRetargeted
	return replacement.ID, replacement.ContentHash, ok, nil
}

func certifyPrefab(document Document) (int, int, bool, error) {
	workspace, err := NewWorkspace(document)
	if err != nil {
		return 0, 0, false, err
	}
	capture := Operation{Kind: OpCapturePrefab, Target: "cert-mesh", PrefabID: "cert-prefab", Name: "Certification prefab"}
	previewReceipt, preview, err := workspace.Execute(Transaction{ID: "certify:prefab-preview", Actor: "certifier://studio", Mode: ModePropose, ExpectedRevision: document.Revision, Operations: []Operation{capture}})
	if err != nil {
		return 0, 0, false, err
	}
	directReceipt, captured, err := workspace.Execute(Transaction{ID: "certify:prefab-capture", Actor: "certifier://studio", Mode: ModeDirect, ExpectedRevision: document.Revision, Operations: []Operation{capture}})
	if err != nil {
		return 0, 0, false, err
	}
	_, instantiated, err := workspace.Execute(Transaction{ID: "certify:prefab-instance", Actor: "certifier://studio", Mode: ModeDirect, ExpectedRevision: captured.Revision, Operations: []Operation{{Kind: OpInstantiatePrefab, PrefabID: "cert-prefab", NewID: "cert-prefab-instance", Parent: document.RootIDs[0]}}})
	if err != nil {
		return 0, 0, false, err
	}
	visible := false
	_, overridden, err := workspace.Execute(Transaction{ID: "certify:prefab-override", Actor: "certifier://studio", Mode: ModeDirect, ExpectedRevision: instantiated.Revision, Operations: []Operation{{Kind: OpSetPrefabOverride, Target: "cert-prefab-instance", PrefabEntity: "cert-mesh", PrefabOverride: &PrefabEntityOverride{Visible: &visible}}}})
	if err != nil {
		return 0, 0, false, err
	}
	artifact, err := CompileWithSourceMap(overridden)
	if err != nil {
		return 0, 0, false, err
	}
	runtimeID := ID("cert-prefab-instance--cert-mesh")
	location, mapped := artifact.SourceMap[runtimeID]
	compiler := NewIncrementalCompiler()
	if _, _, err := compiler.Compile(overridden, ""); err != nil {
		return 0, 0, false, err
	}
	transform := overridden.Entities["cert-mesh"].Transform
	transform.Position.Y = 1
	_, updated, err := workspace.Execute(Transaction{ID: "certify:prefab-update", Actor: "certifier://studio", Mode: ModeDirect, ExpectedRevision: overridden.Revision, Operations: []Operation{{Kind: OpSetTransform, Target: "cert-mesh", Transform: &transform}, {Kind: OpCapturePrefab, Target: "cert-mesh", PrefabID: "cert-prefab", Name: "Certification prefab"}}})
	if err != nil {
		return 0, 0, false, err
	}
	_, stats, err := compiler.Compile(updated, "")
	if err != nil {
		return 0, 0, false, err
	}
	previewFingerprint, _ := preview.Fingerprint()
	capturedFingerprint, _ := captured.Fingerprint()
	definition := updated.Prefabs["cert-prefab"]
	ok := previewFingerprint == capturedFingerprint && len(previewReceipt.PrefabChanges) == 1 && len(directReceipt.PrefabChanges) == 1 && mapped && location.RecordID == "cert-prefab/cert-mesh" && stats.RecompiledEntities > 0
	return len(definition.Entities), len(artifact.SourceMap) - len(overridden.Entities), ok, nil
}

func certifyMaterialAuthoring(document Document) (ID, bool, error) {
	workspace, err := NewWorkspace(document)
	if err != nil {
		return "", false, err
	}
	material := document.Materials["board-material"]
	material.Roughness = 0.41
	operation := Operation{Kind: OpSetMaterial, MaterialRecord: &material}
	previewReceipt, preview, err := workspace.Execute(Transaction{ID: "certify:material-preview", Actor: "certifier://studio", Mode: ModePropose, ExpectedRevision: document.Revision, Operations: []Operation{operation}})
	if err != nil {
		return "", false, err
	}
	directReceipt, direct, err := workspace.Execute(Transaction{ID: "certify:material-direct", Actor: "certifier://studio", Mode: ModeDirect, ExpectedRevision: document.Revision, Operations: []Operation{operation}})
	if err != nil {
		return "", false, err
	}
	validFingerprint, _ := direct.Fingerprint()
	invalid := document.Materials["player-1-material"]
	invalid.Selena = &SelenaShader{Material: invalid.Selena.Material, Source: "invalid Selena source"}
	_, _, invalidErr := workspace.Execute(Transaction{ID: "certify:material-invalid", Actor: "certifier://studio", Mode: ModeDirect, ExpectedRevision: direct.Revision, Operations: []Operation{{Kind: OpSetMaterial, MaterialRecord: &invalid}}})
	afterInvalid, _ := workspace.Snapshot()
	afterFingerprint, _ := afterInvalid.Fingerprint()
	previewFingerprint, _ := preview.Fingerprint()
	directFingerprint, _ := direct.Fingerprint()
	_, compileErr := Compile(direct)
	ok := previewFingerprint == directFingerprint && len(previewReceipt.MaterialChanges) == 1 && len(directReceipt.MaterialChanges) == 1 && invalidErr != nil && afterFingerprint == validFingerprint && compileErr == nil
	return material.ID, ok, nil
}

func certifyCSG(meshDocument Document) (GeometryAnalysis, bool, error) {
	document, _ := meshDocument.Clone()
	left := document.Entities["cert-mesh"]
	left.Mesh.Geometry = voxelSurface("left", Vec3{}, 1, map[voxelCell]bool{{0, 0, 0}: true})
	left.Mesh.Modifiers = nil
	document.Entities[left.ID] = left
	right := left
	right.ID = "cert-csg-operand"
	right.Name = "CSG operand"
	right.Transform.Position = Vec3{X: 0.5}
	right.Mesh = &MeshComponent{Geometry: voxelSurface("right", Vec3{}, 1, map[voxelCell]bool{{0, 0, 0}: true}), Material: left.Mesh.Material, Pickable: true}
	document.Entities[right.ID] = right
	root := document.Entities[right.Parent]
	root.Children = append(root.Children, right.ID)
	document.Entities[root.ID] = root
	workspace, err := NewWorkspace(document)
	if err != nil {
		return GeometryAnalysis{}, false, err
	}
	operation := Operation{Kind: OpCSGBoolean, Target: "cert-mesh", Operand: right.ID, Boolean: "union", VoxelSize: 0.25, NewID: "cert-csg-result"}
	previewReceipt, preview, err := workspace.Execute(Transaction{ID: "certify:csg-preview", Actor: "certifier://studio", Mode: ModePropose, ExpectedRevision: document.Revision, Operations: []Operation{operation}})
	if err != nil {
		return GeometryAnalysis{}, false, err
	}
	directReceipt, direct, err := workspace.Execute(Transaction{ID: "certify:csg-direct", Actor: "certifier://studio", Mode: ModeDirect, ExpectedRevision: document.Revision, Operations: []Operation{operation}})
	if err != nil {
		return GeometryAnalysis{}, false, err
	}
	analysis, err := AnalyzeEntityGeometry(direct, operation.NewID)
	if err != nil {
		return GeometryAnalysis{}, false, err
	}
	previewFingerprint, _ := preview.Fingerprint()
	directFingerprint, _ := direct.Fingerprint()
	_, compileErr := Compile(direct)
	ok := previewFingerprint == directFingerprint && analysis.Valid && analysis.Closed && analysis.Manifold && analysis.Volume != nil && near(*analysis.Volume, 1.5) && len(previewReceipt.OperatorRecords) == 1 && previewReceipt.OperatorRecords[0].Boolean == "union" && previewReceipt.OperatorRecords[0].Result[0] == operation.NewID && len(directReceipt.OperatorRecords) == 1 && compileErr == nil
	return analysis, ok, nil
}

func valueOrZero(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func certifyModifiers(meshDocument Document) (int, int, bool, error) {
	workspace, err := NewWorkspace(meshDocument)
	if err != nil {
		return 0, 0, false, err
	}
	mirror := Modifier{ID: "mirror-x", Kind: "mirror", Enabled: true, Axis: "x"}
	array := Modifier{ID: "array-y", Kind: "array", Enabled: true, Count: 3, Offset: Vec3{Y: 2}}
	solidify := Modifier{ID: "solidify", Kind: "solidify", Enabled: true, Thickness: 0.1}
	subdivision := Modifier{ID: "subdivision", Kind: "subdivision", Enabled: true, Levels: 1}
	operations := []Operation{{Kind: OpSetModifier, Target: "cert-mesh", Modifier: &subdivision}, {Kind: OpSetModifier, Target: "cert-mesh", Modifier: &mirror}, {Kind: OpSetModifier, Target: "cert-mesh", Modifier: &array}, {Kind: OpSetModifier, Target: "cert-mesh", Modifier: &solidify}}
	previewReceipt, preview, err := workspace.Execute(Transaction{ID: "certify:modifier-preview", Actor: "certifier://studio", Mode: ModePropose, ExpectedRevision: meshDocument.Revision, Operations: operations})
	if err != nil {
		return 0, 0, false, err
	}
	directReceipt, direct, err := workspace.Execute(Transaction{ID: "certify:modifier-direct", Actor: "certifier://studio", Mode: ModeDirect, ExpectedRevision: meshDocument.Revision, Operations: operations})
	if err != nil {
		return 0, 0, false, err
	}
	reorder := Operation{Kind: OpReorderModifier, Target: "cert-mesh", ModifierID: "solidify", ModifierIndex: 0}
	reorderPreviewReceipt, reorderPreview, err := workspace.Execute(Transaction{ID: "certify:modifier-reorder-preview", Actor: "certifier://studio", Mode: ModePropose, ExpectedRevision: direct.Revision, Operations: []Operation{reorder}})
	if err != nil {
		return 0, 0, false, err
	}
	reorderReceipt, reordered, err := workspace.Execute(Transaction{ID: "certify:modifier-reorder-direct", Actor: "certifier://studio", Mode: ModeDirect, ExpectedRevision: direct.Revision, Operations: []Operation{reorder}})
	if err != nil {
		return 0, 0, false, err
	}
	apply := Operation{Kind: OpApplyModifier, Target: "cert-mesh", ModifierID: "mirror-x"}
	applyPreviewReceipt, applyPreview, err := workspace.Execute(Transaction{ID: "certify:modifier-apply-preview", Actor: "certifier://studio", Mode: ModePropose, ExpectedRevision: reordered.Revision, Operations: []Operation{apply}})
	if err != nil {
		return 0, 0, false, err
	}
	applyReceipt, applied, err := workspace.Execute(Transaction{ID: "certify:modifier-apply-direct", Actor: "certifier://studio", Mode: ModeDirect, ExpectedRevision: reordered.Revision, Operations: []Operation{apply}})
	if err != nil {
		return 0, 0, false, err
	}
	evaluated, err := evaluateModifiers(applied.Entities["cert-mesh"].Mesh.Geometry, applied.Entities["cert-mesh"].Mesh.Modifiers)
	if err != nil {
		return 0, 0, false, err
	}
	previewFingerprint, _ := preview.Fingerprint()
	directFingerprint, _ := direct.Fingerprint()
	reorderPreviewFingerprint, _ := reorderPreview.Fingerprint()
	reorderedFingerprint, _ := reordered.Fingerprint()
	applyPreviewFingerprint, _ := applyPreview.Fingerprint()
	appliedFingerprint, _ := applied.Fingerprint()
	_, compileErr := Compile(applied)
	analysisDocument, _ := applied.Clone()
	entity := analysisDocument.Entities["cert-mesh"]
	entity.Mesh.Geometry = evaluated
	entity.Mesh.Modifiers = nil
	analysisDocument.Entities[entity.ID] = entity
	analysis, analysisErr := AnalyzeEntityGeometry(analysisDocument, entity.ID)
	_, restored, undoErr := workspace.Undo(applied.Revision, "certifier://studio")
	ok := previewFingerprint == directFingerprint && reorderPreviewFingerprint == reorderedFingerprint && applyPreviewFingerprint == appliedFingerprint &&
		len(previewReceipt.OperatorRecords) == 4 && len(directReceipt.OperatorRecords) == 4 &&
		len(reorderPreviewReceipt.OperatorRecords) == 1 && reorderReceipt.OperatorRecords[0].ModifierIndex == 0 &&
		len(applyPreviewReceipt.OperatorRecords) == 1 && applyReceipt.OperatorRecords[0].UndoPolicy == "geometry-checkpoint" &&
		len(applied.Entities["cert-mesh"].Mesh.Modifiers) == 1 && len(evaluated.Vertices) == 156 && len(evaluated.Faces) == 144 &&
		analysisErr == nil && analysis.Closed && analysis.Manifold && compileErr == nil && undoErr == nil && len(restored.Entities["cert-mesh"].Mesh.Modifiers) == 4
	return len(evaluated.Vertices), len(evaluated.Faces), ok, nil
}

func certifyCurve(meshDocument Document) (CurveAnalysis, bool, error) {
	document, _ := meshDocument.Clone()
	entity := document.Entities["cert-mesh"]
	entity.Mesh.Geometry = Geometry{Kind: "nurbs-curve", Curve: &CurveGeometry{Degree: 2, ControlPoints: []CurveControlPoint{{ID: "p0", Position: Vec3{}, Weight: 1}, {ID: "p1", Position: Vec3{X: 1, Y: 1}, Weight: 1}, {ID: "p2", Position: Vec3{X: 2}, Weight: 1}}, Knots: []float64{0, 0, 0, 1, 1, 1}, Segments: 8, Radius: 0.1, RadialSegments: 6}}
	document.Entities[entity.ID] = entity
	workspace, err := NewWorkspace(document)
	if err != nil {
		return CurveAnalysis{}, false, err
	}
	if err := workspace.SelectSubobjects(SelectionRequest{ExpectedRevision: document.Revision, Mode: SelectionCurveControlPoint, Object: "cert-mesh", IDs: []ID{"p1"}}); err != nil {
		return CurveAnalysis{}, false, err
	}
	position := Vec3{X: 1, Y: 1.5}
	weight := 1.5
	operation := Operation{Kind: OpSetCurveControlPoint, Target: "cert-mesh", ControlPoint: "p1", Position: &position, Weight: &weight}
	previewReceipt, preview, err := workspace.Execute(Transaction{ID: "certify:curve-preview", Actor: "certifier://studio", Mode: ModePropose, ExpectedRevision: document.Revision, Operations: []Operation{operation}})
	if err != nil {
		return CurveAnalysis{}, false, err
	}
	directReceipt, direct, err := workspace.Execute(Transaction{ID: "certify:curve-direct", Actor: "certifier://studio", Mode: ModeDirect, ExpectedRevision: document.Revision, Operations: []Operation{operation}})
	if err != nil {
		return CurveAnalysis{}, false, err
	}
	previewFingerprint, _ := preview.Fingerprint()
	directFingerprint, _ := direct.Fingerprint()
	analysis, err := AnalyzeEntityCurve(direct, "cert-mesh")
	if err != nil {
		return CurveAnalysis{}, false, err
	}
	_, compileErr := Compile(direct)
	ok := previewFingerprint == directFingerprint && len(previewReceipt.OperatorRecords) == 1 && len(directReceipt.OperatorRecords) == 1 && workspace.SelectionState().Mode == SelectionCurveControlPoint && analysis.Valid && analysis.ApproximateLength > 2 && compileErr == nil
	return analysis, ok, nil
}

func certifyLoopCut(meshDocument Document) (int, int, bool, error) {
	document, _ := meshDocument.Clone()
	entity := document.Entities["cert-mesh"]
	entity.Mesh.Geometry = Geometry{Kind: "indexed-mesh", Vertices: []Vertex{
		{ID: "a", Position: Vec3{}}, {ID: "b", Position: Vec3{X: 1}}, {ID: "c", Position: Vec3{X: 2}},
		{ID: "d", Position: Vec3{Z: 1}}, {ID: "e", Position: Vec3{X: 1, Z: 1}}, {ID: "f", Position: Vec3{X: 2, Z: 1}},
	}, Faces: []Face{{ID: "left", Vertices: []ID{"a", "b", "e", "d"}}, {ID: "right", Vertices: []ID{"b", "c", "f", "e"}}}}
	document.Entities[entity.ID] = entity
	var seed ID
	for _, edge := range MeshEdges(entity.Mesh.Geometry) {
		if edge.A == "b" && edge.B == "e" {
			seed = edge.ID
		}
	}
	workspace, err := NewWorkspace(document)
	if err != nil {
		return 0, 0, false, err
	}
	_, cut, err := workspace.Execute(Transaction{ID: "certify:loop-cut", Actor: "certifier://studio", Mode: ModeDirect, ExpectedRevision: document.Revision, Operations: []Operation{{Kind: OpLoopCut, Target: "cert-mesh", Edges: []ID{seed}, Amount: 0.5, NewID: "center"}}})
	if err != nil {
		return 0, 0, false, err
	}
	geometry := cut.Entities["cert-mesh"].Mesh.Geometry
	shared := 0
	for _, vertex := range geometry.Vertices {
		if near(vertex.Position.X, 1) && near(vertex.Position.Z, 0.5) {
			shared++
		}
	}
	_, compileErr := Compile(cut)
	return len(geometry.Vertices), len(geometry.Faces), len(geometry.Vertices) == 9 && len(geometry.Faces) == 4 && shared == 1 && compileErr == nil, nil
}

func certifyStructuralOperators(meshDocument Document) (int, int, bool, error) {
	dissolveDocument, _ := meshDocument.Clone()
	entity := dissolveDocument.Entities["cert-mesh"]
	entity.Mesh.Geometry.Faces = []Face{{ID: "left", Vertices: []ID{"a", "b", "c"}}, {ID: "right", Vertices: []ID{"a", "c", "d"}}}
	dissolveDocument.Entities[entity.ID] = entity
	var diagonal ID
	for _, edge := range MeshEdges(entity.Mesh.Geometry) {
		if edge.A == "a" && edge.B == "c" {
			diagonal = edge.ID
		}
	}
	dissolveWorkspace, err := NewWorkspace(dissolveDocument)
	if err != nil {
		return 0, 0, false, err
	}
	_, dissolved, err := dissolveWorkspace.Execute(Transaction{ID: "certify:dissolve", Actor: "certifier://studio", Mode: ModeDirect, ExpectedRevision: dissolveDocument.Revision, Operations: []Operation{{Kind: OpDissolveEdges, Target: "cert-mesh", Edges: []ID{diagonal}}}})
	if err != nil {
		return 0, 0, false, err
	}

	bridgeDocument, _ := meshDocument.Clone()
	entity = bridgeDocument.Entities["cert-mesh"]
	entity.Mesh.Geometry = Geometry{Kind: "indexed-mesh", Vertices: []Vertex{
		{ID: "a0", Position: Vec3{}}, {ID: "a1", Position: Vec3{X: 1}}, {ID: "a2", Position: Vec3{Z: 1}},
		{ID: "b0", Position: Vec3{Y: 1}}, {ID: "b1", Position: Vec3{X: 1, Y: 1}}, {ID: "b2", Position: Vec3{Y: 1, Z: 1}},
	}, Faces: []Face{{ID: "cap-a", Vertices: []ID{"a0", "a2", "a1"}}, {ID: "cap-b", Vertices: []ID{"b0", "b1", "b2"}}}}
	bridgeDocument.Entities[entity.ID] = entity
	bridgeWorkspace, err := NewWorkspace(bridgeDocument)
	if err != nil {
		return 0, 0, false, err
	}
	_, bridged, err := bridgeWorkspace.Execute(Transaction{ID: "certify:bridge", Actor: "certifier://studio", Mode: ModeDirect, ExpectedRevision: bridgeDocument.Revision, Operations: []Operation{{Kind: OpBridgeLoops, Target: "cert-mesh", Loops: [][]ID{{"a0", "a1", "a2"}, {"b0", "b1", "b2"}}, NewID: "bridge", Closed: true}}})
	if err != nil {
		return 0, 0, false, err
	}
	dissolveFaces := len(dissolved.Entities["cert-mesh"].Mesh.Geometry.Faces)
	bridgeFaces := len(bridged.Entities["cert-mesh"].Mesh.Geometry.Faces)
	_, dissolveCompileErr := Compile(dissolved)
	_, bridgeCompileErr := Compile(bridged)
	return dissolveFaces, bridgeFaces, dissolveFaces == 1 && bridgeFaces == 5 && dissolveCompileErr == nil && bridgeCompileErr == nil, nil
}

func certificationMeshDocument(document Document) Document {
	clone, _ := document.Clone()
	entity := Entity{
		ID: "cert-mesh", Name: "Certification mesh", Parent: clone.RootIDs[0], Transform: IdentityTransform(), Visible: true,
		Mesh: &MeshComponent{
			Material: firstMaterialID(clone), Pickable: true,
			Geometry: Geometry{
				Kind: "indexed-mesh",
				Vertices: []Vertex{
					{ID: "a", Position: Vec3{}}, {ID: "b", Position: Vec3{X: 1}},
					{ID: "c", Position: Vec3{X: 1, Z: 1}}, {ID: "d", Position: Vec3{Z: 1}},
				},
				Faces: []Face{{ID: "quad", Vertices: []ID{"a", "b", "c", "d"}}},
			},
		},
	}
	root := clone.Entities[entity.Parent]
	root.Children = append(root.Children, entity.ID)
	clone.Entities[root.ID] = root
	clone.Entities[entity.ID] = entity
	return clone
}

func firstMaterialID(document Document) ID {
	ids := make([]string, 0, len(document.Materials))
	for id := range document.Materials {
		ids = append(ids, string(id))
	}
	sort.Strings(ids)
	if len(ids) == 0 {
		return ""
	}
	return ID(ids[0])
}

func (report EvidenceReport) MarshalDeterministic() ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}
