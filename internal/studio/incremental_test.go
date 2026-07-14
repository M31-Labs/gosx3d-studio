package studio

import (
	"reflect"
	"testing"
)

func TestIncrementalCompilerReusesSubtreesAndMatchesCleanCompile(t *testing.T) {
	document := SampleDocument()
	compiler := NewIncrementalCompiler()
	initial, stats, err := compiler.Compile(document, "piece-player-1-01")
	if err != nil {
		t.Fatal(err)
	}
	if stats.RecompiledEntities != len(document.Entities) || stats.ReusedSubtrees != 0 {
		t.Fatalf("initial stats=%+v entities=%d", stats, len(document.Entities))
	}
	clean, err := CompileSelected(document, "piece-player-1-01")
	if err != nil || !reflect.DeepEqual(initial.SceneIR(), clean.SceneIR()) {
		t.Fatalf("initial incremental compilation differs from clean: %v", err)
	}

	changed, err := document.Clone()
	if err != nil {
		t.Fatal(err)
	}
	entity := changed.Entities["piece-player-1-01"]
	entity.Transform.Position.X += 0.25
	changed.Entities[entity.ID] = entity
	changed.Revision++
	incremental, stats, err := compiler.Compile(changed, "piece-player-1-01")
	if err != nil {
		t.Fatal(err)
	}
	if stats.ReusedSubtrees == 0 || stats.RecompiledEntities == 0 || stats.RecompiledEntities >= len(changed.Entities) {
		t.Fatalf("changed stats do not prove incremental work: %+v", stats)
	}
	clean, err = CompileSelected(changed, "piece-player-1-01")
	if err != nil || !reflect.DeepEqual(incremental.SceneIR(), clean.SceneIR()) {
		t.Fatalf("changed incremental compilation differs from clean: %v", err)
	}

	unchanged, stats, err := compiler.Compile(changed, "piece-player-1-01")
	if err != nil {
		t.Fatal(err)
	}
	if stats.ReusedSubtrees != len(changed.RootIDs) || stats.RecompiledEntities != 0 {
		t.Fatalf("unchanged stats=%+v roots=%d", stats, len(changed.RootIDs))
	}
	if !reflect.DeepEqual(unchanged.SceneIR(), clean.SceneIR()) {
		t.Fatal("unchanged incremental compilation differs from clean")
	}
}
