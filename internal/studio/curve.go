package studio

import (
	"fmt"
	"math"

	"m31labs.dev/gosx/scene"
)

type CurveAnalysis struct {
	Schema              string  `json:"schema"`
	Entity              ID      `json:"entity"`
	Revision            uint64  `json:"revision"`
	Degree              int     `json:"degree"`
	ControlPoints       int     `json:"controlPoints"`
	ParameterStart      float64 `json:"parameterStart"`
	ParameterEnd        float64 `json:"parameterEnd"`
	ApproximateLength   float64 `json:"approximateLength"`
	TessellatedVertices int     `json:"tessellatedVertices"`
	Triangles           int     `json:"triangles"`
	Valid               bool    `json:"valid"`
}

func AnalyzeEntityCurve(document Document, entityID ID) (CurveAnalysis, error) {
	entity, ok := document.Entities[entityID]
	if !ok || entity.Mesh == nil || entity.Mesh.Geometry.Kind != "nurbs-curve" || entity.Mesh.Geometry.Curve == nil {
		return CurveAnalysis{}, fmt.Errorf("curve analysis target %q is not a nurbs curve", entityID)
	}
	curve := entity.Mesh.Geometry.Curve
	if err := validateCurveGeometry(curve); err != nil {
		return CurveAnalysis{}, err
	}
	analysis := CurveAnalysis{Schema: "gosx3d.studio.curve-analysis/v1", Entity: entityID, Revision: document.Revision, Degree: curve.Degree, ControlPoints: len(curve.ControlPoints), ParameterStart: curve.Knots[curve.Degree], ParameterEnd: curve.Knots[len(curve.ControlPoints)], TessellatedVertices: (curve.Segments + 1) * curve.RadialSegments, Triangles: curve.Segments * curve.RadialSegments * 2, Valid: true}
	previous := evaluateNURBS(*curve, analysis.ParameterStart)
	for index := 1; index <= curve.Segments; index++ {
		parameter := analysis.ParameterStart + (analysis.ParameterEnd-analysis.ParameterStart)*float64(index)/float64(curve.Segments)
		current := evaluateNURBS(*curve, parameter)
		analysis.ApproximateLength += distanceVec(previous, current)
		previous = current
	}
	return analysis, nil
}

func applySetCurveControlPoint(document *Document, operation Operation) ([]ID, error) {
	entity, ok := document.Entities[operation.Target]
	if !ok || entity.Mesh == nil || entity.Mesh.Geometry.Kind != "nurbs-curve" || entity.Mesh.Geometry.Curve == nil {
		return nil, fmt.Errorf("set-curve-control-point target %q is not a nurbs curve", operation.Target)
	}
	if entity.Locked {
		return nil, fmt.Errorf("entity %q is locked", operation.Target)
	}
	if operation.ControlPoint == "" || (operation.Position == nil && operation.Weight == nil) {
		return nil, fmt.Errorf("set-curve-control-point requires controlPoint and position or weight")
	}
	curve := entity.Mesh.Geometry.Curve
	found := false
	for index := range curve.ControlPoints {
		if curve.ControlPoints[index].ID != operation.ControlPoint {
			continue
		}
		found = true
		if operation.Position != nil {
			if !finiteVec3(*operation.Position) {
				return nil, fmt.Errorf("curve control-point position must be finite")
			}
			curve.ControlPoints[index].Position = *operation.Position
		}
		if operation.Weight != nil {
			if !finite(*operation.Weight) || *operation.Weight <= 0 {
				return nil, fmt.Errorf("curve control-point weight must be finite and positive")
			}
			curve.ControlPoints[index].Weight = *operation.Weight
		}
		break
	}
	if !found {
		return nil, fmt.Errorf("curve control point %q does not exist", operation.ControlPoint)
	}
	document.Entities[entity.ID] = entity
	return []ID{entity.ID}, nil
}

