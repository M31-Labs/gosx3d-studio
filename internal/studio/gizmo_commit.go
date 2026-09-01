package studio

import (
	"fmt"
	"strings"
)

// GizmoCommit is the end-phase payload of an interactive TransformControls
// drag (gosx v0.31.17 "gizmo-commit" input events). Exactly one of the value
// fields applies per mode.
type GizmoCommit struct {
	Target      ID       `json:"target"`
	Mode        string   `json:"mode"`
	Axis        string   `json:"axis,omitempty"`
	Phase       string   `json:"phase,omitempty"`
	Position    *Vec3    `json:"position,omitempty"`
	ScaleFactor *float64 `json:"scaleFactor,omitempty"`
	AngleDelta  *float64 `json:"angleDelta,omitempty"`
}

// ApplyGizmoCommit turns one finished gizmo drag into exactly one revision-
// safe transaction (spec: continuous drags commit one undo step). The gizmo
// reports anchor-space values, which equal local values while parent chains
// compose to identity; rotation deltas compose onto the authoritative
// quaternion; scale commits apply to mesh, model, group, and prefab-group
// entities. Light scale remains an explicit error because it has no render
// meaning.
func ApplyGizmoCommit(w *Workspace, commit GizmoCommit) (Receipt, error) {
	// A drag needs one entity's transform and the current revision. Reading
	// them under the read lock avoids deep-cloning the whole SceneDoc on
	// every drag; the transaction below is what makes the change, and its
	// expected revision still rejects a concurrent edit.
	var (
		revision  uint64
		transform Transform
		canScale  bool
		found     bool
	)
	if err := w.Read(func(document *Document) error {
		revision = document.Revision
		entity, ok := document.Entities[commit.Target]
		if !ok {
			return nil
		}
		found = true
		transform = entity.Transform.canonical()
		canScale = entity.Light == nil
		return nil
	}); err != nil {
		return Receipt{}, err
	}
	if !found {
		return Receipt{}, fmt.Errorf("gizmo target %q does not exist", commit.Target)
	}
	switch strings.ToLower(strings.TrimSpace(commit.Mode)) {
	case "translate":
		if commit.Position == nil {
			return Receipt{}, fmt.Errorf("translate commit requires position")
		}
		transform.Position = *commit.Position
	case "rotate":
		if commit.AngleDelta == nil {
			return Receipt{}, fmt.Errorf("rotate commit requires angleDelta")
		}
		delta := QuaternionFromEuler(Vec3{Z: *commit.AngleDelta})
		transform.Rotation = delta.Mul(transform.Rotation).Normalized()
		transform.Euler = transform.Rotation.Euler()
	case "scale":
		if commit.ScaleFactor == nil {
			return Receipt{}, fmt.Errorf("scale commit requires scaleFactor")
		}
		if !canScale {
			return Receipt{}, fmt.Errorf("scale commits are not meaningful for light entities")
		}
		scale := scaleOrUnit(transform.Scale)
		factor := *commit.ScaleFactor
		switch strings.ToLower(strings.TrimSpace(commit.Axis)) {
		case "x":
			scale.X *= factor
		case "y":
			scale.Y *= factor
		case "z":
			scale.Z *= factor
		default:
			scale = Vec3{X: scale.X * factor, Y: scale.Y * factor, Z: scale.Z * factor}
		}
		transform.Scale = scale
	default:
		return Receipt{}, fmt.Errorf("unsupported gizmo mode %q", commit.Mode)
	}
	receipt, _, err := w.Execute(Transaction{
		ID:    fmt.Sprintf("gizmo-%s-%s-r%d", commit.Mode, commit.Target, revision),
		Actor: "human://viewport-gizmo", Mode: ModeDirect, ExpectedRevision: revision,
		Operations: []Operation{{Kind: OpSetTransform, Target: commit.Target, Transform: &transform}},
	})
	return receipt, err
}
