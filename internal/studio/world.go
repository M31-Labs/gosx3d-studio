package studio

import (
	"fmt"
	"strings"
)

// WorldContract is the authored handoff between Studio and a runtime world.
// It keeps gameplay semantics in the canonical SceneDoc instead of burying
// them in untyped metadata or recreating them in every application.
type WorldContract struct {
	WaterZones map[ID]WaterZone   `json:"waterZones,omitempty"`
	Markers    map[ID]WorldMarker `json:"markers,omitempty"`
}

// WaterZone names a bounded body of water and its gameplay interaction
// properties. The runtime profile binds this authored volume to its renderer.
type WaterZone struct {
	ID             ID      `json:"id"`
	Name           string  `json:"name"`
	Center         Vec3    `json:"center"`
	Size           Vec3    `json:"size"`
	SurfaceY       float64 `json:"surfaceY"`
	Current        Vec3    `json:"current,omitempty"`
	BuoyancyScale  float64 `json:"buoyancyScale,omitempty"`
	LinearDrag     float64 `json:"linearDrag,omitempty"`
	RuntimeProfile string  `json:"runtimeProfile"`
}

// WorldMarker is a stable gameplay anchor. Entity optionally names the scene
// entity it follows; its transform remains useful for standalone anchors.
type WorldMarker struct {
	ID       ID         `json:"id"`
	Name     string     `json:"name"`
	Kind     string     `json:"kind"`
	Entity   ID         `json:"entity,omitempty"`
	Position Vec3       `json:"position"`
	Rotation Quaternion `json:"rotation,omitempty"`
}

var worldMarkerKinds = map[string]bool{
	"player-spawn":     true,
	"camera-start":     true,
	"checkpoint":       true,
	"interactable":     true,
	"cinematic-target": true,
}

func validateWorldContract(world *WorldContract, entities map[ID]Entity) error {
	if world == nil {
		return nil
	}
	if len(world.WaterZones) == 0 && len(world.Markers) == 0 {
		return fmt.Errorf("world contract must contain at least one water zone or marker")
	}
	for id, zone := range world.WaterZones {
		if zone.ID != id || strings.TrimSpace(string(id)) == "" || strings.TrimSpace(zone.Name) == "" {
			return fmt.Errorf("water zone key %q must match a named id", id)
		}
		if !finiteVec(zone.Center) || !finiteVec(zone.Size) || !finite(zone.SurfaceY) || !finiteVec(zone.Current) || !finite(zone.BuoyancyScale) || !finite(zone.LinearDrag) {
			return fmt.Errorf("water zone %q contains non-finite values", id)
		}
		if zone.Size.X <= 0 || zone.Size.Y <= 0 || zone.Size.Z <= 0 {
			return fmt.Errorf("water zone %q size must be positive on every axis", id)
		}
		minY, maxY := zone.Center.Y-zone.Size.Y/2, zone.Center.Y+zone.Size.Y/2
		if zone.SurfaceY < minY || zone.SurfaceY > maxY {
			return fmt.Errorf("water zone %q surfaceY must lie inside its volume", id)
		}
		if zone.BuoyancyScale < 0 || zone.LinearDrag < 0 || strings.TrimSpace(zone.RuntimeProfile) == "" {
			return fmt.Errorf("water zone %q requires non-negative dynamics and a runtime profile", id)
		}
	}
	for id, marker := range world.Markers {
		if marker.ID != id || strings.TrimSpace(string(id)) == "" || strings.TrimSpace(marker.Name) == "" {
			return fmt.Errorf("world marker key %q must match a named id", id)
		}
		if !worldMarkerKinds[marker.Kind] || !finiteVec(marker.Position) || !finiteQuaternion(marker.Rotation) {
			return fmt.Errorf("world marker %q is invalid", id)
		}
		if marker.Entity != "" {
			if _, ok := entities[marker.Entity]; !ok {
				return fmt.Errorf("world marker %q references missing entity %q", id, marker.Entity)
			}
		}
	}
	return nil
}

// WaterZoneDepth returns the signed distance below a zone's calm surface.
// Outside the horizontal bounds it returns zero.
func (zone WaterZone) WaterZoneDepth(position Vec3) float64 {
	halfX, halfZ := zone.Size.X/2, zone.Size.Z/2
	if position.X < zone.Center.X-halfX || position.X > zone.Center.X+halfX || position.Z < zone.Center.Z-halfZ || position.Z > zone.Center.Z+halfZ {
		return 0
	}
	return zone.SurfaceY - position.Y
}
