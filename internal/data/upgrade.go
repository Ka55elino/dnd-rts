package data

import (
	"slices"

	"dnd-rts/internal/fixed"
)

// Requires lists what a player must have: finished buildings of these types
// and these upgrades. Used by buildings, hordes, upgrades and abilities.
type Requires struct {
	Buildings []string `json:"buildings,omitempty"`
	Upgrades  []string `json:"upgrades,omitempty"`
}

// IsEmpty reports whether nothing is required.
func (r *Requires) IsEmpty() bool { return r == nil || (len(r.Buildings) == 0 && len(r.Upgrades) == 0) }

// Upgrade is researched once in a building (content/upgrades/<id>.json) and
// then applies to the player's units permanently, including those already
// on the field.
type Upgrade struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	Cost         Amounts     `json:"cost,omitempty"`
	ResearchTime fixed.Fixed `json:"researchTime"` // seconds
	Requires     *Requires   `json:"requires,omitempty"`

	// Group: upgrades of one group replace each other (choose one element):
	// researching one removes the others of the group.
	Group string `json:"group,omitempty"`

	// Targets: which units get the modifiers (unit ids or tags; empty = all).
	Units []string `json:"units,omitempty"`
	Tags  []string `json:"tags,omitempty"`
	// Buildings: towers (building ids) whose shots get the modifiers too.
	Buildings []string `json:"buildings,omitempty"`

	Modifiers Modifiers `json:"modifiers,omitempty"`
}

// Applies reports whether the upgrade's modifiers apply to unit u.
func (g *Upgrade) Applies(u *Unit) bool {
	if len(g.Units) == 0 && len(g.Tags) == 0 {
		return len(g.Buildings) == 0 // only-towers upgrades don't touch units
	}
	if slices.Contains(g.Units, u.ID) {
		return true
	}
	for _, t := range g.Tags {
		if u.HasTag(t) {
			return true
		}
	}
	return false
}

func (c *Catalog) validateRequires(kind, id string, r *Requires, fail func(kind, id, format string, args ...any)) {
	if r == nil {
		return
	}
	for _, b := range r.Buildings {
		if _, ok := c.Buildings[b]; !ok {
			fail(kind, id, "requires unknown building %q", b)
		}
	}
	for _, u := range r.Upgrades {
		if _, ok := c.Upgrades[u]; !ok {
			fail(kind, id, "requires unknown upgrade %q", u)
		}
	}
}

func (c *Catalog) validateUpgrades(fail func(kind, id, format string, args ...any)) {
	for _, id := range c.UpgradeIDs {
		g := c.Upgrades[id]
		if !validID.MatchString(id) {
			fail("upgrade", id, "id must be lowercase snake_case")
		}
		if g.Name == "" || g.ResearchTime <= 0 {
			fail("upgrade", id, "needs a name and researchTime > 0")
		}
		c.checkAmounts(g.Cost, ResourceStock, func(m string) { fail("upgrade", id, "cost: %s", m) })
		c.validateRequires("upgrade", id, g.Requires, fail)
		for _, b := range g.Buildings {
			if bd, ok := c.Buildings[b]; !ok || bd.Attack == nil {
				fail("upgrade", id, "buildings: %q must be a building with an attack", b)
			}
		}
		for _, u := range g.Units {
			if _, ok := c.Units[u]; !ok {
				fail("upgrade", id, "unknown unit %q", u)
			}
		}
		c.validateMods(id, g.Modifiers, fail)
	}
	for _, id := range c.BuildingIDs {
		b := c.Buildings[id]
		c.validateRequires("building", id, b.Requires, fail)
		for _, r := range b.Research {
			if _, ok := c.Upgrades[r]; !ok {
				fail("building", id, "research: unknown upgrade %q", r)
			}
		}
	}
	for _, id := range c.HordeIDs {
		c.validateRequires("horde", id, c.Hordes[id].Requires, fail)
	}
	for _, id := range c.AbilityIDs {
		c.validateRequires("ability", id, c.Abilities[id].Requires, fail)
	}
}

// AppliesTo reports whether the upgrade changes a tower's shots.
func (g *Upgrade) AppliesTo(b *Building) bool { return slices.Contains(g.Buildings, b.ID) }