func compileNURBSCurve(curve *CurveGeometry) (scene.Geometry, error) {
	if err := validateCurveGeometry(curve); err != nil {
		return nil, err
	}
	path := make([]Vec3, curve.Segments+1)
	start := curve.Knots[curve.Degree]
	end := curve.Knots[len(curve.ControlPoints)]
	for index := range path {
		u := start + (end-start)*float64(index)/float64(curve.Segments)
		path[index] = evaluateNURBS(*curve, u)
	}
	positions := make([]float64, 0, len(path)*curve.RadialSegments*3)
	normals := make([]float64, 0, cap(positions))
	uvs := make([]float64, 0, len(path)*curve.RadialSegments*2)
	for index, point := range path {
		previous := path[maxInt(0, index-1)]
		next := path[minInt(len(path)-1, index+1)]
		tangent := normalizeVec(addVec(next, scaleVec(previous, -1)))
		if tangent == (Vec3{}) {
			return nil, fmt.Errorf("nurbs-curve has a zero-length tangent at sample %d", index)
		}
		reference := Vec3{Y: 1}
		if math.Abs(tangent.Y) > 0.9 {
			reference = Vec3{X: 1}
		}
		normal := normalizeVec(crossVec(tangent, reference))
		binormal := normalizeVec(crossVec(tangent, normal))
		for radial := 0; radial < curve.RadialSegments; radial++ {
			angle := 2 * math.Pi * float64(radial) / float64(curve.RadialSegments)
			direction := addVec(scaleVec(normal, math.Cos(angle)), scaleVec(binormal, math.Sin(angle)))
			position := addVec(point, scaleVec(direction, curve.Radius))
			positions = append(positions, position.X, position.Y, position.Z)
			normals = append(normals, direction.X, direction.Y, direction.Z)
			uvs = append(uvs, float64(index)/float64(curve.Segments), float64(radial)/float64(curve.RadialSegments))
		}
	}
	indices := make([]int, 0, curve.Segments*curve.RadialSegments*6)
	for pathIndex := 0; pathIndex < curve.Segments; pathIndex++ {
		for radial := 0; radial < curve.RadialSegments; radial++ {
			nextRadial := (radial + 1) % curve.RadialSegments
			a := pathIndex*curve.RadialSegments + radial
			b := pathIndex*curve.RadialSegments + nextRadial
			c := (pathIndex+1)*curve.RadialSegments + nextRadial
			d := (pathIndex+1)*curve.RadialSegments + radial
			indices = append(indices, a, b, c, a, c, d)
		}
	}
	return scene.BufferGeometry{Positions: positions, Normals: normals, UVs: uvs, Indices: indices}, nil
}

func evaluateNURBS(curve CurveGeometry, parameter float64) Vec3 {
	numerator := Vec3{}
	denominator := 0.0
	last := len(curve.ControlPoints) - 1
	for index, point := range curve.ControlPoints {
		basis := nurbsBasis(index, curve.Degree, parameter, curve.Knots, last)
		weighted := basis * point.Weight
		numerator = addVec(numerator, scaleVec(point.Position, weighted))
		denominator += weighted
	}
	if denominator == 0 {
		return Vec3{}
	}
	return scaleVec(numerator, 1/denominator)
}

func nurbsBasis(index, degree int, parameter float64, knots []float64, lastControl int) float64 {
	if degree == 0 {
		if (knots[index] <= parameter && parameter < knots[index+1]) || (index == lastControl && parameter == knots[lastControl+1]) {
			return 1
		}
		return 0
	}
	left, right := 0.0, 0.0
	leftDenominator := knots[index+degree] - knots[index]
	if leftDenominator != 0 {
		left = (parameter - knots[index]) / leftDenominator * nurbsBasis(index, degree-1, parameter, knots, lastControl)
	}
	rightDenominator := knots[index+degree+1] - knots[index+1]
	if rightDenominator != 0 {
		right = (knots[index+degree+1] - parameter) / rightDenominator * nurbsBasis(index+1, degree-1, parameter, knots, lastControl)
	}
	return left + right
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}
func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
