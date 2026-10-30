package data

import (
	"fmt"
	"slices"

	"dnd-rts/internal/fixed"
)

// Abilities (docs/ABILITIES.md). Files: content/abilities/<id>.json, units
// list them in "abilities". Everything is data: an ability is a kind (how it
// is used), a target (what it is aimed at) and a list of effects.

// FormationWedge: a triangle, point first (knights' wedge).
const FormationWedge = "wedge"

// Ability kinds.
const (
	AbilityActive  = "active"  // cast on command (or autocast), then cooldown
	AbilityToggle  = "toggle"  // switched on / off; modifiers apply to the own horde while on
	AbilityPassive = "passive" // always on: modifiers on the own horde
	AbilityAura    = "aura"    // always on: modifiers on units around the caster
	AbilityMode    = "mode"    // pick one of Modes (an element): its modifiers apply, other abilities follow it
)

// Ability targets (active abilities).
const (
	TargetSelf  = "self"  // centred on the caster
	TargetPoint = "point" // a ground point within range
	TargetUnit  = "unit"  // a horde within range (enemy or ally, see Affects)
)

// Who effects / auras touch.
const (
	AffectsEnemies = "enemies"
	AffectsAllies  = "allies"
	AffectsAll     = "all"
)

// Effect types.
const (
	EffDamage    = "damage"    // damage of DamageType to everyone affected in Radius (0 = the target only)
	EffHeal      = "heal"      // restore Amount HP (living units only)
	EffStatus    = "status"    // apply Status for Duration
	EffKnockback = "knockback" // push away from the centre with Force
	EffDispel    = "dispel"    // remove negative statuses
	EffBuff      = "buff"      // Modifiers for Duration
	EffZone      = "zone"      // a lingering area: Effects every Interval for Duration
	EffChain     = "chain"     // hits the target, then jumps to the nearest other enemies (Effects on each)
	EffSummon    = "summon"    // Count hordes of type Horde near the centre, for Duration (0 = until killed)
	EffTeleport  = "teleport"  // the caster's horde (and own hordes in Radius around it) jump to the centre
	EffTransform = "transform" // the caster's horde becomes Unit for Duration
)

// Effect area shapes.
const (
	ShapeCircle = "circle" // default: Radius around the centre
	ShapeCone   = "cone"   // from the caster towards the centre: Radius long, Angle degrees wide
)

// Statuses an effect can apply (on top of those from damage types).
const (
	StatusStun   = "stun"
	StatusSlow   = "slow"
	StatusBlind  = "blind" // attacks miss
	StatusBurn   = "burn"
	StatusAcid   = "acid"
	StatusGround = "ground" // flying units are pulled down: ground attacks can hit them
)

// Ability is a skill of a unit (heroes, mages, special units).
type Ability struct {
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	Kind     string          `json:"kind"`             // AbilityActive / Toggle / Passive / Aura
	Target   string          `json:"target,omitempty"` // active: self / point / unit
	Range    fixed.Fixed     `json:"range,omitempty"`  // point / unit: max cast distance
	Cooldown fixed.Fixed     `json:"cooldown,omitempty"`
	Autocast bool            `json:"autocast,omitempty"` // may be set to cast itself when useful
	Requires *Requires       `json:"requires,omitempty"` // e.g. an upgrade that unlocks it
	Affects  string          `json:"affects,omitempty"`  // default: enemies for damage, allies for heal / buff
	Effects  []AbilityEffect `json:"effects,omitempty"`

	// Toggle / passive: modifiers on the own horde. Aura: on units around.
	Modifiers Modifiers   `json:"modifiers,omitempty"`
	Radius    fixed.Fixed `json:"radius,omitempty"` // aura radius
	// ByMode: toggle / passive / aura modifiers per mode of the unit's mode
	// ability (e.g. a ward against the chosen element); Modifiers otherwise.
	ByMode map[string]Modifiers `json:"byMode,omitempty"`
	// Tags: an aura touches only units with one of these tags (empty = all).
	Tags []string `json:"tags,omitempty"`

	// Mode: the choices; switching costs Cooldown (seconds).
	Modes []ModeChoice `json:"modes,omitempty"`

	// Ultimate: the hero's big ability (UI only; rules are in its numbers).
	Ultimate bool `json:"ultimate,omitempty"`
}

// ModeChoice is one choice of a mode ability.
type ModeChoice struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Modifiers Modifiers `json:"modifiers,omitempty"`
}

// ModsFor returns the modifiers for the unit's current mode ("" = none).
func (a *Ability) ModsFor(mode string) *Modifiers {
	if m, ok := a.ByMode[mode]; ok {
		return &m
	}
	return &a.Modifiers
}

