package studio

import (
	"fmt"

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
	props, err := Compile(document)
	if err != nil {
		return PickResult{}, err
	}
	trace := scene.TraceGraph(props.Graph, scene.Ray{Origin: toSceneVec(request.Origin), Direction: toSceneVec(request.Direction)}, scene.PickableOnly())
	result := PickResult{Trace: trace}
	if trace.Closest != nil {
		result.Selected = ID(trace.Closest.ID)
	}
	return result, nil
}
