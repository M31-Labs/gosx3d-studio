package studio

import (
	"fmt"
	"math"
	"sort"
)

type csgTriangle struct{ A, B, C Vec3 }
type voxelCell struct{ X, Y, Z int }

// csgVoxelBudget caps the voxel grid one boolean may evaluate. The evaluation
// runs inside the workspace write lock, so an unbounded grid does not merely
// take a long time: it never returns the lock, and every later request waits
// on it forever.
const csgVoxelBudget = 1_000_000

func applyCSGBoolean(document *Document, operation Operation) ([]ID, error) {
	if operation.NewID == "" || operation.Operand == "" || operation.Target == operation.Operand {
		return nil, fmt.Errorf("csg-boolean requires distinct target and operand plus newId")
	}
	if operation.Boolean != "union" && operation.Boolean != "intersection" && operation.Boolean != "subtract" {
		return nil, fmt.Errorf("csg-boolean requires boolean union, intersection, or subtract")
	}
	if !finite(operation.VoxelSize) || operation.VoxelSize <= 0 {
		return nil, fmt.Errorf("csg-boolean requires positive finite voxelSize")
	}
	if _, exists := document.Entities[operation.NewID]; exists {
		return nil, fmt.Errorf("entity %q already exists", operation.NewID)
	}
	left, ok := document.Entities[operation.Target]
	if !ok {
		return nil, fmt.Errorf("csg target %q does not exist", operation.Target)
	}
	right, ok := document.Entities[operation.Operand]
	if !ok {
		return nil, fmt.Errorf("csg operand %q does not exist", operation.Operand)
	}
	if left.Mesh == nil || right.Mesh == nil || left.Mesh.Geometry.Kind != "indexed-mesh" || right.Mesh.Geometry.Kind != "indexed-mesh" {
		return nil, fmt.Errorf("csg-boolean currently requires indexed-mesh operands")
	}
	if left.Parent != right.Parent {
		return nil, fmt.Errorf("csg operands must share a parent")
	}
	if !left.Transform.Rotation.IsIdentity() || !right.Transform.Rotation.IsIdentity() || !unitScale(left.Transform.Scale) || !unitScale(right.Transform.Scale) {
		return nil, fmt.Errorf("csg operands currently require identity rotation and scale")
	}
	leftGeometry, err := evaluateModifiers(left.Mesh.Geometry, left.Mesh.Modifiers)
	if err != nil {
		return nil, err
	}
	rightGeometry, err := evaluateModifiers(right.Mesh.Geometry, right.Mesh.Modifiers)
	if err != nil {
		return nil, err
	}
	leftAnalysis, rightAnalysis := analyzeGeometry(leftGeometry), analyzeGeometry(rightGeometry)
	if !leftAnalysis.Valid || !leftAnalysis.Closed || !rightAnalysis.Valid || !rightAnalysis.Closed {
		return nil, fmt.Errorf("csg operands must be valid closed manifold meshes")
	}
	leftTriangles, leftMin, leftMax := csgTriangles(leftGeometry, left.Transform.Position)
	rightTriangles, rightMin, rightMax := csgTriangles(rightGeometry, right.Transform.Position)
	minimum := minVec(leftMin, rightMin)
	maximum := maxVec(leftMax, rightMax)
	size := operation.VoxelSize
	origin := Vec3{X: math.Floor(minimum.X/size) * size, Y: math.Floor(minimum.Y/size) * size, Z: math.Floor(minimum.Z/size) * size}
	// Count cells in float64 before converting. A large span over a small
	// voxel size makes each axis exceed two billion, and the int64 product of
	// three such axes wraps: the guard then read a small number and admitted a
	// grid of 1e28 cells. The loop below runs inside the workspace write lock,
	// so admitting one never returns the lock and wedges every later request.
	// float64 has no wraparound, and this stays exact far past the budget.
	cells := math.Ceil((maximum.X-origin.X)/size) * math.Ceil((maximum.Y-origin.Y)/size) * math.Ceil((maximum.Z-origin.Z)/size)
	if !finite(cells) || cells <= 0 || cells > csgVoxelBudget {
		return nil, fmt.Errorf("csg voxel budget exceeds %d cells (%.0f requested)", csgVoxelBudget, cells)
	}
	nx := int(math.Ceil((maximum.X - origin.X) / size))
	ny := int(math.Ceil((maximum.Y - origin.Y) / size))
	nz := int(math.Ceil((maximum.Z - origin.Z) / size))
	if nx <= 0 || ny <= 0 || nz <= 0 {
		return nil, fmt.Errorf("csg voxel grid is empty (%dx%dx%d)", nx, ny, nz)
	}
	occupied := map[voxelCell]bool{}
	for x := 0; x < nx; x++ {
		for y := 0; y < ny; y++ {
			for z := 0; z < nz; z++ {
				point := Vec3{X: origin.X + (float64(x)+0.5)*size, Y: origin.Y + (float64(y)+0.5)*size, Z: origin.Z + (float64(z)+0.5)*size}
				insideLeft := pointInsideMesh(point, leftTriangles)
				insideRight := pointInsideMesh(point, rightTriangles)
				keep := false
				switch operation.Boolean {
				case "union":
					keep = insideLeft || insideRight
				case "intersection":
					keep = insideLeft && insideRight
				case "subtract":
					keep = insideLeft && !insideRight
				}
				if keep {
					occupied[voxelCell{x, y, z}] = true
				}
			}
		}
	}
	if len(occupied) == 0 {
		return nil, fmt.Errorf("csg %s produced an empty result", operation.Boolean)
	}
	geometry := voxelSurface(operation.NewID, origin, size, occupied)
	if err := validateGeometry(geometry); err != nil {
		return nil, fmt.Errorf("csg result: %w", err)
	}
	result := Entity{ID: operation.NewID, Name: fmt.Sprintf("%s %s %s", left.Name, operation.Boolean, right.Name), Parent: left.Parent, Transform: IdentityTransform(), Visible: true, Mesh: &MeshComponent{Geometry: geometry, Material: left.Mesh.Material, Pickable: true, CastShadow: left.Mesh.CastShadow, ReceiveShadow: left.Mesh.ReceiveShadow}}
	if result.Parent == "" {
		document.RootIDs = append(document.RootIDs, result.ID)
	} else {
		parent := document.Entities[result.Parent]
		parent.Children = append(parent.Children, result.ID)
		document.Entities[parent.ID] = parent
	}
	document.Entities[result.ID] = result
	return []ID{left.ID, right.ID, result.ID}, nil
}

