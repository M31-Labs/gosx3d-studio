package studio

import (
	"fmt"
	"math"
	"testing"
	"time"
)

// Parent and child consistency had no test coverage at all, which mattered
// when the "does my parent list me?" check moved from scanning the parent's
// slice to a precomputed reverse index. These pin the rejections themselves,
// so the implementation can change again without the guarantees going quiet.
func TestParentAndChildLinksMustAgree(t *testing.T) {
	cases := map[string]func(*Document){
		"parent does not list the child": func(d *Document) {
			root := d.Entities["scene-root"]
			orphan := root.Children[0]
			root.Children = root.Children[1:]
			d.Entities["scene-root"] = root
			_ = orphan
		},
		"child names a different parent": func(d *Document) {
			root := d.Entities["scene-root"]
			child := d.Entities[root.Children[0]]
			child.Parent = "scene-root-imposter"
			d.Entities[child.ID] = child
			imposter := Entity{ID: "scene-root-imposter", Name: "Imposter", Transform: IdentityTransform(), Visible: true}
			d.Entities[imposter.ID] = imposter
			d.RootIDs = append(d.RootIDs, imposter.ID)
		},
		"two parents list the same child": func(d *Document) {
			root := d.Entities["scene-root"]
			shared := root.Children[0]
			second := Entity{ID: "second-parent", Name: "Second", Transform: IdentityTransform(), Visible: true, Children: []ID{shared}}
			d.Entities[second.ID] = second
			d.RootIDs = append(d.RootIDs, second.ID)
		},
		"child listed twice by one parent": func(d *Document) {
			root := d.Entities["scene-root"]
			root.Children = append(root.Children, root.Children[0])
			d.Entities["scene-root"] = root
		},
		"parent does not exist": func(d *Document) {
			root := d.Entities["scene-root"]
			child := d.Entities[root.Children[0]]
			child.Parent = "no-such-entity"
			d.Entities[child.ID] = child
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			document := SampleDocument()
			if err := document.Validate(); err != nil {
				t.Fatalf("sample document is invalid before mutation: %v", err)
			}
			mutate(&document)
			if err := document.Validate(); err == nil {
				t.Fatal("validation accepted an inconsistent parent/child link")
			}
		})
	}
}

// Validation runs on the result of every transaction, so its cost is paid on
// every edit. Scanning a parent's child list per entity made it quadratic: a
// flat scene puts every entity under one parent, and 8,000 entities cost
// 36.7 ms and grew as the square. This holds the shape, not a wall-clock
// number, so it stays meaningful on any machine.
func TestValidateScalesLinearlyWithEntityCount(t *testing.T) {
	if testing.Short() {
		t.Skip("validation scaling skipped in short mode")
	}
	measure := func(entities int) time.Duration {
		document := SampleDocument()
		root := document.Entities["scene-root"]
		for i := 0; i < entities; i++ {
			id := ID(fmt.Sprintf("scale-%05d", i))
			entity := meshEntity(id, "Scale", Vec3{}, Geometry{Kind: "box", Width: 1, Height: 1, Depth: 1}, "board-material", true)
			entity.Parent = root.ID
			root.Children = append(root.Children, id)
			document.Entities[id] = entity
		}
		document.Entities[root.ID] = root
		best := time.Duration(0)
		for i := 0; i < 5; i++ {
			start := time.Now()
			if err := document.Validate(); err != nil {
				t.Fatal(err)
			}
			if elapsed := time.Since(start); best == 0 || elapsed < best {
				best = elapsed
			}
		}
		return best
	}

	small := measure(2000)
	large := measure(8000)
	// Four times the entities. Linear predicts about 4x, quadratic about 16x.
	// Ten is clear of measurement noise in both directions.
	ratio := float64(large) / float64(small)
	t.Logf("Validate: 2000 entities %v, 8000 entities %v (x%.1f for 4x entities)", small, large, ratio)
	if ratio > 10 {
		t.Fatalf("Validate grew %.1fx for 4x the entities; linear is about 4x and quadratic about 16x", ratio)
	}
}

