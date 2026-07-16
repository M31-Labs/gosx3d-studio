package studio

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"

	"m31labs.dev/gosx/scene"
)

// IncrementalStats is browser-free evidence that a compilation reused or
// rebuilt canonical SceneDoc subtrees. A reused parent includes its descendants.
type IncrementalStats struct {
	ReusedSubtrees     int `json:"reusedSubtrees"`
	RecompiledEntities int `json:"recompiledEntities"`
}

type cachedEntity struct {
	fingerprint string
	node        scene.Node
}

// IncrementalCompiler caches immutable typed Scene3D nodes by a deterministic
// fingerprint of their complete authored subtree and current selection state.
type IncrementalCompiler struct {
	mu    sync.Mutex
	cache map[ID]cachedEntity
}

func NewIncrementalCompiler() *IncrementalCompiler {
	return &IncrementalCompiler{cache: make(map[ID]cachedEntity)}
}

func (compiler *IncrementalCompiler) Compile(document Document, selected ID) (scene.Props, IncrementalStats, error) {
	compiler.mu.Lock()
	defer compiler.mu.Unlock()
	if err := document.Validate(); err != nil {
		return scene.Props{}, IncrementalStats{}, err
	}

	fingerprints := make(map[ID]string, len(document.Entities))
	var fingerprint func(ID) (string, error)
	fingerprint = func(id ID) (string, error) {
		if value, ok := fingerprints[id]; ok {
			return value, nil
		}
		entity := document.Entities[id]
		children := make([]string, 0, len(entity.Children))
		for _, childID := range entity.Children {
			value, err := fingerprint(childID)
			if err != nil {
				return "", err
			}
			children = append(children, value)
		}
		var material *Material
		var prefab *PrefabDefinition
		var asset *AssetRecord
		var armature *Armature
		if entity.Mesh != nil {
			value := document.Materials[entity.Mesh.Material]
			material = &value
		}
		if entity.Prefab != nil {
			value := document.Prefabs[entity.Prefab.Prefab]
			prefab = &value
		}
		if entity.Model != nil {
			value := document.Assets[entity.Model.Asset]
			asset = &value
		}
		if entity.Skin != nil {
			value := document.Armatures[entity.Skin.Armature]
			armature = &value
		}
		payload, err := json.Marshal(struct {
			Entity   Entity
			Material *Material
			Prefab   *PrefabDefinition
			Asset    *AssetRecord
			Armature *Armature
			Selected bool
			Children []string
		}{entity, material, prefab, asset, armature, id == selected, children})
		if err != nil {
			return "", fmt.Errorf("fingerprint entity %q: %w", id, err)
		}
		sum := sha256.Sum256(payload)
		value := hex.EncodeToString(sum[:])
		fingerprints[id] = value
		return value, nil
	}

	next := make(map[ID]cachedEntity, len(document.Entities))
	stats := IncrementalStats{}
	var resolve func(ID) (scene.Node, error)
	resolve = func(id ID) (scene.Node, error) {
		value, err := fingerprint(id)
		if err != nil {
			return nil, err
		}
		if cached, ok := compiler.cache[id]; ok && cached.fingerprint == value {
			next[id] = cached
			stats.ReusedSubtrees++
			return cached.node, nil
		}
		node, err := compileEntity(document, id, selected, resolve)
		if err != nil {
			return nil, err
		}
		next[id] = cachedEntity{fingerprint: value, node: node}
		stats.RecompiledEntities++
		return node, nil
	}

	nodes := make([]scene.Node, 0, len(document.RootIDs))
	for _, id := range document.RootIDs {
		node, err := resolve(id)
		if err != nil {
			return scene.Props{}, stats, err
		}
		nodes = append(nodes, node)
	}
	compiler.cache = next
	return appendSelectionGizmo(document, selected, compileProps(document, nodes)), stats, nil
}
