package studio

import (
	"fmt"
	"math"
	"sort"
)

func applyMeshOperation(document *Document, operation Operation) ([]ID, error) {
	if operation.CoordinateSpace != "" && operation.CoordinateSpace != "object" {
		return nil, fmt.Errorf("%s currently supports object coordinate space only", operation.Kind)
	}
	entity, ok := document.Entities[operation.Target]
	if !ok || entity.Mesh == nil {
		return nil, fmt.Errorf("%s target %q is not a mesh", operation.Kind, operation.Target)
	}
	if entity.Locked {
		return nil, fmt.Errorf("entity %q is locked", operation.Target)
	}
	if entity.Mesh.Geometry.Kind != "indexed-mesh" {
		return nil, fmt.Errorf("%s requires indexed-mesh geometry", operation.Kind)
	}
	var err error
	switch operation.Kind {
	case OpExtrudeFaces:
		err = extrudeFaces(&entity.Mesh.Geometry, operation)
	case OpInsetFaces:
		err = insetFaces(&entity.Mesh.Geometry, operation)
	case OpTriangulateFaces:
		err = triangulateFaces(&entity.Mesh.Geometry, operation)
	case OpWeldVertices:
		err = weldVertices(&entity.Mesh.Geometry, operation)
	case OpFillFace:
		err = fillFace(&entity.Mesh.Geometry, operation)
	case OpRecalculateNormals:
		err = recalculateNormals(&entity.Mesh.Geometry)
	case OpProjectPlanarUV:
		err = projectPlanarUV(&entity.Mesh.Geometry, operation)
	case OpDissolveEdges:
		err = dissolveEdges(&entity.Mesh.Geometry, operation)
	case OpBevelEdges:
		err = bevelEdges(&entity.Mesh.Geometry, operation)
	case OpBridgeLoops:
		err = bridgeLoops(&entity.Mesh.Geometry, operation)
	case OpLoopCut:
		err = loopCut(&entity.Mesh.Geometry, operation)
	default:
		err = fmt.Errorf("unsupported mesh operation %q", operation.Kind)
	}
	if err != nil {
		return nil, err
	}
	document.Entities[entity.ID] = entity
	return []ID{entity.ID}, nil
}

func applyCreate(document *Document, operation Operation) ([]ID, error) {
	if operation.Entity == nil {
		return nil, fmt.Errorf("create-entity requires entity")
	}
	entity := *operation.Entity
	if entity.ID == "" {
		entity.ID = operation.Target
	}
	if entity.ID == "" {
		return nil, fmt.Errorf("entity id is required")
	}
	if _, exists := document.Entities[entity.ID]; exists {
		return nil, fmt.Errorf("entity %q already exists", entity.ID)
	}
	entity.Parent = operation.Parent
	if entity.Transform.Scale == (Vec3{}) {
		entity.Transform.Scale = Vec3{X: 1, Y: 1, Z: 1}
	}
	if entity.Parent == "" {
		document.RootIDs = append(document.RootIDs, entity.ID)
	} else {
		parent, ok := document.Entities[entity.Parent]
		if !ok {
			return nil, fmt.Errorf("parent %q does not exist", entity.Parent)
		}
		parent.Children = append(parent.Children, entity.ID)
		document.Entities[parent.ID] = parent
	}
	document.Entities[entity.ID] = entity
	return []ID{entity.ID}, nil
}

func applyDelete(document *Document, operation Operation) ([]ID, error) {
	entity, ok := document.Entities[operation.Target]
	if !ok {
		return nil, fmt.Errorf("entity %q does not exist", operation.Target)
	}
	if entity.Locked {
		return nil, fmt.Errorf("entity %q is locked", operation.Target)
	}
	ids := []ID{}
	var walk func(ID)
	walk = func(id ID) {
		for _, child := range document.Entities[id].Children {
			walk(child)
		}
		ids = append(ids, id)
		delete(document.Entities, id)
	}
	walk(entity.ID)
	if entity.Parent == "" {
		document.RootIDs = removeID(document.RootIDs, entity.ID)
	} else {
		parent := document.Entities[entity.Parent]
		parent.Children = removeID(parent.Children, entity.ID)
		document.Entities[parent.ID] = parent
	}
	return ids, nil
}

