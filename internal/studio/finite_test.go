package studio

import (
	"math"
	"reflect"
	"sort"
	"testing"
)

// A SceneDoc that validates has to be a SceneDoc that persists.
//
// encoding/json refuses NaN and +/-Inf. A non-finite value that survived
// Validate produced a document the workspace could not clone, fingerprint,
// journal, or save: the editor would accept the edit and then fail every
// operation that touches the document. Worse, certification checks compare
// fingerprints, and a swallowed fingerprint error yields "" on both sides,
// so a document too broken to hash read as a match.
//
// Camera, environment, and light records each reached that state — 19 fields
// in all. This test walks every float64 a document can carry and holds the
// implication: if Validate accepts it, Marshal must accept it too.
func TestValidatedDocumentsAlwaysMarshal(t *testing.T) {
	seeds := map[string]func() Document{
		"sample":            SampleDocument,
		"articulated proof": ArticulatedProofDocument,
	}
	for name, seed := range seeds {
		t.Run(name, func(t *testing.T) {
			targets := reachableFloatPaths(seed())
			if len(targets) == 0 {
				t.Fatal("walked no float64 fields; the walker no longer matches the document shape")
			}
			for _, target := range targets {
				document := seed()
				if !setFloatAt(&document, target, math.NaN()) {
					continue
				}
				if document.Validate() != nil {
					continue
				}
				if _, err := document.Fingerprint(); err != nil {
					t.Errorf("%s accepts NaN but then cannot be marshalled: %v", target, err)
				}
			}
			t.Logf("checked %d reachable float64 fields", len(targets))
		})
	}
}

func reachableFloatPaths(document Document) []string {
	var paths []string
	walkFloats(reflect.ValueOf(&document).Elem(), "Document", 0, func(path string, _ reflect.Value) {
		paths = append(paths, path)
	})
	return paths
}

// setFloatAt sets the first float64 reachable at path and reports whether it
// found one.
func setFloatAt(document *Document, target string, value float64) bool {
	done := false
	walkFloats(reflect.ValueOf(document).Elem(), "Document", 0, func(path string, field reflect.Value) {
		if !done && path == target {
			field.SetFloat(value)
			done = true
		}
	})
	return done
}

// walkFloats visits every settable float64 reachable from value. It descends
// one element of each slice and one key of each map: the goal is covering
// every distinct field in the schema, not every value in the document.
func walkFloats(value reflect.Value, path string, depth int, visit func(path string, field reflect.Value)) {
	if depth > 8 || !value.IsValid() {
		return
	}
	switch value.Kind() {
	case reflect.Float64:
		if value.CanSet() {
			visit(path, value)
		}
	case reflect.Pointer, reflect.Interface:
		if !value.IsNil() {
			walkFloats(value.Elem(), path, depth+1, visit)
		}
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			if !value.Type().Field(i).IsExported() {
				continue
			}
			walkFloats(value.Field(i), path+"."+value.Type().Field(i).Name, depth+1, visit)
		}
	case reflect.Slice, reflect.Array:
		if value.Len() > 0 {
			walkFloats(value.Index(0), path+"[0]", depth+1, visit)
		}
	case reflect.Map:
		keys := value.MapKeys()
		if len(keys) == 0 {
			return
		}
		sort.Slice(keys, func(i, j int) bool { return keys[i].String() < keys[j].String() })
		key := keys[0]
		// Map values are not addressable, so poke a copy and write it back.
		element := reflect.New(value.Type().Elem()).Elem()
		element.Set(value.MapIndex(key))
		walkFloats(element, path+"["+key.String()+"]", depth+1, func(p string, f reflect.Value) {
			visit(p, f)
			value.SetMapIndex(key, element)
		})
	}
}

// The concrete records that were unguarded, held directly so a regression
// names the field rather than only failing the schema walk.
func TestCameraEnvironmentAndLightRejectNonFiniteValues(t *testing.T) {
	cases := map[string]func(*Document){
		"camera position":      func(d *Document) { d.Camera.Position.X = math.NaN() },
		"camera rotation":      func(d *Document) { d.Camera.Rotation.Y = math.Inf(-1) },
		"camera fov":           func(d *Document) { d.Camera.FOV = math.NaN() },
		"camera near":          func(d *Document) { d.Camera.Near = math.Inf(1) },
		"camera far":           func(d *Document) { d.Camera.Far = math.NaN() },
		"ambient intensity":    func(d *Document) { d.Environment.AmbientIntensity = math.NaN() },
		"environment exposure": func(d *Document) { d.Environment.Exposure = math.Inf(1) },
		"light intensity":      func(d *Document) { setFirstLight(d, func(l *LightComponent) { l.Intensity = math.NaN() }) },
		"light range":          func(d *Document) { setFirstLight(d, func(l *LightComponent) { l.Range = math.Inf(1) }) },
		"light direction":      func(d *Document) { setFirstLight(d, func(l *LightComponent) { l.Direction.Z = math.NaN() }) },
		"light position":       func(d *Document) { setFirstLight(d, func(l *LightComponent) { l.Position.X = math.NaN() }) },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			document := SampleDocument()
			mutate(&document)
			if err := document.Validate(); err == nil {
				t.Fatal("validation accepted a non-finite value")
			}
		})
	}
}

func setFirstLight(document *Document, mutate func(*LightComponent)) {
	ids := make([]string, 0, len(document.Entities))
	for id := range document.Entities {
		ids = append(ids, string(id))
	}
	sort.Strings(ids)
	for _, id := range ids {
		entity := document.Entities[ID(id)]
		if entity.Light != nil {
			light := *entity.Light
			mutate(&light)
			entity.Light = &light
			document.Entities[ID(id)] = entity
			return
		}
	}
}
