package studio

import "math"

// Quaternion is the authoritative rotation representation for SceneDoc
// transforms. The zero value is treated as identity so legacy documents and
// in-code literals that never set a rotation stay valid; JSON encode/decode
// canonicalizes zero to the explicit identity form.
//
// Euler conversion mirrors the engine's composition exactly (Rz*Ry*Rx with X
// applied first) so Studio math, skinning matrices, and scene lowering agree.
type Quaternion struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
	W float64 `json:"w"`
}

func identityQuaternion() Quaternion { return Quaternion{W: 1} }

// QuaternionFromEuler composes axis rotations as Rz*Ry*Rx, matching both the
// skinning matrix composition and the engine's quaternionFromEuler.
func QuaternionFromEuler(rotation Vec3) Quaternion {
	qx := axisAngleQuaternion(Vec3{X: 1}, rotation.X)
	qy := axisAngleQuaternion(Vec3{Y: 1}, rotation.Y)
	qz := axisAngleQuaternion(Vec3{Z: 1}, rotation.Z)
	return qz.Mul(qy).Mul(qx).Normalized()
}

func axisAngleQuaternion(axis Vec3, angle float64) Quaternion {
	if angle == 0 {
		return identityQuaternion()
	}
	half := angle / 2
	sine := math.Sin(half)
	return Quaternion{X: axis.X * sine, Y: axis.Y * sine, Z: axis.Z * sine, W: math.Cos(half)}
}

func (q Quaternion) Normalized() Quaternion {
	length := math.Sqrt(q.X*q.X + q.Y*q.Y + q.Z*q.Z + q.W*q.W)
	if length == 0 {
		return identityQuaternion()
	}
	return Quaternion{X: q.X / length, Y: q.Y / length, Z: q.Z / length, W: q.W / length}
}

// IsIdentity reports whether the quaternion encodes no rotation. The zero
// value counts as identity for legacy literals and documents.
func (q Quaternion) IsIdentity() bool {
	const tolerance = 1e-12
	if math.Abs(q.X) > tolerance || math.Abs(q.Y) > tolerance || math.Abs(q.Z) > tolerance {
		return false
	}
	return q.W == 0 || math.Abs(math.Abs(q.W)-1) <= tolerance
}

func (q Quaternion) Conjugate() Quaternion { return Quaternion{X: -q.X, Y: -q.Y, Z: -q.Z, W: q.W} }

// Inverse returns the rotation inverse, normalizing first so the zero value
// and denormalized inputs behave as their canonical rotations.
func (q Quaternion) Inverse() Quaternion { return q.Normalized().Conjugate() }

func (q Quaternion) Mul(other Quaternion) Quaternion {
	return Quaternion{
		X: q.W*other.X + q.X*other.W + q.Y*other.Z - q.Z*other.Y,
		Y: q.W*other.Y - q.X*other.Z + q.Y*other.W + q.Z*other.X,
		Z: q.W*other.Z + q.X*other.Y - q.Y*other.X + q.Z*other.W,
		W: q.W*other.W - q.X*other.X - q.Y*other.Y - q.Z*other.Z,
	}
}

func (q Quaternion) Rotate(vector Vec3) Vec3 {
	unit := q.Normalized()
	point := Quaternion{X: vector.X, Y: vector.Y, Z: vector.Z}
	result := unit.Mul(point).Mul(unit.Conjugate())
	return Vec3{X: result.X, Y: result.Y, Z: result.Z}
}

// Euler extracts angles for the Rz*Ry*Rx convention, mirroring the engine's
// eulerFromQuaternion including its gimbal-lock branch.
func (q Quaternion) Euler() Vec3 {
	unit := q.Normalized()
	m00 := 1 - 2*(unit.Y*unit.Y+unit.Z*unit.Z)
	m10 := 2 * (unit.X*unit.Y + unit.Z*unit.W)
	m20 := 2 * (unit.X*unit.Z - unit.Y*unit.W)
	m21 := 2 * (unit.Y*unit.Z + unit.X*unit.W)
	m22 := 1 - 2*(unit.X*unit.X+unit.Y*unit.Y)
	m01 := 2 * (unit.X*unit.Y - unit.Z*unit.W)
	m11 := 1 - 2*(unit.X*unit.X+unit.Z*unit.Z)

	y := math.Asin(clampFloat(-m20, -1, 1))
	if math.Abs(math.Cos(y)) > 1e-9 {
		return Vec3{X: math.Atan2(m21, m22), Y: y, Z: math.Atan2(m10, m00)}
	}
	return Vec3{Y: y, Z: math.Atan2(-m01, m11)}
}

// rotationMatrix expands the quaternion into the upper-left 3x3 of a matrix4.
func (q Quaternion) rotationMatrix() matrix4 {
	unit := q.Normalized()
	x, y, z, w := unit.X, unit.Y, unit.Z, unit.W
	return matrix4{
		1 - 2*(y*y+z*z), 2 * (x*y - z*w), 2 * (x*z + y*w), 0,
		2 * (x*y + z*w), 1 - 2*(x*x+z*z), 2 * (y*z - x*w), 0,
		2 * (x*z - y*w), 2 * (y*z + x*w), 1 - 2*(x*x+y*y), 0,
		0, 0, 0, 1,
	}
}

// Slerp interpolates along the shortest arc. Endpoints return the normalized
// inputs exactly so deterministic sampling at key times stays byte-stable.
func Slerp(a, b Quaternion, t float64) Quaternion {
	start := a.Normalized()
	end := b.Normalized()
	dot := start.X*end.X + start.Y*end.Y + start.Z*end.Z + start.W*end.W
	if dot < 0 {
		end = Quaternion{X: -end.X, Y: -end.Y, Z: -end.Z, W: -end.W}
		dot = -dot
	}
	if t <= 0 {
		return start
	}
	if t >= 1 {
		return end
	}
	if dot > 0.9995 {
		return Quaternion{
			X: start.X + t*(end.X-start.X),
			Y: start.Y + t*(end.Y-start.Y),
			Z: start.Z + t*(end.Z-start.Z),
			W: start.W + t*(end.W-start.W),
		}.Normalized()
	}
	theta := math.Acos(clampFloat(dot, -1, 1))
	sinTheta := math.Sin(theta)
	scaleStart := math.Sin((1-t)*theta) / sinTheta
	scaleEnd := math.Sin(t*theta) / sinTheta
	return Quaternion{
		X: scaleStart*start.X + scaleEnd*end.X,
		Y: scaleStart*start.Y + scaleEnd*end.Y,
		Z: scaleStart*start.Z + scaleEnd*end.Z,
		W: scaleStart*start.W + scaleEnd*end.W,
	}.Normalized()
}

func clampFloat(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