func applyReparent(document *Document, operation Operation) ([]ID, error) {
	entity, ok := document.Entities[operation.Target]
	if !ok {
		return nil, fmt.Errorf("entity %q does not exist", operation.Target)
	}
	if entity.Locked {
		return nil, fmt.Errorf("entity %q is locked", operation.Target)
	}
	if operation.Parent == entity.ID {
		return nil, fmt.Errorf("entity cannot parent itself")
	}
	if entity.Parent == "" {
		document.RootIDs = removeID(document.RootIDs, entity.ID)
	} else {
		old := document.Entities[entity.Parent]
		old.Children = removeID(old.Children, entity.ID)
		document.Entities[old.ID] = old
	}
	entity.Parent = operation.Parent
	if operation.Parent == "" {
		document.RootIDs = append(document.RootIDs, entity.ID)
	} else {
		parent, exists := document.Entities[operation.Parent]
		if !exists {
			return nil, fmt.Errorf("parent %q does not exist", operation.Parent)
		}
		parent.Children = append(parent.Children, entity.ID)
		document.Entities[parent.ID] = parent
	}
	document.Entities[entity.ID] = entity
	return []ID{entity.ID}, nil
}

func applyDuplicate(document *Document, operation Operation) ([]ID, error) {
	source, ok := document.Entities[operation.Target]
	if !ok {
		return nil, fmt.Errorf("entity %q does not exist", operation.Target)
	}
	if operation.NewID == "" {
		return nil, fmt.Errorf("duplicate-entity requires newId")
	}
	if _, exists := document.Entities[operation.NewID]; exists {
		return nil, fmt.Errorf("entity %q already exists", operation.NewID)
	}
	parent := operation.Parent
	if parent == "" {
		parent = source.Parent
	}
	ids := []ID{}
	var clone func(ID, ID, ID) error
	clone = func(sourceID, newID, parentID ID) error {
		original := document.Entities[sourceID]
		copy := original
		copy.ID = newID
		copy.Name = original.Name + " Copy"
		copy.Parent = parentID
		copy.Children = nil
		for _, childID := range original.Children {
			next := ID(string(newID) + "--" + string(childID))
			copy.Children = append(copy.Children, next)
			if err := clone(childID, next, newID); err != nil {
				return err
			}
		}
		document.Entities[newID] = copy
		ids = append(ids, newID)
		return nil
	}
	if err := clone(source.ID, operation.NewID, parent); err != nil {
		return nil, err
	}
	if parent == "" {
		document.RootIDs = append(document.RootIDs, operation.NewID)
	} else {
		p, exists := document.Entities[parent]
		if !exists {
			return nil, fmt.Errorf("parent %q does not exist", parent)
		}
		p.Children = append(p.Children, operation.NewID)
		document.Entities[parent] = p
	}
	return ids, nil
}

func applyExtrude(document *Document, operation Operation) ([]ID, error) {
	return applyMeshOperation(document, operation)
}

func extrudeFaces(geometry *Geometry, operation Operation) error {
	if len(operation.Faces) == 0 || operation.Distance == 0 {
		return fmt.Errorf("extrude-faces requires faces and non-zero distance")
	}
	vertexByID, faceByID := topologyIndexes(*geometry)
	selected, err := requireIDs(operation.Faces, faceByID, "face")
	if err != nil {
		return err
	}
	newFaces := make([]Face, 0, len(geometry.Faces)+len(operation.Faces)*5)
	for _, face := range geometry.Faces {
		if !selected[face.ID] {
			newFaces = append(newFaces, face)
			continue
		}
		normal := faceNormal(face, vertexByID)
		topIDs := make([]ID, len(face.Vertices))
		for i, oldID := range face.Vertices {
			newID := ID(fmt.Sprintf("%s--x-%s", face.ID, oldID))
			if _, exists := vertexByID[newID]; exists {
				return fmt.Errorf("extrude generated duplicate vertex id %q", newID)
			}
			v := vertexByID[oldID]
			v.ID = newID
			v.Position.X += normal.X * operation.Distance
			v.Position.Y += normal.Y * operation.Distance
			v.Position.Z += normal.Z * operation.Distance
			geometry.Vertices = append(geometry.Vertices, v)
			vertexByID[newID] = v
			topIDs[i] = newID
		}
		newFaces = append(newFaces, Face{ID: face.ID + "--top", Vertices: topIDs})
		for i, oldID := range face.Vertices {
			next := (i + 1) % len(face.Vertices)
			newFaces = append(newFaces, Face{ID: ID(fmt.Sprintf("%s--side-%d", face.ID, i)), Vertices: []ID{oldID, face.Vertices[next], topIDs[next], topIDs[i]}})
		}
	}
	geometry.Faces = newFaces
	return nil
}

