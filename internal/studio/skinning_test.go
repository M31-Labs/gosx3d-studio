package studio

import (
	"bytes"
	"encoding/json"
	"math"
	"testing"
)

func TestLinearBlendSkinningPreservesIdentityAndMovesWeightedVertices(t *testing.T) {
	document := ArticulatedProofDocument()
	armature := document.Armatures["arm"]
	rootPose := armature.Bones["root"].Rest
	rootPose.Position.X = 1
	armature.Pose["root"] = rootPose
	document.Armatures["arm"] = armature
	original := append([]Vertex(nil), document.Entities["skinned"].Mesh.Geometry.Vertices...)
	deformed, report, err := DeformSkinnedGeometry(document, "skinned")
	if err != nil {
		t.Fatal(err)
	}
	if report.Vertices != 3 || report.Influences != 3 || report.MovedVertices != 3 || math.Abs(report.MaximumDelta-1) > 1e-9 {
		t.Fatalf("deformation report = %+v", report)
	}
	for index, vertex := range deformed.Vertices {
		if vertex.ID != original[index].ID || math.Abs(vertex.Position.X-original[index].Position.X-1) > 1e-9 {
			t.Fatalf("vertex %d = %+v, original %+v", index, vertex, original[index])
		}
	}
	if document.Entities["skinned"].Mesh.Geometry.Vertices[0].Position != original[0].Position {
		t.Fatal("deformation mutated authored SceneDoc geometry")
	}
}

func TestSkinnedPoseChangesCanonicalSceneIRAndInvalidatesIncrementalCache(t *testing.T) {
	document := ArticulatedProofDocument()
	compiler := NewIncrementalCompiler()
	initial, initialStats, err := compiler.Compile(document, "")
	if err != nil {
		t.Fatal(err)
	}
	if initialStats.RecompiledEntities != len(document.Entities) {
		t.Fatalf("initial stats = %+v", initialStats)
	}
	armature := document.Armatures["arm"]
	pose := armature.Bones["root"].Rest
	pose.Position.X = 0.75
	armature.Pose["root"] = pose
	document.Armatures["arm"] = armature
	changed, changedStats, err := compiler.Compile(document, "")
	if err != nil {
		t.Fatal(err)
	}
	left, _ := json.Marshal(initial.SceneIR())
	right, _ := json.Marshal(changed.SceneIR())
	if bytes.Equal(left, right) {
		t.Fatal("armature pose did not change canonical SceneIR")
	}
	if changedStats.RecompiledEntities < 2 || changedStats.ReusedSubtrees == 0 {
		t.Fatalf("pose invalidation stats = %+v", changedStats)
	}
}

func TestSkinModifierOrderingFailsExplicitly(t *testing.T) {
	document := ArticulatedProofDocument()
	entity := document.Entities["skinned"]
	entity.Mesh.Modifiers = []Modifier{{ID: "mirror", Kind: "mirror", Enabled: true, Axis: "x"}}
	document.Entities[entity.ID] = entity
	if err := document.Validate(); err == nil {
		t.Fatal("expected explicit skin/modifier ordering validation failure")
	}
}

func TestSkinInspectionReturnsRevisionTaggedStableGeometry(t *testing.T) {
	document := ArticulatedProofDocument()
	inspection, err := InspectSkinDeformation(document, "skinned")
	if err != nil {
		t.Fatal(err)
	}
	if inspection.Revision != document.Revision || inspection.Report.Entity != "skinned" || len(inspection.Geometry.Vertices) != 3 || inspection.Geometry.Vertices[0].ID != "v0" {
		t.Fatalf("skin inspection = %+v", inspection)
	}
}
