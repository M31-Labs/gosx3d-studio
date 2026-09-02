package studio

import (
	"math"
	"testing"

	"m31labs.dev/gosx/scene"
)

func TestTorusGeometryValidatesAndCompilesToGoSX(t *testing.T) {
	want := scene.TorusGeometry{Radius: 4.06, Tube: 0.115, RadialSegments: 12, TubularSegments: 64}
	geometry := Geometry{
		Kind:            "torus",
		Radius:          want.Radius,
		Tube:            want.Tube,
		RadialSegments:  want.RadialSegments,
		TubularSegments: want.TubularSegments,
	}

	document := SampleDocument()
	root := document.Entities["scene-root"]
	rim := meshEntity("board-rim", "Board rim", Vec3{Y: 0.02}, geometry, "pedestal-material", false)
	rim.Parent = root.ID
	root.Children = append(root.Children, rim.ID)
	document.Entities[root.ID] = root
	document.Entities[rim.ID] = rim

	if err := document.Validate(); err != nil {
		t.Fatalf("torus document must validate: %v", err)
	}
	compiled, err := compileGeometry(geometry)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := compiled.(scene.TorusGeometry); !ok || got != want {
		t.Fatalf("compiled geometry = %#v, want %#v", compiled, want)
	}

	props, err := Compile(document)
	if err != nil {
		t.Fatal(err)
	}
	for _, object := range props.SceneIR().Objects {
		if object.ID != string(rim.ID) {
			continue
		}
		if object.Kind != "torus" || object.Radius != want.Radius || object.Tube != want.Tube || object.RadialSegments != want.RadialSegments || object.TubularSegments != want.TubularSegments {
			t.Fatalf("compiled torus IR = %+v", object)
		}
		return
	}
	t.Fatal("compiled torus is missing from SceneIR")
}

func TestTorusGeometryRejectsNonFiniteDimensions(t *testing.T) {
	for name, value := range map[string]float64{
		"nan":               math.NaN(),
		"positive infinity": math.Inf(1),
		"negative infinity": math.Inf(-1),
	} {
		t.Run(name, func(t *testing.T) {
			geometry := Geometry{Kind: "torus", Radius: 4.06, Tube: value, RadialSegments: 12, TubularSegments: 64}
			if err := validateGeometry(geometry); err == nil {
				t.Fatal("non-finite torus tube must fail validation")
			}
		})
	}
}