// Each modifier was bounded on its own, but the stack was not, and array and
// mirror multiply. Nothing evaluates at commit time, so an unbounded stack
// commits cheaply, journals, and then kills every later compile — including
// the compile on the page-load path, which makes the project stop opening.
func TestModifierStackIsBoundedBeforeItIsStored(t *testing.T) {
	build := func(modifiers []Modifier) Document {
		document := SampleDocument()
		root := document.Entities["scene-root"]
		quad := Entity{ID: "stacked", Name: "Stacked", Parent: root.ID, Transform: IdentityTransform(), Visible: true, Mesh: &MeshComponent{
			Material: "board-material",
			Geometry: Geometry{Kind: "indexed-mesh", Vertices: []Vertex{
				{ID: "a", Position: Vec3{}}, {ID: "b", Position: Vec3{X: 1}}, {ID: "c", Position: Vec3{X: 1, Z: 1}}, {ID: "d", Position: Vec3{Z: 1}},
			}, Faces: []Face{{ID: "top", Vertices: []ID{"a", "b", "c", "d"}}}},
			Modifiers: modifiers,
		}}
		root.Children = append(root.Children, quad.ID)
		document.Entities[root.ID] = root
		document.Entities[quad.ID] = quad
		return document
	}

	// One array of 1000 is inside the budget and must still be accepted.
	single := build([]Modifier{{ID: "one", Kind: "array", Count: 1000, Offset: Vec3{X: 2}}})
	if err := single.Validate(); err != nil {
		t.Fatalf("a single in-budget array modifier was rejected: %v", err)
	}

	// Three stacked arrays of 1000 turn four vertices into four billion.
	explosive := build([]Modifier{
		{ID: "one", Kind: "array", Count: 1000, Offset: Vec3{X: 2}},
		{ID: "two", Kind: "array", Count: 1000, Offset: Vec3{Y: 2}},
		{ID: "three", Kind: "array", Count: 1000, Offset: Vec3{Z: 2}},
	})
	if err := explosive.Validate(); err == nil {
		t.Fatal("validation accepted a modifier stack evaluating to four billion elements")
	}

	// The stack length itself is capped, independent of what each entry costs.
	long := make([]Modifier, 0, modifierStackLimit+1)
	for i := 0; i <= modifierStackLimit; i++ {
		long = append(long, Modifier{ID: ID(fmt.Sprintf("m%d", i)), Kind: "mirror", Axis: "x"})
	}
	if err := build(long).Validate(); err == nil {
		t.Fatalf("validation accepted %d modifiers, limit is %d", len(long), modifierStackLimit)
	}
}

// NURBS segment counts were checked only for lower bounds, so a billion
// segments validated and then asked compileNURBSCurve for 24 GB. Degree was
// capped only against the control-point count, while nurbsBasis recurses two
// children per level: degree 22 already costs 31 ms for a single sample.
func TestNURBSTessellationAndDegreeAreBounded(t *testing.T) {
	build := func(mutate func(*CurveGeometry)) Document {
		document := SampleDocument()
		root := document.Entities["scene-root"]
		points := make([]CurveControlPoint, 0, 4)
		for i := 0; i < 4; i++ {
			points = append(points, CurveControlPoint{ID: ID(fmt.Sprintf("p%d", i)), Position: Vec3{X: float64(i)}, Weight: 1})
		}
		curve := &CurveGeometry{Degree: 3, ControlPoints: points, Knots: []float64{0, 0, 0, 0, 1, 1, 1, 1}, Segments: 32, Radius: 0.1, RadialSegments: 8}
		mutate(curve)
		entity := Entity{ID: "curve", Name: "Curve", Parent: root.ID, Transform: IdentityTransform(), Visible: true, Mesh: &MeshComponent{
			Material: "board-material",
			Geometry: Geometry{Kind: "nurbs-curve", Curve: curve},
		}}
		root.Children = append(root.Children, entity.ID)
		document.Entities[root.ID] = root
		document.Entities[entity.ID] = entity
		return document
	}

	if err := build(func(*CurveGeometry) {}).Validate(); err != nil {
		t.Fatalf("an ordinary curve was rejected: %v", err)
	}
	for name, mutate := range map[string]func(*CurveGeometry){
		"one billion path segments": func(c *CurveGeometry) { c.Segments = 1_000_000_000 },
		"one million radial segments": func(c *CurveGeometry) {
			c.RadialSegments = 1_000_000
		},
		"tessellation product too large": func(c *CurveGeometry) {
			c.Segments = nurbsMaxSegments
			c.RadialSegments = nurbsMaxRadialSegments
		},
		"non-finite radius": func(c *CurveGeometry) { c.Radius = math.Inf(1) },
	} {
		t.Run(name, func(t *testing.T) {
			if err := build(mutate).Validate(); err == nil {
				t.Fatal("validation accepted an unbounded curve")
			}
		})
	}

	// Degree beyond the cap must be refused even when the control-point count
	// would allow it.
	deep := SampleDocument()
	root := deep.Entities["scene-root"]
	count := nurbsMaxDegree + 8
	points := make([]CurveControlPoint, 0, count)
	knots := make([]float64, 0, count+nurbsMaxDegree+2)
	for i := 0; i < count; i++ {
		points = append(points, CurveControlPoint{ID: ID(fmt.Sprintf("d%d", i)), Position: Vec3{X: float64(i)}, Weight: 1})
	}
	degree := nurbsMaxDegree + 1
	for i := 0; i < count+degree+1; i++ {
		knots = append(knots, float64(i))
	}
	entity := Entity{ID: "deep-curve", Name: "Deep", Parent: root.ID, Transform: IdentityTransform(), Visible: true, Mesh: &MeshComponent{
		Material: "board-material",
		Geometry: Geometry{Kind: "nurbs-curve", Curve: &CurveGeometry{Degree: degree, ControlPoints: points, Knots: knots, Segments: 32, Radius: 0.1, RadialSegments: 8}},
	}}
	root.Children = append(root.Children, entity.ID)
	deep.Entities[root.ID] = root
	deep.Entities[entity.ID] = entity
	if err := deep.Validate(); err == nil {
		t.Fatalf("validation accepted degree %d; basis evaluation is exponential in degree", degree)
	}
}
