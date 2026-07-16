package studio

import (
	"math"
	"testing"
)

func rotateByMatrix(t *testing.T, rotation Vec3, input Vec3) Vec3 {
	t.Helper()
	m := transformMatrix(Transform{Rotation: QuaternionFromEuler(rotation), Euler: rotation, Scale: Vec3{X: 1, Y: 1, Z: 1}})
	return Vec3{
		X: m[0]*input.X + m[1]*input.Y + m[2]*input.Z,
		Y: m[4]*input.X + m[5]*input.Y + m[6]*input.Z,
		Z: m[8]*input.X + m[9]*input.Y + m[10]*input.Z,
	}
}

func vecNear(t *testing.T, got, want Vec3, tolerance float64, context string) {
	t.Helper()
	if math.Abs(got.X-want.X) > tolerance || math.Abs(got.Y-want.Y) > tolerance || math.Abs(got.Z-want.Z) > tolerance {
		t.Fatalf("%s: got %+v want %+v", context, got, want)
	}
}

func TestQuaternionFromEulerIdentity(t *testing.T) {
	q := QuaternionFromEuler(Vec3{})
	if q != (Quaternion{W: 1}) {
		t.Fatalf("identity euler must produce identity quaternion, got %+v", q)
	}
	if !q.IsIdentity() {
		t.Fatalf("identity quaternion must report IsIdentity")
	}
}

func TestZeroQuaternionCanonicalizesToIdentity(t *testing.T) {
	var zero Quaternion
	if got := zero.Normalized(); got != (Quaternion{W: 1}) {
		t.Fatalf("zero quaternion must normalize to identity, got %+v", got)
	}
	if !zero.IsIdentity() {
		t.Fatalf("zero quaternion must count as identity for legacy literals")
	}
}

func TestQuaternionFromEulerMatchesEngineOrder(t *testing.T) {
	// The engine composes Rz*Ry*Rx (X applied first). Rotating +X by 90deg
	// around Z must land on +Y under that convention.
	q := QuaternionFromEuler(Vec3{Z: math.Pi / 2})
	vecNear(t, q.Rotate(Vec3{X: 1}), Vec3{Y: 1}, 1e-12, "Rz(90) on +X")

	// Multi-axis: quaternion rotation must agree with the skinning matrix
	// composition for arbitrary angles.
	cases := []Vec3{
		{X: 0.35},
		{Y: -1.2},
		{Z: 2.6},
		{X: 0.4, Y: 0.7, Z: -1.1},
		{X: -2.9, Y: 1.4, Z: 0.2},
	}
	inputs := []Vec3{{X: 1}, {Y: 1}, {Z: 1}, {X: 0.3, Y: -0.5, Z: 0.8}}
	for _, rotation := range cases {
		q := QuaternionFromEuler(rotation)
		for _, input := range inputs {
			vecNear(t, q.Rotate(input), rotateByMatrix(t, rotation, input), 1e-9, "quaternion vs matrix")
		}
	}
}

func TestEulerFromQuaternionRoundTrip(t *testing.T) {
	cases := []Vec3{
		{},
		{Z: 0.35},
		{X: 1.1},
		{Y: -0.9},
		{X: 0.4, Y: 0.7, Z: -1.1},
		{Y: math.Pi / 2}, // gimbal-lock branch
	}
	inputs := []Vec3{{X: 1}, {Y: 1}, {Z: 1}}
	for _, rotation := range cases {
		q := QuaternionFromEuler(rotation)
		back := QuaternionFromEuler(q.Euler())
		for _, input := range inputs {
			vecNear(t, back.Rotate(input), q.Rotate(input), 1e-9, "euler round trip")
		}
	}
}

func TestQuaternionSlerp(t *testing.T) {
	a := QuaternionFromEuler(Vec3{Z: 0})
	b := QuaternionFromEuler(Vec3{Z: 1.0})
	if got := Slerp(a, b, 0); got != a.Normalized() {
		t.Fatalf("slerp t=0 must return start, got %+v", got)
	}
	if got := Slerp(a, b, 1); got != b.Normalized() {
		t.Fatalf("slerp t=1 must return end, got %+v", got)
	}
	mid := Slerp(a, b, 0.5)
	want := QuaternionFromEuler(Vec3{Z: 0.5})
	vecNear(t, mid.Rotate(Vec3{X: 1}), want.Rotate(Vec3{X: 1}), 1e-12, "single-axis slerp midpoint")

	// Shortest path: interpolation toward a negated-dot quaternion must not
	// swing the long way around.
	c := QuaternionFromEuler(Vec3{Z: 0.2})
	negated := Quaternion{X: -c.X, Y: -c.Y, Z: -c.Z, W: -c.W}
	shortest := Slerp(a, negated, 0.5)
	wantShort := QuaternionFromEuler(Vec3{Z: 0.1})
	vecNear(t, shortest.Rotate(Vec3{X: 1}), wantShort.Rotate(Vec3{X: 1}), 1e-12, "shortest-path slerp")
}