func insetFaces(geometry *Geometry, operation Operation) error {
	if operation.Amount <= 0 || operation.Amount >= 1 || len(operation.Faces) == 0 {
		return fmt.Errorf("inset-faces requires faces and amount strictly between zero and one")
	}
	vertices, faces := topologyIndexes(*geometry)
	selected, err := requireIDs(operation.Faces, faces, "face")
	if err != nil {
		return err
	}
	out := make([]Face, 0, len(geometry.Faces)+len(operation.Faces)*4)
	for _, face := range geometry.Faces {
		if !selected[face.ID] {
			out = append(out, face)
			continue
		}
		centroid := Vec3{}
		for _, id := range face.Vertices {
			centroid = addVec(centroid, vertices[id].Position)
		}
		centroid = scaleVec(centroid, 1/float64(len(face.Vertices)))
		inner := make([]ID, len(face.Vertices))
		for i, oldID := range face.Vertices {
			newID := ID(fmt.Sprintf("%s--inset-%s", face.ID, oldID))
			if _, exists := vertices[newID]; exists {
				return fmt.Errorf("inset generated duplicate vertex id %q", newID)
			}
			old := vertices[oldID]
			old.ID = newID
			old.Position = addVec(scaleVec(old.Position, 1-operation.Amount), scaleVec(centroid, operation.Amount))
			old.Normal = Vec3{}
			geometry.Vertices = append(geometry.Vertices, old)
			vertices[newID] = old
			inner[i] = newID
		}
		out = append(out, Face{ID: face.ID, Vertices: inner})
		for i, oldID := range face.Vertices {
			next := (i + 1) % len(face.Vertices)
			out = append(out, Face{ID: ID(fmt.Sprintf("%s--rim-%d", face.ID, i)), Vertices: []ID{oldID, face.Vertices[next], inner[next], inner[i]}})
		}
	}
	geometry.Faces = out
	return nil
}

func triangulateFaces(geometry *Geometry, operation Operation) error {
	if len(operation.Faces) == 0 {
		return fmt.Errorf("triangulate-faces requires faces")
	}
	_, faces := topologyIndexes(*geometry)
	selected, err := requireIDs(operation.Faces, faces, "face")
	if err != nil {
		return err
	}
	out := make([]Face, 0, len(geometry.Faces)*2)
	for _, face := range geometry.Faces {
		if !selected[face.ID] || len(face.Vertices) == 3 {
			out = append(out, face)
			continue
		}
		for i := 1; i+1 < len(face.Vertices); i++ {
			id := face.ID
			if i > 1 {
				id = ID(fmt.Sprintf("%s--tri-%d", face.ID, i))
			}
			out = append(out, Face{ID: id, Vertices: []ID{face.Vertices[0], face.Vertices[i], face.Vertices[i+1]}})
		}
	}
	geometry.Faces = out
	return nil
}

// weldToleranceLimit bounds how far apart welded vertices may be. Without an
// upper bound, a tolerance of 1e308 makes the distance check below true for
// every finite distance, which is the same unbounded weld a non-finite
// tolerance produced.
const weldToleranceLimit = 1e6

