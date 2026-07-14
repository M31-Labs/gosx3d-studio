package studio

import (
	"fmt"
	"sort"
)

func evaluateModifiers(geometry Geometry, modifiers []Modifier) (Geometry, error) {
	if err := validateModifiers(geometry, modifiers); err != nil {
		return Geometry{}, err
	}
	result := geometry
	result.Vertices = append([]Vertex(nil), geometry.Vertices...)
	result.Faces = cloneFaces(geometry.Faces)
	for _, modifier := range modifiers {
		if !modifier.Enabled {
			continue
		}
		var err error
		switch modifier.Kind {
		case "mirror":
			result, err = evaluateMirror(result, modifier)
		case "array":
			result, err = evaluateArray(result, modifier)
		case "solidify":
			result, err = evaluateSolidify(result, modifier)
		case "subdivision":
			result, err = evaluateSubdivision(result, modifier)
		}
		if err != nil {
			return Geometry{}, fmt.Errorf("modifier %q: %w", modifier.ID, err)
		}
		if err := validateGeometry(result); err != nil {
			return Geometry{}, fmt.Errorf("modifier %q output: %w", modifier.ID, err)
		}
	}
	return result, nil
}

type subdivisionEdge struct {
	a, b  ID
	faces []ID
}

func evaluateSubdivision(geometry Geometry, modifier Modifier) (Geometry, error) {
	result := geometry
	for level := 1; level <= modifier.Levels; level++ {
		var err error
		result, err = subdivideOnce(result, modifier.ID, level)
		if err != nil {
			return Geometry{}, err
		}
		if len(result.Faces) > 1_000_000 || len(result.Vertices) > 1_000_000 {
			return Geometry{}, fmt.Errorf("subdivision exceeds one-million element budget at level %d", level)
		}
	}
	return result, nil
}

