package studio

import (
	"encoding/json"
	"reflect"
	"testing"
)

// Clone is what keeps the canonical document and everything derived from it —
// undo history, previews, the journal — from sharing mutable state. It used to
// round-trip through JSON, which was slow but structurally could not alias.
// A copier can, so these hold the two properties that matters:
//
//  1. Nothing mutable is shared with the original.
//  2. The copy is indistinguishable from the JSON round trip it replaced.

// sharedMutableState walks two values in parallel and reports every place they
// refer to the same slice, map, or pointer. A field the copier forgot shows up
// here as a shared address, whatever its type and however deeply nested.
func sharedMutableState(t *testing.T, a, b reflect.Value, path string, found *[]string) {
	t.Helper()
	if !a.IsValid() || !b.IsValid() || a.Type() != b.Type() {
		return
	}
	switch a.Kind() {
	case reflect.Pointer:
		if a.IsNil() || b.IsNil() {
			return
		}
		if a.Pointer() == b.Pointer() {
			*found = append(*found, path+" (pointer)")
			return
		}
		sharedMutableState(t, a.Elem(), b.Elem(), path, found)
	case reflect.Slice:
		if a.IsNil() || b.IsNil() || a.Len() == 0 {
			return
		}
		if a.Pointer() == b.Pointer() {
			*found = append(*found, path+" (slice backing array)")
			return
		}
		for i := 0; i < a.Len() && i < b.Len(); i++ {
			sharedMutableState(t, a.Index(i), b.Index(i), path+"[]", found)
		}
	case reflect.Map:
		if a.IsNil() || b.IsNil() || a.Len() == 0 {
			return
		}
		if a.Pointer() == b.Pointer() {
			*found = append(*found, path+" (map)")
			return
		}
		iter := a.MapRange()
		for iter.Next() {
			other := b.MapIndex(iter.Key())
			if other.IsValid() {
				sharedMutableState(t, iter.Value(), other, path+"["+iter.Key().String()+"]", found)
			}
		}
	case reflect.Struct:
		for i := 0; i < a.NumField(); i++ {
			if !a.Type().Field(i).IsExported() {
				continue
			}
			sharedMutableState(t, a.Field(i), b.Field(i), path+"."+a.Type().Field(i).Name, found)
		}
	case reflect.Array:
		for i := 0; i < a.Len(); i++ {
			sharedMutableState(t, a.Index(i), b.Index(i), path+"[]", found)
		}
	}
}

func TestCloneSharesNoMutableStateWithItsSource(t *testing.T) {
	for name, seed := range map[string]func() Document{
		"sample":            SampleDocument,
		"articulated proof": ArticulatedProofDocument,
	} {
		t.Run(name, func(t *testing.T) {
			document := seed()
			clone, err := document.Clone()
			if err != nil {
				t.Fatal(err)
			}
			var shared []string
			sharedMutableState(t, reflect.ValueOf(document), reflect.ValueOf(clone), "Document", &shared)
			if len(shared) > 0 {
				t.Errorf("clone shares mutable state with its source at %d place(s):", len(shared))
				for _, where := range shared {
					t.Errorf("  %s", where)
				}
			}
		})
	}
}

// The copy has to be indistinguishable from the JSON round trip it replaced.
// Fingerprints are compared for equality across save, reopen, propose, and
// direct commit, so any difference — a nil slice that became empty, a
// transform that stopped being canonicalized — would surface as two documents
// that are the same but do not compare equal.
func TestCloneMatchesTheJSONRoundTripItReplaced(t *testing.T) {
	for name, seed := range map[string]func() Document{
		"sample":            SampleDocument,
		"articulated proof": ArticulatedProofDocument,
	} {
		t.Run(name, func(t *testing.T) {
			document := seed()

			clone, err := document.Clone()
			if err != nil {
				t.Fatal(err)
			}
			data, err := json.Marshal(document)
			if err != nil {
				t.Fatal(err)
			}
			var roundTrip Document
			if err := json.Unmarshal(data, &roundTrip); err != nil {
				t.Fatal(err)
			}

			cloned, err := clone.Fingerprint()
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := roundTrip.Fingerprint()
			if err != nil {
				t.Fatal(err)
			}
			if cloned != decoded {
				t.Fatalf("clone fingerprint %s does not match the JSON round trip %s", cloned, decoded)
			}
			if !reflect.DeepEqual(clone, roundTrip) {
				t.Fatal("clone differs structurally from the JSON round trip it replaced")
			}
		})
	}
}

// A clone through JSON canonicalized every transform, because Transform's
// MarshalJSON and UnmarshalJSON both do. Code reading a cloned transform has
// always seen a canonical one, so the copier preserves it.
func TestCloneCanonicalizesTransformsLikeTheJSONPathDid(t *testing.T) {
	document := SampleDocument()
	root := document.Entities["scene-root"]
	// A zero quaternion beside a non-zero euler is the shape a legacy document
	// or an in-code literal produces. Canonicalizing derives the quaternion.
	raw := Transform{Position: Vec3{X: 1}, Euler: Vec3{Z: 0.5}, Scale: Vec3{X: 1, Y: 1, Z: 1}}
	root.Transform = raw
	document.Entities["scene-root"] = root

	clone, err := document.Clone()
	if err != nil {
		t.Fatal(err)
	}
	got := clone.Entities["scene-root"].Transform
	if got.Rotation == (Quaternion{}) {
		t.Fatal("clone left a zero quaternion; the JSON path derived it from the euler angles")
	}
	if want := raw.canonical(); got != want {
		t.Fatalf("cloned transform = %+v, want the canonical %+v", got, want)
	}
}

// Mutating a clone must never reach the document it came from. This is the
// property every other guarantee rests on: undo history retains documents, and
// a commit rebinds rather than edits.
func TestMutatingACloneDoesNotReachTheOriginal(t *testing.T) {
	document := ArticulatedProofDocument()
	clone, err := document.Clone()
	if err != nil {
		t.Fatal(err)
	}
	before, err := document.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}

	for id, entity := range clone.Entities {
		entity.Name = "mutated"
		entity.Children = append(entity.Children, "injected")
		if entity.Mesh != nil {
			entity.Mesh.Material = "mutated-material"
			if len(entity.Mesh.Geometry.Vertices) > 0 {
				entity.Mesh.Geometry.Vertices[0].Position.X += 100
			}
		}
		clone.Entities[id] = entity
	}
	for id, material := range clone.Materials {
		material.Color = "#000000"
		clone.Materials[id] = material
	}
	clone.RootIDs = append(clone.RootIDs, "injected-root")
	if clone.Metadata != nil {
		clone.Metadata["injected"] = "yes"
	}

	after, err := document.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatal("mutating a clone changed the document it was cloned from")
	}
}
