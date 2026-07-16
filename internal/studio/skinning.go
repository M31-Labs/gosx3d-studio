package studio

import (
	"fmt"
	"math"
)

type SkinDeformationReport struct {
	Entity        ID      `json:"entity"`
	Armature      ID      `json:"armature"`
	Vertices      int     `json:"vertices"`
	Influences    int     `json:"influences"`
	MovedVertices int     `json:"movedVertices"`
	MaximumDelta  float64 `json:"maximumDelta"`
}

type matrix4 [16]float64

// DeformSkinnedGeometry evaluates linear blend skinning from the armature's
// hierarchical rest and pose transforms. Stable vertex IDs, faces, UVs, and
// attributes are preserved; only positions and authored normals are deformed.
func DeformSkinnedGeometry(document Document, entityID ID) (Geometry, SkinDeformationReport, error) {
	entity, ok := document.Entities[entityID]
	if !ok || entity.Mesh == nil || entity.Skin == nil {
		return Geometry{}, SkinDeformationReport{}, fmt.Errorf("entity %q is not a skinned mesh", entityID)
	}
	return deformSkinGeometry(document, entity, entityID)
}

func deformSkinGeometry(document Document, entity Entity, runtimeID ID) (Geometry, SkinDeformationReport, error) {
	if len(entity.Mesh.Modifiers) != 0 {
		return Geometry{}, SkinDeformationReport{}, fmt.Errorf("entity %q skin/modifier ordering is not implemented", runtimeID)
	}
	armature := document.Armatures[entity.Skin.Armature]
	restGlobals, err := armatureGlobalMatrices(armature, false)
	if err != nil {
		return Geometry{}, SkinDeformationReport{}, err
	}
	poseGlobals, err := armatureGlobalMatrices(armature, true)
	if err != nil {
		return Geometry{}, SkinDeformationReport{}, err
	}
	deltas := make(map[ID]matrix4, len(armature.Bones))
	for boneID := range armature.Bones {
		inverse, ok := inverseMatrix(restGlobals[boneID])
		if !ok {
			return Geometry{}, SkinDeformationReport{}, fmt.Errorf("bone %q rest transform is singular", boneID)
		}
		deltas[boneID] = multiplyMatrix(poseGlobals[boneID], inverse)
	}
	geometry := entity.Mesh.Geometry
	geometry.Vertices = append([]Vertex(nil), geometry.Vertices...)
	report := SkinDeformationReport{Entity: runtimeID, Armature: armature.ID, Vertices: len(geometry.Vertices)}
	for index := range geometry.Vertices {
		vertex := &geometry.Vertices[index]
		originalPosition, originalNormal := vertex.Position, vertex.Normal
		position, normal := Vec3{}, Vec3{}
		for _, influence := range entity.Skin.Weights[vertex.ID] {
			delta := deltas[influence.Bone]
			position = addVec(position, scaleVec(transformPoint(delta, originalPosition), influence.Weight))
			if originalNormal != (Vec3{}) {
				normal = addVec(normal, scaleVec(transformNormal(delta, originalNormal), influence.Weight))
			}
			report.Influences++
		}
		vertex.Position = position
		if originalNormal != (Vec3{}) {
			vertex.Normal = normalizeVec(normal)
		}
		delta := vectorLength(addVec(position, scaleVec(originalPosition, -1)))
		if delta > 1e-9 {
			report.MovedVertices++
		}
		if delta > report.MaximumDelta {
			report.MaximumDelta = delta
		}
	}
	return geometry, report, nil
}

func armatureGlobalMatrices(armature Armature, posed bool) (map[ID]matrix4, error) {
	globals := make(map[ID]matrix4, len(armature.Bones))
	var walk func(ID, matrix4) error
	walk = func(id ID, parent matrix4) error {
		bone, ok := armature.Bones[id]
		if !ok {
			return fmt.Errorf("armature %q references missing bone %q", armature.ID, id)
		}
		transform := bone.Rest
		if posed {
			if value, ok := armature.Pose[id]; ok {
				transform = value
			}
		}
		global := multiplyMatrix(parent, transformMatrix(transform))
		globals[id] = global
		for _, child := range bone.Children {
			if err := walk(child, global); err != nil {
				return err
			}
		}
		return nil
	}
	for _, root := range armature.RootBones {
		if err := walk(root, identityMatrix()); err != nil {
			return nil, err
		}
	}
	return globals, nil
}

func identityMatrix() matrix4 { return matrix4{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1} }

func transformMatrix(value Transform) matrix4 {
	rotation := value.canonical().Rotation.rotationMatrix()
	scale := matrix4{value.Scale.X, 0, 0, 0, 0, value.Scale.Y, 0, 0, 0, 0, value.Scale.Z, 0, 0, 0, 0, 1}
	result := multiplyMatrix(rotation, scale)
	result[3], result[7], result[11] = value.Position.X, value.Position.Y, value.Position.Z
	return result
}

func multiplyMatrix(a, b matrix4) matrix4 {
	var result matrix4
	for row := 0; row < 4; row++ {
		for column := 0; column < 4; column++ {
			for index := 0; index < 4; index++ {
				result[row*4+column] += a[row*4+index] * b[index*4+column]
			}
		}
	}
	return result
}

func inverseMatrix(input matrix4) (matrix4, bool) {
	var augmented [4][8]float64
	for row := 0; row < 4; row++ {
		for column := 0; column < 4; column++ {
			augmented[row][column] = input[row*4+column]
		}
		augmented[row][row+4] = 1
	}
	for column := 0; column < 4; column++ {
		pivot := column
		for row := column + 1; row < 4; row++ {
			if math.Abs(augmented[row][column]) > math.Abs(augmented[pivot][column]) {
				pivot = row
			}
		}
		if math.Abs(augmented[pivot][column]) < 1e-12 {
			return matrix4{}, false
		}
		augmented[column], augmented[pivot] = augmented[pivot], augmented[column]
		value := augmented[column][column]
		for index := 0; index < 8; index++ {
			augmented[column][index] /= value
		}
		for row := 0; row < 4; row++ {
			if row == column {
				continue
			}
			factor := augmented[row][column]
			for index := 0; index < 8; index++ {
				augmented[row][index] -= factor * augmented[column][index]
			}
		}
	}
	var result matrix4
	for row := 0; row < 4; row++ {
		for column := 0; column < 4; column++ {
			result[row*4+column] = augmented[row][column+4]
		}
	}
	return result, true
}

func transformPoint(matrix matrix4, value Vec3) Vec3 {
	return Vec3{X: matrix[0]*value.X + matrix[1]*value.Y + matrix[2]*value.Z + matrix[3], Y: matrix[4]*value.X + matrix[5]*value.Y + matrix[6]*value.Z + matrix[7], Z: matrix[8]*value.X + matrix[9]*value.Y + matrix[10]*value.Z + matrix[11]}
}

func transformNormal(matrix matrix4, value Vec3) Vec3 {
	inverse, ok := inverseMatrix(matrix)
	if !ok {
		return Vec3{}
	}
	return normalizeVec(Vec3{X: inverse[0]*value.X + inverse[4]*value.Y + inverse[8]*value.Z, Y: inverse[1]*value.X + inverse[5]*value.Y + inverse[9]*value.Z, Z: inverse[2]*value.X + inverse[6]*value.Y + inverse[10]*value.Z})
}
