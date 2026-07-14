package studio

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
)

type SelectionMode string

const (
	SelectionObject            SelectionMode = "object"
	SelectionVertex            SelectionMode = "vertex"
	SelectionEdge              SelectionMode = "edge"
	SelectionFace              SelectionMode = "face"
	SelectionCurveControlPoint SelectionMode = "curve-control-point"
)

type SelectionRequest struct {
	ExpectedRevision uint64        `json:"expectedRevision"`
	Mode             SelectionMode `json:"mode"`
	Object           ID            `json:"object,omitempty"`
	IDs              []ID          `json:"ids,omitempty"`
}

type SelectionState struct {
	Revision uint64        `json:"revision"`
	Mode     SelectionMode `json:"mode"`
	Object   ID            `json:"object,omitempty"`
	IDs      []ID          `json:"ids"`
}

type MeshEdge struct {
	ID ID `json:"id"`
	A  ID `json:"a"`
	B  ID `json:"b"`
}

func (w *Workspace) SelectAtRevision(expectedRevision uint64, ids ...ID) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.selectLocked(SelectionRequest{ExpectedRevision: expectedRevision, Mode: SelectionObject, IDs: ids})
}

func (w *Workspace) SelectSubobjects(request SelectionRequest) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.selectLocked(request)
}

func (w *Workspace) SelectionState() SelectionState {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return cloneSelection(w.selectionState)
}

func (w *Workspace) selectLocked(request SelectionRequest) error {
	if request.ExpectedRevision != w.doc.Revision {
		return fmt.Errorf("%w: have %d, expected %d", ErrRevisionConflict, w.doc.Revision, request.ExpectedRevision)
	}
	state, err := validateSelection(w.doc, request)
	if err != nil {
		return err
	}
	w.selection = nil
	if state.Mode == SelectionObject {
		w.selection = append([]ID(nil), state.IDs...)
	} else if state.Object != "" {
		w.selection = []ID{state.Object}
	}
	w.selectionState = state
	return nil
}

func (w *Workspace) syncLegacySelectionLocked() {
	w.selection = nil
	if w.selectionState.Mode == SelectionObject {
		w.selection = append([]ID(nil), w.selectionState.IDs...)
	} else if w.selectionState.Object != "" {
		w.selection = []ID{w.selectionState.Object}
	}
}

func validateSelection(document Document, request SelectionRequest) (SelectionState, error) {
	mode := request.Mode
	if mode == "" {
		mode = SelectionObject
	}
	ids := uniqueIDs(request.IDs)
	if len(ids) != len(request.IDs) {
		return SelectionState{}, fmt.Errorf("selection ids must be non-empty and unique")
	}
	state := SelectionState{Revision: document.Revision, Mode: mode, Object: request.Object, IDs: ids}
	if mode == SelectionObject {
		if request.Object != "" {
			return SelectionState{}, fmt.Errorf("object selection must use ids, not object")
		}
		for _, id := range ids {
			if _, ok := document.Entities[id]; !ok {
				return SelectionState{}, fmt.Errorf("entity %q does not exist", id)
			}
		}
		return state, nil
	}
	entity, ok := document.Entities[request.Object]
	if !ok || entity.Mesh == nil {
		return SelectionState{}, fmt.Errorf("subobject selection target %q is not a mesh", request.Object)
	}
	valid := map[ID]bool{}
	switch mode {
	case SelectionVertex:
		if entity.Mesh.Geometry.Kind != "indexed-mesh" {
			return SelectionState{}, fmt.Errorf("vertex selection requires indexed-mesh geometry")
		}
		for _, vertex := range entity.Mesh.Geometry.Vertices {
			valid[vertex.ID] = true
		}
	case SelectionFace:
		if entity.Mesh.Geometry.Kind != "indexed-mesh" {
			return SelectionState{}, fmt.Errorf("face selection requires indexed-mesh geometry")
		}
		for _, face := range entity.Mesh.Geometry.Faces {
			valid[face.ID] = true
		}
	case SelectionEdge:
		if entity.Mesh.Geometry.Kind != "indexed-mesh" {
			return SelectionState{}, fmt.Errorf("edge selection requires indexed-mesh geometry")
		}
		for _, edge := range MeshEdges(entity.Mesh.Geometry) {
			valid[edge.ID] = true
		}
	case SelectionCurveControlPoint:
		if entity.Mesh.Geometry.Kind != "nurbs-curve" || entity.Mesh.Geometry.Curve == nil {
			return SelectionState{}, fmt.Errorf("curve control-point selection requires nurbs-curve geometry")
		}
		for _, point := range entity.Mesh.Geometry.Curve.ControlPoints {
			valid[point.ID] = true
		}
	default:
		return SelectionState{}, fmt.Errorf("unsupported selection mode %q", mode)
	}
	for _, id := range ids {
		if !valid[id] {
			return SelectionState{}, fmt.Errorf("%s %q does not exist on %q", mode, id, request.Object)
		}
	}
	return state, nil
}

func MeshEdges(geometry Geometry) []MeshEdge {
	edges := map[string]MeshEdge{}
	for _, face := range geometry.Faces {
		for index, a := range face.Vertices {
			b := face.Vertices[(index+1)%len(face.Vertices)]
			left, right := a, b
			if right < left {
				left, right = right, left
			}
			key := fmt.Sprintf("%d:%s%d:%s", len(left), left, len(right), right)
			sum := sha256.Sum256([]byte(key))
			edges[key] = MeshEdge{ID: ID("edge-" + hex.EncodeToString(sum[:])), A: left, B: right}
		}
	}
	keys := make([]string, 0, len(edges))
	for key := range edges {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]MeshEdge, 0, len(keys))
	for _, key := range keys {
		out = append(out, edges[key])
	}
	return out
}

func cloneSelection(state SelectionState) SelectionState {
	state.IDs = append([]ID(nil), state.IDs...)
	return state
}

func reconcileSelection(document Document, state SelectionState) SelectionState {
	if state.Mode == "" {
		return SelectionState{Revision: document.Revision, Mode: SelectionObject, IDs: []ID{}}
	}
	request := SelectionRequest{ExpectedRevision: document.Revision, Mode: state.Mode, Object: state.Object, IDs: state.IDs}
	if next, err := validateSelection(document, request); err == nil {
		return next
	}
	if state.Object != "" {
		if _, ok := document.Entities[state.Object]; ok {
			return SelectionState{Revision: document.Revision, Mode: SelectionObject, IDs: []ID{state.Object}}
		}
	}
	return SelectionState{Revision: document.Revision, Mode: SelectionObject, IDs: []ID{}}
}