func subdivideOnce(geometry Geometry, modifierID ID, level int) (Geometry, error) {
	vertices, _ := topologyIndexes(geometry)
	facePoints := make(map[ID]Vertex, len(geometry.Faces))
	edges := map[string]*subdivisionEdge{}
	incidentFaces := map[ID][]ID{}
	neighbors := map[ID]map[ID]bool{}
	for _, face := range geometry.Faces {
		if len(face.Vertices) < 3 {
			return Geometry{}, fmt.Errorf("face %q has fewer than three vertices", face.ID)
		}
		point := Vertex{ID: ID(fmt.Sprintf("%s--sub-%d--face-point--%s", modifierID, level, face.ID))}
		uvComplete := true
		for index, id := range face.Vertices {
			vertex := vertices[id]
			point.Position = addVec(point.Position, vertex.Position)
			if vertex.UV == nil {
				uvComplete = false
			} else {
				if point.UV == nil {
					point.UV = &Vec2{}
				}
				point.UV.X += vertex.UV.X
				point.UV.Y += vertex.UV.Y
			}
			incidentFaces[id] = append(incidentFaces[id], face.ID)
			next := face.Vertices[(index+1)%len(face.Vertices)]
			if neighbors[id] == nil {
				neighbors[id] = map[ID]bool{}
			}
			if neighbors[next] == nil {
				neighbors[next] = map[ID]bool{}
			}
			neighbors[id][next], neighbors[next][id] = true, true
			key := stableEdgeKey(id, next)
			edge := edges[key]
			if edge == nil {
				a, b := id, next
				if b < a {
					a, b = b, a
				}
				edge = &subdivisionEdge{a: a, b: b}
				edges[key] = edge
			}
			edge.faces = append(edge.faces, face.ID)
		}
		point.Position = scaleVec(point.Position, 1/float64(len(face.Vertices)))
		if uvComplete {
			point.UV.X /= float64(len(face.Vertices))
			point.UV.Y /= float64(len(face.Vertices))
		} else {
			point.UV = nil
		}
		facePoints[face.ID] = point
	}

	edgeKeys := make([]string, 0, len(edges))
	boundaryNeighbors := map[ID][]ID{}
	for key, edge := range edges {
		edgeKeys = append(edgeKeys, key)
		if len(edge.faces) == 1 {
			boundaryNeighbors[edge.a] = append(boundaryNeighbors[edge.a], edge.b)
			boundaryNeighbors[edge.b] = append(boundaryNeighbors[edge.b], edge.a)
		} else if len(edge.faces) != 2 {
			return Geometry{}, fmt.Errorf("edge %s is non-manifold with %d faces", key, len(edge.faces))
		}
	}
	sort.Strings(edgeKeys)

	out := Geometry{Kind: "indexed-mesh"}
	for _, original := range geometry.Vertices {
		vertex := original
		boundary := boundaryNeighbors[original.ID]
		if len(boundary) > 0 {
			if len(boundary) != 2 {
				return Geometry{}, fmt.Errorf("boundary vertex %q has %d boundary neighbors", original.ID, len(boundary))
			}
			vertex.Position = addVec(scaleVec(original.Position, 0.75), scaleVec(addVec(vertices[boundary[0]].Position, vertices[boundary[1]].Position), 0.125))
		} else {
			faces := incidentFaces[original.ID]
			n := len(faces)
			if n == 0 || len(neighbors[original.ID]) != n {
				return Geometry{}, fmt.Errorf("vertex %q has inconsistent closed fan", original.ID)
			}
			faceAverage := Vec3{}
			for _, faceID := range faces {
				faceAverage = addVec(faceAverage, facePoints[faceID].Position)
			}
			faceAverage = scaleVec(faceAverage, 1/float64(n))
			edgeAverage := Vec3{}
			for neighbor := range neighbors[original.ID] {
				edgeAverage = addVec(edgeAverage, scaleVec(addVec(original.Position, vertices[neighbor].Position), 0.5))
			}
			edgeAverage = scaleVec(edgeAverage, 1/float64(n))
			vertex.Position = scaleVec(addVec(addVec(faceAverage, scaleVec(edgeAverage, 2)), scaleVec(original.Position, float64(n-3))), 1/float64(n))
		}
		vertex.Normal = Vec3{}
		out.Vertices = append(out.Vertices, vertex)
	}

	edgePointIDs := map[string]ID{}
	for _, key := range edgeKeys {
		edge := edges[key]
		a, b := vertices[edge.a], vertices[edge.b]
		point := Vertex{ID: ID(fmt.Sprintf("%s--sub-%d--edge-point--%s", modifierID, level, key))}
		if len(edge.faces) == 2 {
			point.Position = scaleVec(addVec(addVec(a.Position, b.Position), addVec(facePoints[edge.faces[0]].Position, facePoints[edge.faces[1]].Position)), 0.25)
		} else {
			point.Position = scaleVec(addVec(a.Position, b.Position), 0.5)
		}
		if a.UV != nil && b.UV != nil {
			point.UV = &Vec2{X: (a.UV.X + b.UV.X) / 2, Y: (a.UV.Y + b.UV.Y) / 2}
		}
		edgePointIDs[key] = point.ID
		out.Vertices = append(out.Vertices, point)
	}
	for _, face := range geometry.Faces {
		point := facePoints[face.ID]
		out.Vertices = append(out.Vertices, point)
		for index, id := range face.Vertices {
			previous := face.Vertices[(index+len(face.Vertices)-1)%len(face.Vertices)]
			next := face.Vertices[(index+1)%len(face.Vertices)]
			out.Faces = append(out.Faces, Face{
				ID:       ID(fmt.Sprintf("%s--sub-%d--%s--corner-%d", modifierID, level, face.ID, index)),
				Vertices: []ID{id, edgePointIDs[stableEdgeKey(id, next)], point.ID, edgePointIDs[stableEdgeKey(previous, id)]},
			})
		}
	}
	if err := recalculateNormals(&out); err != nil {
		return Geometry{}, err
	}
	return out, nil
}