// ModeIDs lists the mode ability's choices.
func (a *Ability) ModeIDs() []string {
	out := make([]string, len(a.Modes))
	for i, m := range a.Modes {
		out[i] = m.ID
	}
	return out
}

// AbilityEffect is one thing an ability does.
type AbilityEffect struct {
	Type       string      `json:"type"`
	DamageType string      `json:"damageType,omitempty"`
	Amount     fixed.Fixed `json:"amount,omitempty"`   // damage / heal
	Radius     fixed.Fixed `json:"radius,omitempty"`   // 0 = only the target
	Status     string      `json:"status,omitempty"`   // EffStatus
	Duration   fixed.Fixed `json:"duration,omitempty"` // status / buff / zone, seconds
	Force      fixed.Fixed `json:"force,omitempty"`    // knockback speed at the centre
	Affects    string      `json:"affects,omitempty"`  // overrides the ability's
	Mode       string      `json:"mode,omitempty"`     // runs only while the caster's unit is in this mode

	Modifiers Modifiers       `json:"modifiers,omitempty"` // EffBuff
	Interval  fixed.Fixed     `json:"interval,omitempty"`  // EffZone: seconds between pulses
	Effects   []AbilityEffect `json:"effects,omitempty"`   // EffZone: each pulse; EffChain: each target

	// Area shape (damage / heal / status / knockback / dispel / buff).
	Shape string      `json:"shape,omitempty"` // circle (default) or cone
	Angle fixed.Fixed `json:"angle,omitempty"` // cone width, degrees

	// EffChain.
	Jumps     int         `json:"jumps,omitempty"`     // extra targets after the first
	JumpRange fixed.Fixed `json:"jumpRange,omitempty"` // max distance of one jump
	Falloff   fixed.Fixed `json:"falloff,omitempty"`   // amount lost per jump (0.2 = −20%)

	// EffSummon.
	Horde string `json:"horde,omitempty"` // horde type
	Count int    `json:"count,omitempty"` // hordes (default 1)
	// Limit: one summon of this ability at a time; its cooldown only runs
	// while nothing it summoned is alive (the emperor's guard).
	Limit bool `json:"limit,omitempty"`

	// EffTeleport: only next to an own finished building (the centre must be
	// within 12 m of one); otherwise anywhere within range.
	ToBuilding bool `json:"toBuilding,omitempty"`

	// EffTransform.
	Unit string `json:"unit,omitempty"` // the form (keeps abilities, level and the health fraction)
}

// Modifiers change unit stats while active. Fractions add up: two +0.15
// damage buffs give ×1.3. Negative values weaken.
type Modifiers struct {
	Armor        fixed.Fixed `json:"armor,omitempty"`        // + flat armor
	MagicDefense fixed.Fixed `json:"magicDefense,omitempty"` // + flat magic defense
	Damage       fixed.Fixed `json:"damage,omitempty"`       // + fraction of attack damage
	AttackSpeed  fixed.Fixed `json:"attackSpeed,omitempty"`  // + fraction
	MoveSpeed    fixed.Fixed `json:"moveSpeed,omitempty"`    // + fraction (−0.4 = 40% slower)
	Range        fixed.Fixed `json:"range,omitempty"`        // + fraction of ranged range
	DamageTaken  fixed.Fixed `json:"damageTaken,omitempty"`  // + fraction of all damage taken
	RangedTaken  fixed.Fixed `json:"rangedTaken,omitempty"`  // + fraction of damage taken from volleys
	Invulnerable bool        `json:"invulnerable,omitempty"`
	NoPierce     bool        `json:"noPierce,omitempty"` // pierce attacks lose their armor-ignore chance (fire mode)
	StunImmune   bool        `json:"stunImmune,omitempty"`
	// Extra damage of another type added to every attack hit, as a fraction of it.
	ExtraType   string      `json:"extraType,omitempty"`
	ExtraDamage fixed.Fixed `json:"extraDamage,omitempty"`
	// AttackType replaces the damage type of the unit's attacks (element choice).
	AttackType string `json:"attackType,omitempty"`
	// Resist: fraction of damage of ResistType removed (elemental ward).
	ResistType string      `json:"resistType,omitempty"`
	Resist     fixed.Fixed `json:"resist,omitempty"`
	// AbilityHaste: + fraction of ability cooldown speed (0.33 ≈ −25% cooldown).
	AbilityHaste fixed.Fixed `json:"abilityHaste,omitempty"`
	// Immune: a status effect that doesn't apply (burn, slow, stun, acid, poison).
	Immune string `json:"immune,omitempty"`
	// WorkSpeed: auras only — + fraction of construction and training speed
	// of own buildings in the radius (the emperor's oversight).
	WorkSpeed fixed.Fixed `json:"workSpeed,omitempty"`

	// Rooted: the horde stands its ground (doesn't move or chase) and can't be
	// trampled; a charge into it is broken.
	Rooted bool `json:"rooted,omitempty"`
	// SpeedDamage: pierce damage per m/s of speed to enemies that run into the
	// horde (spear wall) — cavalry at full gallop takes the most.
	SpeedDamage fixed.Fixed `json:"speedDamage,omitempty"`
	// Formation: "wedge" lines the horde up as a triangle, point first.
	Formation string `json:"formation,omitempty"`
	// Charge: + fraction of trample damage and knockback; a wedge also loses
	// half the momentum per trampled foe (it drives deeper).
	Charge fixed.Fixed `json:"charge,omitempty"`
}