func weldVertices(geometry *Geometry, operation Operation) error {
	// Test the tolerance for finiteness first. Every comparison against NaN is
	// false, so `Tolerance <= 0` admitted NaN here and `distance > Tolerance`
	// then accepted every vertex, welding arbitrarily distant ones into their
	// centroid. The result validates and commits. The browser form reaches
	// this through strconv.ParseFloat, which accepts the literal "NaN".
	if !finite(operation.Tolerance) || operation.Tolerance <= 0 || operation.Tolerance > weldToleranceLimit {
		return fmt.Errorf("weld-vertices requires a finite tolerance in (0, %g]", weldToleranceLimit)
	}
	if len(operation.Vertices) < 2 {
		return fmt.Errorf("weld-vertices requires at least two vertices")
	}
	vertices, _ := topologyIndexes(*geometry)
	ids := uniqueIDs(operation.Vertices)
	if len(ids) < 2 {
		return fmt.Errorf("weld-vertices requires at least two distinct vertices")
	}
	for _, id := range ids {
		if _, ok := vertices[id]; !ok {
			return fmt.Errorf("vertex %q does not exist", id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	centroid := Vec3{}
	for _, id := range ids {
		centroid = addVec(centroid, vertices[id].Position)
	}
	centroid = scaleVec(centroid, 1/float64(len(ids)))
	for _, id := range ids {
		if distanceVec(vertices[id].Position, centroid) > operation.Tolerance {
			return fmt.Errorf("vertex %q exceeds weld tolerance", id)
		}
	}
	survivor := ids[0]
	welded := map[ID]bool{}
	for _, id := range ids {
		welded[id] = true
	}
	for i := range geometry.Vertices {
		if geometry.Vertices[i].ID == survivor {
			geometry.Vertices[i].Position = centroid
			geometry.Vertices[i].Normal = Vec3{}
		}
	}
	keptVertices := geometry.Vertices[:0]
	for _, vertex := range geometry.Vertices {
		if vertex.ID == survivor || !welded[vertex.ID] {
			keptVertices = append(keptVertices, vertex)
		}
	}
	geometry.Vertices = keptVertices
	keptFaces := geometry.Faces[:0]
	for _, face := range geometry.Faces {
		mapped := make([]ID, 0, len(face.Vertices))
		for _, id := range face.Vertices {
			if welded[id] {
				id = survivor
			}
			if len(mapped) == 0 || mapped[len(mapped)-1] != id {
				mapped = append(mapped, id)
			}
		}
		if len(mapped) > 1 && mapped[0] == mapped[len(mapped)-1] {
			mapped = mapped[:len(mapped)-1]
		}
		if len(uniqueIDs(mapped)) >= 3 {
			face.Vertices = mapped
			keptFaces = append(keptFaces, face)
		}
	}
	geometry.Faces = keptFaces
	if len(geometry.Faces) == 0 {
		return fmt.Errorf("weld would remove every face")
	}
	return nil
}

func fillFace(geometry *Geometry, operation Operation) error {
	if len(operation.Vertices) < 3 || operation.NewID == "" {
		return fmt.Errorf("fill-face requires at least three vertices and newId")
	}
	vertices, faces := topologyIndexes(*geometry)
	if _, exists := faces[operation.NewID]; exists {
		return fmt.Errorf("face %q already exists", operation.NewID)
	}
	ids := uniqueIDs(operation.Vertices)
	if len(ids) != len(operation.Vertices) {
		return fmt.Errorf("fill-face vertices must be distinct")
	}
	for _, id := range ids {
		if _, ok := vertices[id]; !ok {
			return fmt.Errorf("vertex %q does not exist", id)
		}
	}
	geometry.Faces = append(geometry.Faces, Face{ID: operation.NewID, Vertices: append([]ID(nil), ids...)})
	return nil
}

func recalculateNormals(geometry *Geometry) error {
	vertices, _ := topologyIndexes(*geometry)
	sums := map[ID]Vec3{}
	for _, face := range geometry.Faces {
		normal := faceNormal(face, vertices)
		for _, id := range face.Vertices {
			sums[id] = addVec(sums[id], normal)
		}
	}
	for i := range geometry.Vertices {
		geometry.Vertices[i].Normal = normalizeVec(sums[geometry.Vertices[i].ID])
	}
	return nil
}

func projectPlanarUV(geometry *Geometry, operation Operation) error {
	if operation.Projection != "xy" && operation.Projection != "xz" && operation.Projection != "yz" {
		return fmt.Errorf("project-planar-uv requires projection xy, xz, or yz")
	}
	selected := map[ID]bool{}
	if len(operation.Vertices) > 0 {
		vertices, _ := topologyIndexes(*geometry)
		var err error
		selected, err = requireIDs(operation.Vertices, vertices, "vertex")
		if err != nil {
			return err
		}
	} else {
		for _, vertex := range geometry.Vertices {
			selected[vertex.ID] = true
		}
	}
	if len(selected) < 2 {
		return fmt.Errorf("project-planar-uv requires at least two vertices")
	}
	minU, minV := math.Inf(1), math.Inf(1)
	maxU, maxV := math.Inf(-1), math.Inf(-1)
	project := func(position Vec3) (float64, float64) {
		switch operation.Projection {
		case "xy":
			return position.X, position.Y
		case "yz":
			return position.Y, position.Z
		default:
			return position.X, position.Z
		}
	}
	for _, vertex := range geometry.Vertices {
		if !selected[vertex.ID] {
			continue
		}
		u, v := project(vertex.Position)
		minU, maxU = math.Min(minU, u), math.Max(maxU, u)
		minV, maxV = math.Min(minV, v), math.Max(maxV, v)
	}
	if maxU-minU <= 1e-12 || maxV-minV <= 1e-12 {
		return fmt.Errorf("projection %s collapses the selected vertices", operation.Projection)
	}
	for index := range geometry.Vertices {
		if !selected[geometry.Vertices[index].ID] {
			continue
		}
		u, v := project(geometry.Vertices[index].Position)
		geometry.Vertices[index].UV = &Vec2{X: (u - minU) / (maxU - minU), Y: (v - minV) / (maxV - minV)}
	}
	return nil
}

func dissolveEdges(geometry *Geometry, operation Operation) error {
	if len(operation.Edges) != 1 {
		return fmt.Errorf("dissolve-edges currently requires exactly one stable edge id")
	}
	var target MeshEdge
	found := false
	for _, edge := range MeshEdges(*geometry) {
		if edge.ID == operation.Edges[0] {
			target, found = edge, true
			break
		}
	}
	if !found {
		return fmt.Errorf("edge %q does not exist", operation.Edges[0])
	}
	adjacent := make([]int, 0, 2)
	for index, face := range geometry.Faces {
		if faceContainsEdge(face, target.A, target.B) {
			adjacent = append(adjacent, index)
		}
	}
	if len(adjacent) != 2 {
		return fmt.Errorf("edge %q belongs to %d faces, want exactly two", target.ID, len(adjacent))
	}
	left, right := geometry.Faces[adjacent[0]], geometry.Faces[adjacent[1]]
	neighbors := map[ID][]ID{}
	for _, face := range []Face{left, right} {
		for index, a := range face.Vertices {
			b := face.Vertices[(index+1)%len(face.Vertices)]
			if canonicalEdgeKey(a, b) == canonicalEdgeKey(target.A, target.B) {
				continue
			}
			neighbors[a] = appendUniqueID(neighbors[a], b)
			neighbors[b] = appendUniqueID(neighbors[b], a)
		}
	}
	for id, values := range neighbors {
		if len(values) != 2 {
			return fmt.Errorf("dissolving edge %q produces non-polygon boundary at %q", target.ID, id)
		}
		sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
		neighbors[id] = values
	}
	ids := make([]ID, 0, len(neighbors))
	for id := range neighbors {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	start := ids[0]
	merged := []ID{start}
	previous, current := ID(""), start
	for {
		choices := neighbors[current]
		next := choices[0]
		if next == previous {
			next = choices[1]
		}
		if next == start {
			break
		}
		if len(merged) > len(neighbors) {
			return fmt.Errorf("dissolve boundary did not close")
		}
		merged = append(merged, next)
		previous, current = current, next
	}
	if len(merged) < 3 || len(merged) != len(neighbors) {
		return fmt.Errorf("dissolve produced invalid polygon with %d vertices", len(merged))
	}
	faceID := left.ID
	if right.ID < faceID {
		faceID = right.ID
	}
	out := make([]Face, 0, len(geometry.Faces)-1)
	for index, face := range geometry.Faces {
		if index == adjacent[0] {
			out = append(out, Face{ID: faceID, Vertices: merged})
		}
		if index != adjacent[0] && index != adjacent[1] {
			out = append(out, face)
		}
	}
	geometry.Faces = out
	return nil
}

func bridgeLoops(geometry *Geometry, operation Operation) error {
	if len(operation.Loops) != 2 || len(operation.Loops[0]) < 2 || len(operation.Loops[0]) != len(operation.Loops[1]) || operation.NewID == "" {
		return fmt.Errorf("bridge-loops requires two equally sized loops and newId")
	}
	if !operation.Closed {
		return fmt.Errorf("bridge-loops currently requires closed=true")
	}
	vertices, faces := topologyIndexes(*geometry)
	seen := map[ID]bool{}
	for _, loop := range operation.Loops {
		for _, id := range loop {
			if _, ok := vertices[id]; !ok {
				return fmt.Errorf("vertex %q does not exist", id)
			}
			if seen[id] {
				return fmt.Errorf("bridge loop vertex %q is repeated", id)
			}
			seen[id] = true
		}
	}
	for index := range operation.Loops[0] {
		id := ID(fmt.Sprintf("%s--%d", operation.NewID, index))
		if _, exists := faces[id]; exists {
			return fmt.Errorf("bridge face %q already exists", id)
		}
	}
	count := len(operation.Loops[0])
	for index := 0; index < count; index++ {
		next := (index + 1) % count
		geometry.Faces = append(geometry.Faces, Face{ID: ID(fmt.Sprintf("%s--%d", operation.NewID, index)), Vertices: []ID{operation.Loops[0][index], operation.Loops[0][next], operation.Loops[1][next], operation.Loops[1][index]}})
	}
	return nil
}

func loopCut(geometry *Geometry, operation Operation) error {
	if len(operation.Edges) != 1 || operation.Amount <= 0 || operation.Amount >= 1 || operation.NewID == "" {
		return fmt.Errorf("loop-cut requires one stable edge, amount strictly between zero and one, and newId")
	}
	vertices, faces := topologyIndexes(*geometry)
	var seed MeshEdge
	found := false
	for _, edge := range MeshEdges(*geometry) {
		if edge.ID == operation.Edges[0] {
			seed, found = edge, true
			break
		}
	}
	if !found {
		return fmt.Errorf("edge %q does not exist", operation.Edges[0])
	}
	allowed := map[ID]bool{}
	if len(operation.Faces) > 0 {
		var err error
		allowed, err = requireIDs(operation.Faces, faces, "face")
		if err != nil {
			return err
		}
	} else {
		for id := range faces {
			allowed[id] = true
		}
	}
	type step struct {
		faceIndex int
		a, b      ID
	}
	queue := []step{}
	for index, face := range geometry.Faces {
		if allowed[face.ID] && faceContainsEdge(face, seed.A, seed.B) {
			queue = append(queue, step{index, seed.A, seed.B})
		}
	}
	if len(queue) == 0 {
		return fmt.Errorf("seed edge %q is not on an allowed face", seed.ID)
	}
	visited := map[int]bool{}
	replacements := map[int][2]Face{}
	generated := map[string]ID{}
	newVertices := []Vertex{}
	cutVertex := func(a, b ID) (ID, error) {
		key := canonicalEdgeKey(a, b)
		if id, ok := generated[key]; ok {
			return id, nil
		}
		left, right := a, b
		if right < left {
			left, right = right, left
		}
		var edgeID ID
		for _, edge := range MeshEdges(Geometry{Faces: []Face{{ID: "edge", Vertices: []ID{left, right}}}}) {
			edgeID = edge.ID
			break
		}
		id := ID(fmt.Sprintf("%s--%s", operation.NewID, edgeID))
		if _, exists := vertices[id]; exists {
			return "", fmt.Errorf("loop-cut vertex %q already exists", id)
		}
		first, second := vertices[left], vertices[right]
		vertex := Vertex{ID: id, Position: addVec(scaleVec(first.Position, 1-operation.Amount), scaleVec(second.Position, operation.Amount))}
		vertex.Normal = normalizeVec(addVec(scaleVec(first.Normal, 1-operation.Amount), scaleVec(second.Normal, operation.Amount)))
		if first.UV != nil && second.UV != nil {
			vertex.UV = &Vec2{X: first.UV.X*(1-operation.Amount) + second.UV.X*operation.Amount, Y: first.UV.Y*(1-operation.Amount) + second.UV.Y*operation.Amount}
		}
		generated[key] = id
		vertices[id] = vertex
		newVertices = append(newVertices, vertex)
		return id, nil
	}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if visited[current.faceIndex] {
			continue
		}
		face := geometry.Faces[current.faceIndex]
		if len(face.Vertices) != 4 {
			return fmt.Errorf("loop-cut reached non-quad face %q", face.ID)
		}
		start := -1
		for index, a := range face.Vertices {
			if canonicalEdgeKey(a, face.Vertices[(index+1)%4]) == canonicalEdgeKey(current.a, current.b) {
				start = index
				break
			}
		}
		if start < 0 {
			return fmt.Errorf("loop-cut traversal lost edge on face %q", face.ID)
		}
		r := []ID{face.Vertices[start], face.Vertices[(start+1)%4], face.Vertices[(start+2)%4], face.Vertices[(start+3)%4]}
		cutA, err := cutVertex(r[0], r[1])
		if err != nil {
			return err
		}
		cutB, err := cutVertex(r[2], r[3])
		if err != nil {
			return err
		}
		newFaceID := ID(fmt.Sprintf("%s--%s", face.ID, operation.NewID))
		if _, exists := faces[newFaceID]; exists {
			return fmt.Errorf("loop-cut face %q already exists", newFaceID)
		}
		replacements[current.faceIndex] = [2]Face{{ID: face.ID, Vertices: []ID{cutA, r[1], r[2], cutB}}, {ID: newFaceID, Vertices: []ID{r[0], cutA, cutB, r[3]}}}
		visited[current.faceIndex] = true
		for index, other := range geometry.Faces {
			if index != current.faceIndex && !visited[index] && allowed[other.ID] && faceContainsEdge(other, r[2], r[3]) {
				queue = append(queue, step{index, r[2], r[3]})
			}
		}
	}
	geometry.Vertices = append(geometry.Vertices, newVertices...)
	out := make([]Face, 0, len(geometry.Faces)+len(replacements))
	for index, face := range geometry.Faces {
		if pair, ok := replacements[index]; ok {
			out = append(out, pair[0], pair[1])
		} else {
			out = append(out, face)
		}
	}
	geometry.Faces = out
	return nil
}

func faceContainsEdge(face Face, a, b ID) bool {
	for index, current := range face.Vertices {
		if canonicalEdgeKey(current, face.Vertices[(index+1)%len(face.Vertices)]) == canonicalEdgeKey(a, b) {
			return true
		}
	}
	return false
}

func appendUniqueID(values []ID, value ID) []ID {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func topologyIndexes(geometry Geometry) (map[ID]Vertex, map[ID]Face) {
	vertices := make(map[ID]Vertex, len(geometry.Vertices))
	faces := make(map[ID]Face, len(geometry.Faces))
	for _, vertex := range geometry.Vertices {
		vertices[vertex.ID] = vertex
	}
	for _, face := range geometry.Faces {
		faces[face.ID] = face
	}
	return vertices, faces
}

func requireIDs[T any](ids []ID, index map[ID]T, kind string) (map[ID]bool, error) {
	selected := map[ID]bool{}
	for _, id := range ids {
		if selected[id] {
			return nil, fmt.Errorf("%s %q is selected more than once", kind, id)
		}
		if _, ok := index[id]; !ok {
			return nil, fmt.Errorf("%s %q does not exist", kind, id)
		}
		selected[id] = true
	}
	return selected, nil
}

func addVec(a, b Vec3) Vec3 { return Vec3{X: a.X + b.X, Y: a.Y + b.Y, Z: a.Z + b.Z} }
func scaleVec(v Vec3, scale float64) Vec3 {
	return Vec3{X: v.X * scale, Y: v.Y * scale, Z: v.Z * scale}
}
func distanceVec(a, b Vec3) float64 {
	d := addVec(a, scaleVec(b, -1))
	return math.Sqrt(d.X*d.X + d.Y*d.Y + d.Z*d.Z)
}
func normalizeVec(v Vec3) Vec3 {
	length := math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
	if length == 0 {
		return Vec3{}
	}
	return scaleVec(v, 1/length)
}

func faceNormal(face Face, vertices map[ID]Vertex) Vec3 {
	a := vertices[face.Vertices[0]].Position
	b := vertices[face.Vertices[1]].Position
	c := vertices[face.Vertices[2]].Position
	u := Vec3{b.X - a.X, b.Y - a.Y, b.Z - a.Z}
	v := Vec3{c.X - a.X, c.Y - a.Y, c.Z - a.Z}
	n := Vec3{u.Y*v.Z - u.Z*v.Y, u.Z*v.X - u.X*v.Z, u.X*v.Y - u.Y*v.X}
	length := math.Sqrt(n.X*n.X + n.Y*n.Y + n.Z*n.Z)
	if length == 0 {
		return Vec3{Y: 1}
	}
	return Vec3{n.X / length, n.Y / length, n.Z / length}
}
func removeID(values []ID, target ID) []ID {
	out := values[:0]
	for _, value := range values {
		if value != target {
			out = append(out, value)
		}
	}
	return out
}

// bevelEdges replaces one shared interior edge with an inset quad. The floor
// implementation bevels exactly one stable edge whose endpoints belong only
// to the two adjacent faces; multi-edge bevels, segment counts, and vertex
// bevels remain explicitly partial.
func bevelEdges(geometry *Geometry, operation Operation) error {
	if len(operation.Edges) != 1 {
		return fmt.Errorf("bevel-edges currently requires exactly one stable edge id")
	}
	if operation.NewID == "" {
		return fmt.Errorf("bevel-edges requires newId for the bevel face")
	}
	if !(operation.Amount > 0) {
		return fmt.Errorf("bevel-edges requires a positive amount")
	}
	var target MeshEdge
	found := false
	for _, edge := range MeshEdges(*geometry) {
		if edge.ID == operation.Edges[0] {
			target, found = edge, true
			break
		}
	}
	if !found {
		return fmt.Errorf("edge %q does not exist", operation.Edges[0])
	}
	adjacent := make([]int, 0, 2)
	for index, face := range geometry.Faces {
		if faceContainsEdge(face, target.A, target.B) {
			adjacent = append(adjacent, index)
		}
	}
	if len(adjacent) != 2 {
		return fmt.Errorf("edge %q belongs to %d faces, want exactly two", target.ID, len(adjacent))
	}
	endpointUse := map[ID]bool{target.A: true, target.B: true}
	for index, face := range geometry.Faces {
		if index == adjacent[0] || index == adjacent[1] {
			continue
		}
		for _, vertex := range face.Vertices {
			if endpointUse[vertex] {
				return fmt.Errorf("bevel endpoint %q is shared with face %q; only two-face endpoints are supported", vertex, face.ID)
			}
		}
	}
	vertices, _ := topologyIndexes(*geometry)
	existing := map[ID]bool{}
	for _, vertex := range geometry.Vertices {
		existing[vertex.ID] = true
	}
	replacement := map[int]map[ID]ID{adjacent[0]: {}, adjacent[1]: {}}
	created := make([]Vertex, 0, 4)
	for _, faceIndex := range adjacent {
		face := geometry.Faces[faceIndex]
		for _, endpoint := range []ID{target.A, target.B} {
			other := target.B
			if endpoint == target.B {
				other = target.A
			}
			var neighbor ID
			for position, vertex := range face.Vertices {
				if vertex != endpoint {
					continue
				}
				before := face.Vertices[(position-1+len(face.Vertices))%len(face.Vertices)]
				after := face.Vertices[(position+1)%len(face.Vertices)]
				neighbor = before
				if neighbor == other {
					neighbor = after
				}
			}
			if neighbor == "" || neighbor == other {
				return fmt.Errorf("bevel endpoint %q has no offset direction in face %q", endpoint, face.ID)
			}
			from, to := vertices[endpoint], vertices[neighbor]
			length := distanceVec(from.Position, to.Position)
			if operation.Amount >= length {
				return fmt.Errorf("bevel amount %.6f must be smaller than adjacent segment %.6f", operation.Amount, length)
			}
			fraction := operation.Amount / length
			id := ID(fmt.Sprintf("bevel-%s-%s", endpoint, face.ID))
			if existing[id] {
				return fmt.Errorf("bevel vertex id %q already exists", id)
			}
			vertex := Vertex{ID: id, Position: lerpVec(from.Position, to.Position, fraction), Normal: lerpVec(from.Normal, to.Normal, fraction)}
			if from.UV != nil && to.UV != nil {
				vertex.UV = &Vec2{X: from.UV.X + (to.UV.X-from.UV.X)*fraction, Y: from.UV.Y + (to.UV.Y-from.UV.Y)*fraction}
			}
			created = append(created, vertex)
			existing[id] = true
			replacement[faceIndex][endpoint] = id
		}
	}
	for _, faceIndex := range adjacent {
		face := geometry.Faces[faceIndex]
		for position, vertex := range face.Vertices {
			if substitute, ok := replacement[faceIndex][vertex]; ok {
				face.Vertices[position] = substitute
			}
		}
		geometry.Faces[faceIndex] = face
	}
	kept := make([]Vertex, 0, len(geometry.Vertices)+2)
	for _, vertex := range geometry.Vertices {
		if vertex.ID != target.A && vertex.ID != target.B {
			kept = append(kept, vertex)
		}
	}
	geometry.Vertices = append(kept, created...)
	geometry.Faces = append(geometry.Faces, Face{ID: operation.NewID, Vertices: []ID{
		replacement[adjacent[0]][target.A], replacement[adjacent[0]][target.B],
		replacement[adjacent[1]][target.B], replacement[adjacent[1]][target.A],
	}})
	return nil
}