func evaluateSolidify(geometry Geometry, modifier Modifier) (Geometry, error) {
	vertices, _ := topologyIndexes(geometry)
	normals := make(map[ID]Vec3, len(vertices))
	edgeUses := map[string]int{}
	edgeDirections := map[string][2]ID{}
	for _, face := range geometry.Faces {
		normal := faceNormal(face, vertices)
		for index, id := range face.Vertices {
			normals[id] = addVec(normals[id], normal)
			next := face.Vertices[(index+1)%len(face.Vertices)]
			key := stableEdgeKey(id, next)
			edgeUses[key]++
			edgeDirections[key] = [2]ID{id, next}
		}
	}
	half := modifier.Thickness / 2
	top := make(map[ID]ID, len(vertices))
	bottom := make(map[ID]ID, len(vertices))
	out := Geometry{Kind: "indexed-mesh"}
	for _, vertex := range geometry.Vertices {
		normal := normalizeVec(normals[vertex.ID])
		if normal == (Vec3{}) {
			return Geometry{}, fmt.Errorf("vertex %q has no usable surface normal", vertex.ID)
		}
		topID := ID(fmt.Sprintf("%s--solidify-top--%s", modifier.ID, vertex.ID))
		bottomID := ID(fmt.Sprintf("%s--solidify-bottom--%s", modifier.ID, vertex.ID))
		top[vertex.ID], bottom[vertex.ID] = topID, bottomID
		topVertex, bottomVertex := vertex, vertex
		topVertex.ID, bottomVertex.ID = topID, bottomID
		topVertex.Position = addVec(vertex.Position, scaleVec(normal, half))
		bottomVertex.Position = addVec(vertex.Position, scaleVec(normal, -half))
		topVertex.Normal, bottomVertex.Normal = normal, scaleVec(normal, -1)
		out.Vertices = append(out.Vertices, topVertex, bottomVertex)
	}
	for _, face := range geometry.Faces {
		topFace := Face{ID: ID(fmt.Sprintf("%s--solidify-top--%s", modifier.ID, face.ID)), Vertices: make([]ID, len(face.Vertices))}
		bottomFace := Face{ID: ID(fmt.Sprintf("%s--solidify-bottom--%s", modifier.ID, face.ID)), Vertices: make([]ID, len(face.Vertices))}
		for index, id := range face.Vertices {
			topFace.Vertices[index] = top[id]
			bottomFace.Vertices[len(face.Vertices)-1-index] = bottom[id]
		}
		out.Faces = append(out.Faces, topFace, bottomFace)
	}
	boundary := make([]string, 0, len(edgeUses))
	for key, uses := range edgeUses {
		if uses != 1 {
			continue
		}
		boundary = append(boundary, key)
	}
	sort.Strings(boundary)
	for _, key := range boundary {
		edge := edgeDirections[key]
		out.Faces = append(out.Faces, Face{
			ID:       ID(fmt.Sprintf("%s--solidify-side--%s", modifier.ID, key)),
			Vertices: []ID{bottom[edge[0]], bottom[edge[1]], top[edge[1]], top[edge[0]]},
		})
	}
	return out, nil
}

func stableEdgeKey(a, b ID) string {
	if b < a {
		a, b = b, a
	}
	return fmt.Sprintf("%d-%s-%d-%s", len(a), a, len(b), b)
}

func evaluateMirror(geometry Geometry, modifier Modifier) (Geometry, error) {
	sourceVertices := append([]Vertex(nil), geometry.Vertices...)
	sourceFaces := cloneFaces(geometry.Faces)
	ids := map[ID]bool{}
	for _, vertex := range geometry.Vertices {
		ids[vertex.ID] = true
	}
	mapID := map[ID]ID{}
	for _, vertex := range sourceVertices {
		originalID := vertex.ID
		newID := ID(fmt.Sprintf("%s--mirror--%s", modifier.ID, originalID))
		if ids[newID] {
			return Geometry{}, fmt.Errorf("generated vertex id %q collides", newID)
		}
		vertex.ID = newID
		switch modifier.Axis {
		case "x":
			vertex.Position.X = -vertex.Position.X
			vertex.Normal.X = -vertex.Normal.X
		case "y":
			vertex.Position.Y = -vertex.Position.Y
			vertex.Normal.Y = -vertex.Normal.Y
		case "z":
			vertex.Position.Z = -vertex.Position.Z
			vertex.Normal.Z = -vertex.Normal.Z
		}
		geometry.Vertices = append(geometry.Vertices, vertex)
		ids[newID] = true
		mapID[originalID] = newID
	}
	faceIDs := map[ID]bool{}
	for _, face := range geometry.Faces {
		faceIDs[face.ID] = true
	}
	for _, face := range sourceFaces {
		newID := ID(fmt.Sprintf("%s--mirror--%s", modifier.ID, face.ID))
		if faceIDs[newID] {
			return Geometry{}, fmt.Errorf("generated face id %q collides", newID)
		}
		mapped := make([]ID, len(face.Vertices))
		for index, id := range face.Vertices {
			mapped[len(face.Vertices)-1-index] = mapID[id]
		}
		geometry.Faces = append(geometry.Faces, Face{ID: newID, Vertices: mapped})
		faceIDs[newID] = true
	}
	return geometry, nil
}