// IsZero reports whether the modifiers change nothing.
func (m Modifiers) IsZero() bool { return m == Modifiers{} }

func (a *Ability) affects(e *AbilityEffect) string {
	if e.Affects != "" {
		return e.Affects
	}
	if a.Affects != "" {
		return a.Affects
	}
	switch e.Type {
	case EffHeal, EffBuff, EffDispel:
		return AffectsAllies
	}
	return AffectsEnemies
}

// AffectsOf returns who an effect of this ability touches (with defaults).
func (a *Ability) AffectsOf(e *AbilityEffect) string { return a.affects(e) }

func (c *Catalog) validateAbilities(fail func(kind, id, format string, args ...any)) {
	for _, id := range c.AbilityIDs {
		a := c.Abilities[id]
		if !validID.MatchString(id) {
			fail("ability", id, "id must be lowercase snake_case")
		}
		if a.Name == "" {
			fail("ability", id, "name is required")
		}
		switch a.Kind {
		case AbilityActive:
			if !slices.Contains([]string{TargetSelf, TargetPoint, TargetUnit}, a.Target) {
				fail("ability", id, "active abilities need target self, point or unit")
			}
			if a.Target != TargetSelf && a.Range <= 0 {
				fail("ability", id, "point / unit abilities need range > 0")
			}
			if a.Cooldown <= 0 || len(a.Effects) == 0 {
				fail("ability", id, "active abilities need cooldown > 0 and effects")
			}
		case AbilityToggle, AbilityPassive:
			if a.Modifiers.IsZero() && len(a.ByMode) == 0 {
				fail("ability", id, "%s abilities need modifiers", a.Kind)
			}
		case AbilityAura:
			if (a.Modifiers.IsZero() && len(a.ByMode) == 0) || a.Radius <= 0 {
				fail("ability", id, "auras need modifiers and radius > 0")
			}
		case AbilityMode:
			if len(a.Modes) < 2 {
				fail("ability", id, "mode abilities need at least 2 modes")
			}
			seen := map[string]bool{}
			for _, m := range a.Modes {
				if !validID.MatchString(m.ID) || m.Name == "" || seen[m.ID] {
					fail("ability", id, "mode %q: needs a unique snake_case id and a name", m.ID)
				}
				seen[m.ID] = true
				c.validateMods(id, m.Modifiers, fail)
			}
		default:
			fail("ability", id, "kind must be active, toggle, passive, aura or mode")
		}
		if a.Affects != "" && !slices.Contains([]string{AffectsEnemies, AffectsAllies, AffectsAll}, a.Affects) {
			fail("ability", id, "affects must be enemies, allies or all")
		}
		c.validateEffects(id, a.Effects, fail, 0)
		c.validateMods(id, a.Modifiers, fail)
		for _, m := range a.ByMode {
			c.validateMods(id, m, fail)
		}
	}
	for _, uid := range c.UnitIDs {
		var modes []string
		var needed []string
		for _, ab := range c.Units[uid].Abilities {
			a, ok := c.Abilities[ab]
			if !ok {
				fail("unit", uid, "unknown ability %q", ab)
				continue
			}
			if a.Kind == AbilityMode {
				if modes != nil {
					fail("unit", uid, "only one mode ability per unit")
				}
				modes = a.ModeIDs()
			}
			for m := range a.ByMode {
				needed = append(needed, m)
			}
			needed = append(needed, effectModes(a.Effects)...)
		}
		for _, m := range needed {
			if !slices.Contains(modes, m) {
				fail("unit", uid, "mode %q is used by its abilities but no mode ability offers it", m)
			}
		}
	}
}

