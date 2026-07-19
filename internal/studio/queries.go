package studio

import (
	"fmt"
	"sync"

	"m31labs.dev/gosx/scene"
)

type PickRequest struct {
	Origin    Vec3 `json:"origin"`
	Direction Vec3 `json:"direction"`
	Select    bool `json:"select,omitempty"`
}

type PickResult struct {
	Selected ID             `json:"selected,omitempty"`
	Trace    scene.RayTrace `json:"trace"`
}

type ViewportSelection struct {
	Selected ID     `json:"selected"`
	Kind     string `json:"kind,omitempty"`
	Source   string `json:"source"`
}

type SkinInspection struct {
	Revision uint64                `json:"revision"`
	Report   SkinDeformationReport `json:"report"`
	Geometry Geometry              `json:"geometry"`
}

func InspectSkinDeformation(document Document, entity ID) (SkinInspection, error) {
	geometry, report, err := DeformSkinnedGeometry(document, entity)
	if err != nil {
		return SkinInspection{}, err
	}
	return SkinInspection{Revision: document.Revision, Report: report, Geometry: geometry}, nil
}

// ValidateViewportSelection accepts the stable object ID emitted by the
// mounted Scene3D picker only when it still denotes a visible, pickable mesh in
// the current canonical document. This prevents stale browser hits from
// becoming editor selection truth after a revision change.
func ValidateViewportSelection(document Document, selected ID, kind string) (ViewportSelection, error) {
	entity, ok := document.Entities[selected]
	if !ok {
		return ViewportSelection{}, fmt.Errorf("viewport selected unknown entity %q", selected)
	}
	if !entity.Visible || entity.Mesh == nil || !entity.Mesh.Pickable {
		return ViewportSelection{}, fmt.Errorf("viewport selected non-pickable entity %q", selected)
	}
	return ViewportSelection{Selected: selected, Kind: kind, Source: "scene3d-mount-input"}, nil
}

func ExactPick(document Document, request PickRequest) (PickResult, error) {
	graph, err := compiledPickGraph(document)
	if err != nil {
		return PickResult{}, err
	}
	trace := scene.TraceGraph(graph, scene.Ray{Origin: toSceneVec(request.Origin), Direction: toSceneVec(request.Direction)}, scene.PickableOnly())
	result := PickResult{Trace: trace}
	if trace.Closest != nil {
		result.Selected = ID(trace.Closest.ID)
	}
	return result, nil
}

// ViewportPick is the browser-reported GPU hit forwarded with a viewport
// selection request. World is the GPU picker's world-space hit position;
// absence means the client could not report one and confirmation degrades to
// id-only validation.
type ViewportPick struct {
	Selected ID           `json:"selected"`
	Kind     string       `json:"kind,omitempty"`
	World    *Vec3        `json:"world,omitempty"`
	Ray      *ViewportRay `json:"ray,omitempty"`
	Triangle int          `json:"triangle,omitempty"`
	Depth    float64      `json:"depth,omitempty"`
}

// ViewportRay is the world-space click ray the live interactive camera used,
// reported by the engine's pick event since gosx v0.31.16. When present it is
// the preferred confirmation input: the canonical CPU query retraces the
// exact ray instead of probing from the authored camera.
type ViewportRay struct {
	Origin    Vec3 `json:"origin"`
	Direction Vec3 `json:"direction"`
}

// SelectionDisagreement is the machine-readable record of a GPU/CPU picking
// divergence, per the spec's requirement that such disagreements become
// visible diagnostics rather than silent selection truth.
type SelectionDisagreement struct {
	GPUSelected ID      `json:"gpuSelected"`
	GPUWorld    Vec3    `json:"gpuWorld"`
	CPUSelected ID      `json:"cpuSelected,omitempty"`
	CPUWorld    *Vec3   `json:"cpuWorld,omitempty"`
	DistanceGap float64 `json:"distanceGap"`
	Tolerance   float64 `json:"tolerance"`
	Reason      string  `json:"reason"`
}