func csgTriangles(geometry Geometry, translation Vec3) ([]csgTriangle, Vec3, Vec3) {
	vertices := map[ID]Vec3{}
	minimum := Vec3{X: math.Inf(1), Y: math.Inf(1), Z: math.Inf(1)}
	maximum := Vec3{X: math.Inf(-1), Y: math.Inf(-1), Z: math.Inf(-1)}
	for _, vertex := range geometry.Vertices {
		position := addVec(vertex.Position, translation)
		vertices[vertex.ID] = position
		minimum = minVec(minimum, position)
		maximum = maxVec(maximum, position)
	}
	triangles := []csgTriangle{}
	for _, face := range geometry.Faces {
		a := vertices[face.Vertices[0]]
		for index := 1; index+1 < len(face.Vertices); index++ {
			triangles = append(triangles, csgTriangle{a, vertices[face.Vertices[index]], vertices[face.Vertices[index+1]]})
		}
	}
	return triangles, minimum, maximum
}

func pointInsideMesh(point Vec3, triangles []csgTriangle) bool {
	direction := normalizeVec(Vec3{X: 1, Y: 0.3713906763541037, Z: 0.529103782122995})
	hits := []float64{}
	for _, triangle := range triangles {
		if distance, ok := rayTriangle(point, direction, triangle); ok {
			hits = append(hits, distance)
		}
	}
	sort.Float64s(hits)
	unique := 0
	previous := math.Inf(-1)
	for _, hit := range hits {
		if math.Abs(hit-previous) > 1e-8 {
			unique++
			previous = hit
		}
	}
	return unique%2 == 1
}