func (c *Catalog) validateEffects(id string, effects []AbilityEffect, fail func(kind, id, format string, args ...any), depth int) {
	for i, e := range effects {
		where := fmt.Sprintf("effect %d (%s)", i, e.Type)
		switch e.Type {
		case EffDamage:
			if !slices.Contains(DamageTypes, e.DamageType) || e.Amount <= 0 {
				fail("ability", id, "%s: needs a known damageType and amount > 0", where)
			}
		case EffHeal:
			if e.Amount <= 0 {
				fail("ability", id, "%s: amount must be > 0", where)
			}
		case EffStatus:
			if !slices.Contains([]string{StatusStun, StatusSlow, StatusBlind, StatusBurn, StatusAcid, StatusGround}, e.Status) || e.Duration <= 0 {
				fail("ability", id, "%s: status must be stun, slow, blind, burn or acid, duration > 0", where)
			}
		case EffKnockback:
			if e.Force <= 0 {
				fail("ability", id, "%s: force must be > 0", where)
			}
		case EffDispel:
		case EffBuff:
			if e.Modifiers.IsZero() || e.Duration <= 0 {
				fail("ability", id, "%s: needs modifiers and duration > 0", where)
			}
			c.validateMods(id, e.Modifiers, fail)
		case EffChain:
			if e.Jumps < 1 || e.JumpRange <= 0 || len(e.Effects) == 0 || e.Falloff < 0 || e.Falloff >= fixed.One {
				fail("ability", id, "%s: chains need jumps >= 1, jumpRange > 0, falloff 0..1 and effects", where)
			}
			for _, n := range e.Effects {
				if n.Type != EffDamage && n.Type != EffStatus {
					fail("ability", id, "%s: chain effects can only be damage or status", where)
				}
			}
			c.validateEffects(id, e.Effects, fail, depth+1)
		case EffSummon:
			if _, ok := c.Hordes[e.Horde]; !ok {
				fail("ability", id, "%s: unknown horde %q", where, e.Horde)
			}
			if e.Count < 0 || e.Duration < 0 {
				fail("ability", id, "%s: count and duration must be >= 0", where)
			}
		case EffTeleport:
		case EffTransform:
			if _, ok := c.Units[e.Unit]; !ok || e.Duration <= 0 {
				fail("ability", id, "%s: needs a known unit and duration > 0", where)
			}
		case EffZone:
			if depth > 0 {
				fail("ability", id, "%s: zones can't contain zones", where)
			}
			if e.Radius <= 0 || e.Duration <= 0 || e.Interval <= 0 || len(e.Effects) == 0 {
				fail("ability", id, "%s: zones need radius, duration, interval > 0 and effects", where)
			}
			c.validateEffects(id, e.Effects, fail, depth+1)
		default:
			fail("ability", id, "%s: unknown effect type", where)
		}
		switch e.Shape {
		case "", ShapeCircle:
		case ShapeCone:
			if e.Radius <= 0 || e.Angle <= 0 || e.Angle > fixed.FromInt(360) {
				fail("ability", id, "%s: cones need radius > 0 and angle 0..360", where)
			}
		default:
			fail("ability", id, "%s: shape must be circle or cone", where)
		}
		if e.Radius < 0 || e.Amount < 0 {
			fail("ability", id, "%s: radius and amount must be >= 0", where)
		}
	}
}

// effectModes lists the modes effects (and nested effects) depend on.
func effectModes(effects []AbilityEffect) []string {
	var out []string
	for _, e := range effects {
		if e.Mode != "" {
			out = append(out, e.Mode)
		}
		out = append(out, effectModes(e.Effects)...)
	}
	return out
}

func (c *Catalog) validateMods(id string, m Modifiers, fail func(kind, id, format string, args ...any)) {
	if m.Formation != "" && m.Formation != FormationWedge {
		fail("ability", id, "modifiers.formation must be %q", FormationWedge)
	}
	if m.Immune != "" && !slices.Contains(StatusEffects, m.Immune) {
		fail("ability", id, "modifiers.immune %q must be one of %v", m.Immune, StatusEffects)
	}
	if m.AttackType != "" && !slices.Contains(DamageTypes, m.AttackType) {
		fail("ability", id, "modifiers.attackType %q is unknown", m.AttackType)
	}
	if (m.ResistType == "") != (m.Resist == 0) || (m.ResistType != "" && !slices.Contains(DamageTypes, m.ResistType)) {
		fail("ability", id, "modifiers.resistType (a known damage type) and resist go together")
	}
	if m.ExtraType != "" && !slices.Contains(DamageTypes, m.ExtraType) {
		fail("ability", id, "modifiers.extraType %q is unknown", m.ExtraType)
	}
	if (m.ExtraType == "") != (m.ExtraDamage == 0) {
		fail("ability", id, "modifiers.extraType and extraDamage go together")
	}
}