// SelectionConfirmation reports how a viewport selection was decided. Method
// "exact-cpu" means the canonical CPU query confirmed or corrected the GPU
// report; "id-only" means the client supplied no world hit and only stale-ID
// validation ran.
type SelectionConfirmation struct {
	Selected     ID                     `json:"selected"`
	Kind         string                 `json:"kind,omitempty"`
	Source       string                 `json:"source"`
	Method       string                 `json:"method"`
	Confirmed    bool                   `json:"confirmed"`
	Revision     uint64                 `json:"revision"`
	Disagreement *SelectionDisagreement `json:"disagreement,omitempty"`
}

const (
	// viewportAgreementTolerance is the world-space distance within which the
	// GPU-reported hit and the canonical CPU hit count as the same point.
	viewportAgreementTolerance = 1e-3
	// viewportProbeBackoff is how far behind the reported point the CPU
	// confirmation ray starts.
	viewportProbeBackoff = 1e-2
	// viewportNearMissLimit is the largest CPU-to-GPU gap still treated as a
	// position disagreement on a real surface; beyond it the canonical scene
	// simply has no surface at the reported point.
	viewportNearMissLimit = 5e-2
)

// ConfirmViewportSelection validates a browser viewport pick against the
// canonical exact CPU query. When the client reports a world-space hit, a
// short probe ray through that point must strike the same entity at the same
// point; the canonical result wins any disagreement and the divergence is
// returned as a diagnostic instead of being silently discarded.
func ConfirmViewportSelection(document Document, pick ViewportPick) (SelectionConfirmation, error) {
	validated, err := ValidateViewportSelection(document, pick.Selected, pick.Kind)
	if err != nil {
		return SelectionConfirmation{}, err
	}
	confirmation := SelectionConfirmation{Selected: validated.Selected, Kind: validated.Kind, Source: validated.Source, Method: "id-only", Revision: document.Revision}
	if pick.Ray != nil && vectorLength(pick.Ray.Direction) > 1e-9 {
		return confirmAlongReportedRay(document, pick, confirmation)
	}
	if pick.World == nil {
		return confirmation, nil
	}
	world := *pick.World
	direction := normalizeOrDefault(Vec3{X: world.X - document.Camera.Position.X, Y: world.Y - document.Camera.Position.Y, Z: world.Z - document.Camera.Position.Z}, Vec3{Y: -1})
	origin := Vec3{X: world.X - direction.X*viewportProbeBackoff, Y: world.Y - direction.Y*viewportProbeBackoff, Z: world.Z - direction.Z*viewportProbeBackoff}
	result, err := ExactPick(document, PickRequest{Origin: origin, Direction: direction})
	if err != nil {
		return SelectionConfirmation{}, err
	}
	confirmation.Method = "exact-cpu"
	confirmation.Source = validated.Source + "+exact-cpu"
	disagreement := &SelectionDisagreement{GPUSelected: pick.Selected, GPUWorld: world, Tolerance: viewportAgreementTolerance}
	if result.Trace.Closest == nil {
		disagreement.Reason = "no-cpu-hit"
		confirmation.Disagreement = disagreement
		return confirmation, nil
	}
	hit := result.Trace.Closest
	cpuWorld := Vec3{X: hit.Point.X, Y: hit.Point.Y, Z: hit.Point.Z}
	gap := vectorLength(Vec3{X: cpuWorld.X - world.X, Y: cpuWorld.Y - world.Y, Z: cpuWorld.Z - world.Z})
	disagreement.CPUSelected = ID(hit.ID)
	disagreement.CPUWorld = &cpuWorld
	disagreement.DistanceGap = gap
	if gap > viewportNearMissLimit {
		disagreement.Reason = "no-cpu-hit"
		confirmation.Disagreement = disagreement
		return confirmation, nil
	}
	switch {
	case ID(hit.ID) != pick.Selected:
		disagreement.Reason = "id-mismatch"
		confirmation.Selected = ID(hit.ID)
		confirmation.Kind = hit.Kind
		confirmation.Confirmed = true
		confirmation.Disagreement = disagreement
	case gap > viewportAgreementTolerance:
		disagreement.Reason = "position-gap"
		confirmation.Confirmed = true
		confirmation.Disagreement = disagreement
	default:
		confirmation.Confirmed = true
	}
	return confirmation, nil
}