func evaluateArray(geometry Geometry, modifier Modifier) (Geometry, error) {
	sourceVertices := append([]Vertex(nil), geometry.Vertices...)
	sourceFaces := cloneFaces(geometry.Faces)
	vertexIDs := map[ID]bool{}
	for _, vertex := range geometry.Vertices {
		vertexIDs[vertex.ID] = true
	}
	faceIDs := map[ID]bool{}
	for _, face := range geometry.Faces {
		faceIDs[face.ID] = true
	}
	for copyIndex := 1; copyIndex < modifier.Count; copyIndex++ {
		mapID := map[ID]ID{}
		for _, vertex := range sourceVertices {
			originalID := vertex.ID
			newID := ID(fmt.Sprintf("%s--array-%d--%s", modifier.ID, copyIndex, originalID))
			if vertexIDs[newID] {
				return Geometry{}, fmt.Errorf("generated vertex id %q collides", newID)
			}
			vertex.ID = newID
			vertex.Position = addVec(vertex.Position, scaleVec(modifier.Offset, float64(copyIndex)))
			geometry.Vertices = append(geometry.Vertices, vertex)
			vertexIDs[newID] = true
			mapID[originalID] = newID
		}
		for _, face := range sourceFaces {
			newID := ID(fmt.Sprintf("%s--array-%d--%s", modifier.ID, copyIndex, face.ID))
			if faceIDs[newID] {
				return Geometry{}, fmt.Errorf("generated face id %q collides", newID)
			}
			mapped := make([]ID, len(face.Vertices))
			for index, id := range face.Vertices {
				mapped[index] = mapID[id]
			}
			geometry.Faces = append(geometry.Faces, Face{ID: newID, Vertices: mapped})
			faceIDs[newID] = true
		}
	}
	return geometry, nil
}

func cloneFaces(faces []Face) []Face {
	out := make([]Face, len(faces))
	for index, face := range faces {
		out[index] = Face{ID: face.ID, Vertices: append([]ID(nil), face.Vertices...)}
	}
	return out
}

func applySetModifier(document *Document, operation Operation) ([]ID, error) {
	entity, ok := document.Entities[operation.Target]
	if !ok || entity.Mesh == nil {
		return nil, fmt.Errorf("set-modifier target %q is not a mesh", operation.Target)
	}
	if entity.Locked {
		return nil, fmt.Errorf("entity %q is locked", operation.Target)
	}
	if operation.Modifier == nil || operation.Modifier.ID == "" {
		return nil, fmt.Errorf("set-modifier requires modifier with id")
	}
	modifiers := append([]Modifier(nil), entity.Mesh.Modifiers...)
	found := false
	for index := range modifiers {
		if modifiers[index].ID == operation.Modifier.ID {
			modifiers[index] = *operation.Modifier
			found = true
			break
		}
	}
	if !found {
		modifiers = append(modifiers, *operation.Modifier)
	}
	if err := validateModifiers(entity.Mesh.Geometry, modifiers); err != nil {
		return nil, err
	}
	entity.Mesh.Modifiers = modifiers
	document.Entities[entity.ID] = entity
	return []ID{entity.ID}, nil
}