func rayTriangle(origin, direction Vec3, triangle csgTriangle) (float64, bool) {
	edge1 := addVec(triangle.B, scaleVec(triangle.A, -1))
	edge2 := addVec(triangle.C, scaleVec(triangle.A, -1))
	h := crossVec(direction, edge2)
	det := dotVec(edge1, h)
	if math.Abs(det) < 1e-10 {
		return 0, false
	}
	inverse := 1 / det
	s := addVec(origin, scaleVec(triangle.A, -1))
	u := inverse * dotVec(s, h)
	if u < 0 || u > 1 {
		return 0, false
	}
	q := crossVec(s, edge1)
	v := inverse * dotVec(direction, q)
	if v < 0 || u+v > 1 {
		return 0, false
	}
	distance := inverse * dotVec(edge2, q)
	return distance, distance > 1e-9
}

func voxelSurface(prefix ID, origin Vec3, size float64, occupied map[voxelCell]bool) Geometry {
	cells := make([]voxelCell, 0, len(occupied))
	for cell := range occupied {
		cells = append(cells, cell)
	}
	sort.Slice(cells, func(i, j int) bool {
		if cells[i].X != cells[j].X {
			return cells[i].X < cells[j].X
		}
		if cells[i].Y != cells[j].Y {
			return cells[i].Y < cells[j].Y
		}
		return cells[i].Z < cells[j].Z
	})
	type direction struct {
		DX, DY, DZ int
		Name       string
		Corners    [4][3]int
	}
	directions := []direction{{-1, 0, 0, "nx", [4][3]int{{0, 0, 0}, {0, 0, 1}, {0, 1, 1}, {0, 1, 0}}}, {1, 0, 0, "px", [4][3]int{{1, 0, 0}, {1, 1, 0}, {1, 1, 1}, {1, 0, 1}}}, {0, -1, 0, "ny", [4][3]int{{0, 0, 0}, {1, 0, 0}, {1, 0, 1}, {0, 0, 1}}}, {0, 1, 0, "py", [4][3]int{{0, 1, 0}, {0, 1, 1}, {1, 1, 1}, {1, 1, 0}}}, {0, 0, -1, "nz", [4][3]int{{0, 0, 0}, {0, 1, 0}, {1, 1, 0}, {1, 0, 0}}}, {0, 0, 1, "pz", [4][3]int{{0, 0, 1}, {1, 0, 1}, {1, 1, 1}, {0, 1, 1}}}}
	geometry := Geometry{Kind: "indexed-mesh"}
	vertexIDs := map[[3]int]ID{}
	for _, cell := range cells {
		for _, faceDirection := range directions {
			if occupied[voxelCell{cell.X + faceDirection.DX, cell.Y + faceDirection.DY, cell.Z + faceDirection.DZ}] {
				continue
			}
			face := Face{ID: ID(fmt.Sprintf("%s--cell-%d-%d-%d--%s", prefix, cell.X, cell.Y, cell.Z, faceDirection.Name))}
			for _, offset := range faceDirection.Corners {
				corner := [3]int{cell.X + offset[0], cell.Y + offset[1], cell.Z + offset[2]}
				id, ok := vertexIDs[corner]
				if !ok {
					id = ID(fmt.Sprintf("%s--grid-%d-%d-%d", prefix, corner[0], corner[1], corner[2]))
					vertexIDs[corner] = id
					geometry.Vertices = append(geometry.Vertices, Vertex{ID: id, Position: Vec3{X: origin.X + float64(corner[0])*size, Y: origin.Y + float64(corner[1])*size, Z: origin.Z + float64(corner[2])*size}})
				}
				face.Vertices = append(face.Vertices, id)
			}
			geometry.Faces = append(geometry.Faces, face)
		}
	}
	return geometry
}