func normalizeOrDefault(value Vec3, fallback Vec3) Vec3 {
	length := vectorLength(value)
	if length <= 1e-9 {
		return fallback
	}
	return Vec3{X: value.X / length, Y: value.Y / length, Z: value.Z / length}
}

// confirmAlongReportedRay retraces the client's exact click ray through the
// canonical CPU query. The closest hit along that ray IS the canonical
// selection: an ID mismatch is corrected in the canonical result's favor, a
// world-point gap on the same entity stays visible as a diagnostic, and a
// miss refuses confirmation.
func confirmAlongReportedRay(document Document, pick ViewportPick, confirmation SelectionConfirmation) (SelectionConfirmation, error) {
	direction := normalizeOrDefault(pick.Ray.Direction, Vec3{Y: -1})
	result, err := ExactPick(document, PickRequest{Origin: pick.Ray.Origin, Direction: direction})
	if err != nil {
		return SelectionConfirmation{}, err
	}
	confirmation.Method = "exact-cpu-ray"
	confirmation.Source = confirmation.Source + "+exact-cpu-ray"
	disagreement := &SelectionDisagreement{GPUSelected: pick.Selected, Tolerance: viewportAgreementTolerance}
	if pick.World != nil {
		disagreement.GPUWorld = *pick.World
	}
	if result.Trace.Closest == nil {
		disagreement.Reason = "no-cpu-hit"
		confirmation.Disagreement = disagreement
		return confirmation, nil
	}
	hit := result.Trace.Closest
	cpuWorld := Vec3{X: hit.Point.X, Y: hit.Point.Y, Z: hit.Point.Z}
	disagreement.CPUSelected = ID(hit.ID)
	disagreement.CPUWorld = &cpuWorld
	if pick.World != nil {
		disagreement.DistanceGap = vectorLength(Vec3{X: cpuWorld.X - pick.World.X, Y: cpuWorld.Y - pick.World.Y, Z: cpuWorld.Z - pick.World.Z})
	}
	switch {
	case ID(hit.ID) != pick.Selected:
		disagreement.Reason = "id-mismatch"
		confirmation.Selected = ID(hit.ID)
		confirmation.Kind = hit.Kind
		confirmation.Confirmed = true
		confirmation.Disagreement = disagreement
	case pick.World != nil && disagreement.DistanceGap > viewportAgreementTolerance:
		disagreement.Reason = "position-gap"
		confirmation.Confirmed = true
		confirmation.Disagreement = disagreement
	default:
		confirmation.Confirmed = true
	}
	return confirmation, nil
}

// pickGraphCache reuses the compiled scene graph across picks against the
// same document revision+fingerprint. A click confirmation can trace twice
// (selection + certification paths); recompiling per trace is the dominant
// cost at scale.
var pickGraphCache struct {
	mu          sync.Mutex
	revision    uint64
	fingerprint string
	graph       scene.Graph
	valid       bool
}

func compiledPickGraph(document Document) (scene.Graph, error) {
	fingerprint, err := document.Fingerprint()
	if err != nil {
		return scene.Graph{}, err
	}
	pickGraphCache.mu.Lock()
	if pickGraphCache.valid && pickGraphCache.revision == document.Revision && pickGraphCache.fingerprint == fingerprint {
		graph := pickGraphCache.graph
		pickGraphCache.mu.Unlock()
		return graph, nil
	}
	pickGraphCache.mu.Unlock()
	props, err := Compile(document)
	if err != nil {
		return scene.Graph{}, err
	}
	pickGraphCache.mu.Lock()
	pickGraphCache.revision = document.Revision
	pickGraphCache.fingerprint = fingerprint
	pickGraphCache.graph = props.Graph
	pickGraphCache.valid = true
	pickGraphCache.mu.Unlock()
	return props.Graph, nil
}