func applyRemoveModifier(document *Document, operation Operation) ([]ID, error) {
	entity, ok := document.Entities[operation.Target]
	if !ok || entity.Mesh == nil {
		return nil, fmt.Errorf("remove-modifier target %q is not a mesh", operation.Target)
	}
	if entity.Locked {
		return nil, fmt.Errorf("entity %q is locked", operation.Target)
	}
	if operation.ModifierID == "" {
		return nil, fmt.Errorf("remove-modifier requires modifierId")
	}
	out := make([]Modifier, 0, len(entity.Mesh.Modifiers))
	found := false
	for _, modifier := range entity.Mesh.Modifiers {
		if modifier.ID == operation.ModifierID {
			found = true
			continue
		}
		out = append(out, modifier)
	}
	if !found {
		return nil, fmt.Errorf("modifier %q does not exist", operation.ModifierID)
	}
	entity.Mesh.Modifiers = out
	document.Entities[entity.ID] = entity
	return []ID{entity.ID}, nil
}

func applyReorderModifier(document *Document, operation Operation) ([]ID, error) {
	entity, ok := document.Entities[operation.Target]
	if !ok || entity.Mesh == nil {
		return nil, fmt.Errorf("reorder-modifier target %q is not a mesh", operation.Target)
	}
	if entity.Locked {
		return nil, fmt.Errorf("entity %q is locked", operation.Target)
	}
	if operation.ModifierID == "" {
		return nil, fmt.Errorf("reorder-modifier requires modifierId")
	}
	if operation.ModifierIndex < 0 || operation.ModifierIndex >= len(entity.Mesh.Modifiers) {
		return nil, fmt.Errorf("modifierIndex %d is outside stack length %d", operation.ModifierIndex, len(entity.Mesh.Modifiers))
	}
	from := -1
	for index, modifier := range entity.Mesh.Modifiers {
		if modifier.ID == operation.ModifierID {
			from = index
			break
		}
	}
	if from < 0 {
		return nil, fmt.Errorf("modifier %q does not exist", operation.ModifierID)
	}
	modifiers := append([]Modifier(nil), entity.Mesh.Modifiers...)
	value := modifiers[from]
	modifiers = append(modifiers[:from], modifiers[from+1:]...)
	target := operation.ModifierIndex
	modifiers = append(modifiers, Modifier{})
	copy(modifiers[target+1:], modifiers[target:])
	modifiers[target] = value
	if err := validateModifiers(entity.Mesh.Geometry, modifiers); err != nil {
		return nil, err
	}
	entity.Mesh.Modifiers = modifiers
	document.Entities[entity.ID] = entity
	return []ID{entity.ID}, nil
}

func applyApplyModifier(document *Document, operation Operation) ([]ID, error) {
	entity, ok := document.Entities[operation.Target]
	if !ok || entity.Mesh == nil {
		return nil, fmt.Errorf("apply-modifier target %q is not a mesh", operation.Target)
	}
	if entity.Locked {
		return nil, fmt.Errorf("entity %q is locked", operation.Target)
	}
	if operation.ModifierID == "" {
		return nil, fmt.Errorf("apply-modifier requires modifierId")
	}
	through := -1
	for index, modifier := range entity.Mesh.Modifiers {
		if modifier.ID == operation.ModifierID {
			through = index
			break
		}
	}
	if through < 0 {
		return nil, fmt.Errorf("modifier %q does not exist", operation.ModifierID)
	}
	baked, err := evaluateModifiers(entity.Mesh.Geometry, entity.Mesh.Modifiers[:through+1])
	if err != nil {
		return nil, err
	}
	remaining := append([]Modifier(nil), entity.Mesh.Modifiers[through+1:]...)
	if err := validateGeometry(baked); err != nil {
		return nil, fmt.Errorf("baked geometry: %w", err)
	}
	if err := validateModifiers(baked, remaining); err != nil {
		return nil, fmt.Errorf("remaining modifier stack: %w", err)
	}
	entity.Mesh.Geometry = baked
	entity.Mesh.Modifiers = remaining
	document.Entities[entity.ID] = entity
	return []ID{entity.ID}, nil
}
