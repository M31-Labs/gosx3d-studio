package studio

import (
	"fmt"
	"sort"
)

// RenderGraphIR is the backend-neutral retained render/resource schedule that
// CompileRenderGraph lowers authored records into. Studio owns this type.
// GoSX carried an identical contract until it removed the types as dead: no
// GoSX renderer, Go or client-side, ever read them. Studio was the only
// writer. Owning the plan here keeps the same portable JSON and stops a
// Studio contract from depending on a framework type nothing else consumes.
// SceneDoc still does not own an editor renderer graph; it owns authored
// resource and pass records, and this is their deterministic lowering.
type RenderGraphIR struct {
	Resources   []RenderResourceIR   `json:"resources"`
	Passes      []RenderGraphPassIR  `json:"passes"`
	Allocations []RenderAllocationIR `json:"allocations,omitempty"`
}

type RenderResourceIR struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Ownership string `json:"ownership"`
	Format    string `json:"format,omitempty"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
	Bytes     int64  `json:"bytes,omitempty"`
}

type RenderGraphPassIR struct {
	ID      string   `json:"id"`
	Kind    string   `json:"kind"`
	Reads   []string `json:"reads,omitempty"`
	Writes  []string `json:"writes,omitempty"`
	Depends []string `json:"depends,omitempty"`
}

type RenderAllocationIR struct {
	Resource string `json:"resource"`
	Slot     int    `json:"slot"`
	FirstUse int    `json:"firstUse"`
	LastUse  int    `json:"lastUse"`
}

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
	// Copy rather than store the caller's pointer. The same Operation stays
	// reachable from the undo stack's retained Transaction, so keeping the
	// pointer would leave the canonical document sharing mutable state with
	// history. Every other apply function copies for this reason; this one
	// did not, and Clone's guarantee that a document owns everything
	// reachable from it depends on all of them doing so.
	graph := RenderGraph{
		Resources: make(map[ID]RenderResource, len(operation.RenderGraph.Resources)),
		Passes:    make(map[ID]RenderPass, len(operation.RenderGraph.Passes)),
	}
	for id, resource := range operation.RenderGraph.Resources {
		graph.Resources[id] = resource
	}
	for id, pass := range operation.RenderGraph.Passes {
		copied := pass
		copied.Reads = append([]ID(nil), pass.Reads...)
		copied.Writes = append([]ID(nil), pass.Writes...)
		copied.Depends = append([]ID(nil), pass.Depends...)
		graph.Passes[id] = copied
	}
	document.RenderGraph = &graph
	return nil, nil
}

// CompileRenderGraph validates, schedules, and aliases a retained graph into
// the portable Studio plan. Ordering is stable across map iteration.
func CompileRenderGraph(graph RenderGraph) (RenderGraphIR, error) {
	resourceIDs := sortedIDs(graph.Resources)
	passIDs := sortedIDs(graph.Passes)
	for _, id := range resourceIDs {
		r := graph.Resources[id]
		if r.ID != id || (r.Ownership != "imported" && r.Ownership != "persistent" && r.Ownership != "transient") {
			return RenderGraphIR{}, fmt.Errorf("render resource %q has invalid id or ownership", id)
		}
	}
	indegree := make(map[ID]int, len(passIDs))
	edges := make(map[ID][]ID, len(passIDs))
	for _, id := range passIDs {
		p := graph.Passes[id]
		if p.ID != id {
			return RenderGraphIR{}, fmt.Errorf("render pass %q has mismatched id", id)
		}
		for _, resource := range append(append([]ID{}, p.Reads...), p.Writes...) {
			if _, ok := graph.Resources[resource]; !ok {
				return RenderGraphIR{}, fmt.Errorf("render pass %q references missing resource %q", id, resource)
			}
		}
		for _, dependency := range p.Depends {
			if _, ok := graph.Passes[dependency]; !ok {
				return RenderGraphIR{}, fmt.Errorf("render pass %q depends on missing pass %q", id, dependency)
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
		return RenderGraphIR{}, fmt.Errorf("render graph contains a dependency cycle")
	}
	written := map[ID]bool{}
	first, last := map[ID]int{}, map[ID]int{}
	for index, id := range order {
		p := graph.Passes[id]
		for _, r := range p.Reads {
			if graph.Resources[r].Ownership == "transient" && !written[r] {
				return RenderGraphIR{}, fmt.Errorf("render pass %q reads transient resource %q before write", id, r)
			}
			markUse(first, last, r, index)
		}
		for _, r := range p.Writes {
			written[r] = true
			markUse(first, last, r, index)
		}
	}
	result := RenderGraphIR{}
	for _, id := range resourceIDs {
		r := graph.Resources[id]
		result.Resources = append(result.Resources, RenderResourceIR{ID: string(id), Kind: r.Kind, Ownership: r.Ownership, Format: r.Format, Width: r.Width, Height: r.Height, Bytes: r.Bytes})
	}
	for _, id := range order {
		p := graph.Passes[id]
		result.Passes = append(result.Passes, RenderGraphPassIR{ID: string(id), Kind: p.Kind, Reads: idsToStrings(p.Reads), Writes: idsToStrings(p.Writes), Depends: idsToStrings(p.Depends)})
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
		result.Allocations = append(result.Allocations, RenderAllocationIR{Resource: string(id), Slot: slot, FirstUse: first[id], LastUse: last[id]})
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
