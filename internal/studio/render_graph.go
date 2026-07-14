package studio

import (
	"fmt"
	"sort"

	"m31labs.dev/gosx/scene"
)

type RenderGraph struct {
	Resources map[ID]RenderResource `json:"resources"`
	Passes    map[ID]RenderPass     `json:"passes"`
}

type RenderResource struct {
	ID        ID     `json:"id"`
	Kind      string `json:"kind"`
	Ownership string `json:"ownership"`
	Format    string `json:"format,omitempty"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
	Bytes     int64  `json:"bytes,omitempty"`
}

type RenderPass struct {
	ID      ID     `json:"id"`
	Kind    string `json:"kind"`
	Reads   []ID   `json:"reads,omitempty"`
	Writes  []ID   `json:"writes,omitempty"`
	Depends []ID   `json:"depends,omitempty"`
}

func applySetRenderGraph(document *Document, operation Operation) ([]ID, error) {
	if operation.RenderGraph == nil {
		return nil, fmt.Errorf("set-render-graph requires renderGraph")
	}
	if _, err := CompileRenderGraph(*operation.RenderGraph); err != nil {
		return nil, err
	}
	document.RenderGraph = operation.RenderGraph
	return nil, nil
}

// CompileRenderGraph validates, schedules, and aliases a retained graph into
// the shared GoSX SceneIR contract. Ordering is stable across map iteration.
func CompileRenderGraph(graph RenderGraph) (scene.RenderGraphIR, error) {
	resourceIDs := sortedIDs(graph.Resources)
	passIDs := sortedIDs(graph.Passes)
	for _, id := range resourceIDs {
		r := graph.Resources[id]
		if r.ID != id || (r.Ownership != "imported" && r.Ownership != "persistent" && r.Ownership != "transient") {
			return scene.RenderGraphIR{}, fmt.Errorf("render resource %q has invalid id or ownership", id)
		}
	}
	indegree := make(map[ID]int, len(passIDs))
	edges := make(map[ID][]ID, len(passIDs))
	for _, id := range passIDs {
		p := graph.Passes[id]
		if p.ID != id {
			return scene.RenderGraphIR{}, fmt.Errorf("render pass %q has mismatched id", id)
		}
		for _, resource := range append(append([]ID{}, p.Reads...), p.Writes...) {
			if _, ok := graph.Resources[resource]; !ok {
				return scene.RenderGraphIR{}, fmt.Errorf("render pass %q references missing resource %q", id, resource)
			}
		}
		for _, dependency := range p.Depends {
			if _, ok := graph.Passes[dependency]; !ok {
				return scene.RenderGraphIR{}, fmt.Errorf("render pass %q depends on missing pass %q", id, dependency)
			}
			edges[dependency] = append(edges[dependency], id)
			indegree[id]++
		}
	}
	ready := make([]ID, 0)
	for _, id := range passIDs {
		if indegree[id] == 0 {
			ready = append(ready, id)
		}
	}
	order := make([]ID, 0, len(passIDs))
	for len(ready) > 0 {
		id := ready[0]
		ready = ready[1:]
		order = append(order, id)
		sort.Slice(edges[id], func(i, j int) bool { return edges[id][i] < edges[id][j] })
		for _, next := range edges[id] {
			indegree[next]--
			if indegree[next] == 0 {
				ready = append(ready, next)
				sort.Slice(ready, func(i, j int) bool { return ready[i] < ready[j] })
			}
		}
	}
	if len(order) != len(passIDs) {
		return scene.RenderGraphIR{}, fmt.Errorf("render graph contains a dependency cycle")
	}
	written := map[ID]bool{}
	first, last := map[ID]int{}, map[ID]int{}
	for index, id := range order {
		p := graph.Passes[id]
		for _, r := range p.Reads {
			if graph.Resources[r].Ownership == "transient" && !written[r] {
				return scene.RenderGraphIR{}, fmt.Errorf("render pass %q reads transient resource %q before write", id, r)
			}
			markUse(first, last, r, index)
		}
		for _, r := range p.Writes {
			written[r] = true
			markUse(first, last, r, index)
		}
	}
	result := scene.RenderGraphIR{}
	for _, id := range resourceIDs {
		r := graph.Resources[id]
		result.Resources = append(result.Resources, scene.RenderResourceIR{ID: string(id), Kind: r.Kind, Ownership: r.Ownership, Format: r.Format, Width: r.Width, Height: r.Height, Bytes: r.Bytes})
	}
	for _, id := range order {
		p := graph.Passes[id]
		result.Passes = append(result.Passes, scene.RenderGraphPassIR{ID: string(id), Kind: p.Kind, Reads: idsToStrings(p.Reads), Writes: idsToStrings(p.Writes), Depends: idsToStrings(p.Depends)})
	}
	slotsLast := []int{}
	for _, id := range resourceIDs {
		if graph.Resources[id].Ownership != "transient" {
			continue
		}
		slot := -1
		for i, end := range slotsLast {
			if end < first[id] {
				slot = i
				break
			}
		}
		if slot < 0 {
			slot = len(slotsLast)
			slotsLast = append(slotsLast, last[id])
		} else {
			slotsLast[slot] = last[id]
		}
		result.Allocations = append(result.Allocations, scene.RenderAllocationIR{Resource: string(id), Slot: slot, FirstUse: first[id], LastUse: last[id]})
	}
	return result, nil
}

func sortedIDs[T any](values map[ID]T) []ID {
	ids := make([]ID, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}
func idsToStrings(ids []ID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = string(id)
	}
	return out
}
func markUse(first, last map[ID]int, id ID, index int) {
	if _, ok := first[id]; !ok {
		first[id] = index
	}
	last[id] = index
}
