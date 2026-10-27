package data

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"slices"

	"dnd-rts/internal/fixed"
)

// Damage types (docs/FACTIONS.md §1.1).
const (
	DmgPierce    = "pierce"    // physical: chance to ignore armor
	DmgSlash     = "slash"     // physical: plain
	DmgCrush     = "crush"     // physical: bonus vs buildings
	DmgFire      = "fire"      // magic: burning
	DmgLightning = "lightning" // magic: stun chance; no damage to wood / stone buildings
	DmgIce       = "ice"       // magic: slows movement and attacks
	DmgAcid      = "acid"      // magic: lowers armor
	DmgPoison    = "poison"    // magic: damage over time, spreads
	DmgNecrotic  = "necrotic"  // magic: life steal
	DmgRadiant   = "radiant"   // true damage: nothing reduces it; no damage to buildings
)

// DamageTypes lists every damage type (stable order).
var DamageTypes = []string{DmgPierce, DmgSlash, DmgCrush, DmgFire, DmgLightning, DmgIce, DmgAcid, DmgPoison, DmgNecrotic, DmgRadiant}

// IsPhysical reports whether a damage type is physical (armor, dodge, heavy resist apply).
func IsPhysical(t string) bool { return t == DmgPierce || t == DmgSlash || t == DmgCrush }

// Armor classes.
const (
	ArmorLight  = "light"  // dodge chance vs physical
	ArmorMedium = "medium" // plain
	ArmorHeavy  = "heavy"  // resists physical damage
)

// Building materials.
const (
	MaterialWood  = "wood"
	MaterialStone = "stone"
	MaterialMagic = "magic"
)

// Status effects (for immunities).
const (
	EffectBurn   = "burn"
	EffectSlow   = "slow"
	EffectStun   = "stun"
	EffectAcid   = "acid"
	EffectPoison = "poison"
)

// StatusEffects lists every status effect.
var StatusEffects = []string{EffectBurn, EffectSlow, EffectStun, EffectAcid, EffectPoison}

// CombatRules are the global numbers of the damage system (content/combat.json).
type CombatRules struct {
	// MinDamage: armor / magic defense never reduce a hit below this fraction.
	MinDamage fixed.Fixed `json:"minDamage"`
	// EffectInterval: damage-over-time effects tick this often (seconds).
	EffectInterval fixed.Fixed `json:"effectInterval"`

	Armor struct {
		LightDodge  fixed.Fixed `json:"lightDodge"`  // chance to dodge a physical hit
		HeavyResist fixed.Fixed `json:"heavyResist"` // fraction of physical damage removed
	} `json:"armor"`

	Pierce struct {
		IgnoreArmorChance fixed.Fixed `json:"ignoreArmorChance"`
	} `json:"pierce"`
	Crush struct {
		BuildingMultiplier fixed.Fixed `json:"buildingMultiplier"`
	} `json:"crush"`
	Fire struct {
		DotPerSecond fixed.Fixed `json:"dotPerSecond"` // fraction of the hit per second
		Duration     fixed.Fixed `json:"duration"`
		Wood         fixed.Fixed `json:"wood"`  // multiplier vs wooden buildings
		Stone        fixed.Fixed `json:"stone"` // multiplier vs stone buildings
	} `json:"fire"`
	Lightning struct {
		StunChance   fixed.Fixed `json:"stunChance"`
		StunDuration fixed.Fixed `json:"stunDuration"`
	} `json:"lightning"`
	Ice struct {
		Slow     fixed.Fixed `json:"slow"` // fraction of movement and attack speed removed
		Duration fixed.Fixed `json:"duration"`
	} `json:"ice"`
	Acid struct {
		ArmorPerStack fixed.Fixed `json:"armorPerStack"`
		MaxStacks     int         `json:"maxStacks"`
		Duration      fixed.Fixed `json:"duration"`
	} `json:"acid"`
	Poison struct {
		DotPerSecond fixed.Fixed `json:"dotPerSecond"`
		Duration     fixed.Fixed `json:"duration"`
		SpreadChance fixed.Fixed `json:"spreadChance"` // per tick of the effect
		SpreadRadius fixed.Fixed `json:"spreadRadius"`
		MaxSpread    int         `json:"maxSpread"` // extra units one hit can infect
	} `json:"poison"`
	Necrotic struct {
		LifeSteal fixed.Fixed `json:"lifeSteal"` // fraction of damage dealt healed to the attacker
	} `json:"necrotic"`

	// Experience for heroes (docs/FACTIONS.md: kills and captures).
	XP struct {
		Radius  fixed.Fixed `json:"radius"`  // heroes this close to a dying enemy get its XP
		Capture int         `json:"capture"` // XP for each hero near an outpost when it is captured
	} `json:"xp"`
}

func loadCombat(fsys fs.FS, into *CombatRules) []error {
	raw, err := fs.ReadFile(fsys, "combat.json")
	if err != nil {
		return []error{fmt.Errorf("combat.json: %w", err)}
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(into); err != nil {
		return []error{fmt.Errorf("combat.json: %w", err)}
	}
	return nil
}

func (c *Catalog) validateCombat(fail func(kind, id, format string, args ...any)) {
	r := &c.Combat
	frac := func(name string, v fixed.Fixed) {
		if v < 0 || v > fixed.One {
			fail("combat", name, "must be between 0 and 1")
		}
	}
	pos := func(name string, v fixed.Fixed) {
		if v <= 0 {
			fail("combat", name, "must be > 0")
		}
	}
	frac("minDamage", r.MinDamage)
	pos("effectInterval", r.EffectInterval)
	frac("armor.lightDodge", r.Armor.LightDodge)
	frac("armor.heavyResist", r.Armor.HeavyResist)
	frac("pierce.ignoreArmorChance", r.Pierce.IgnoreArmorChance)
	pos("crush.buildingMultiplier", r.Crush.BuildingMultiplier)
	pos("fire.duration", r.Fire.Duration)
	frac("lightning.stunChance", r.Lightning.StunChance)
	frac("ice.slow", r.Ice.Slow)
	pos("ice.duration", r.Ice.Duration)
	pos("acid.duration", r.Acid.Duration)
	pos("poison.duration", r.Poison.Duration)
	frac("poison.spreadChance", r.Poison.SpreadChance)
	frac("necrotic.lifeSteal", r.Necrotic.LifeSteal)
	if r.Acid.MaxStacks < 1 || r.Poison.MaxSpread < 0 {
		fail("combat", "acid / poison", "acid.maxStacks >= 1, poison.maxSpread >= 0")
	}
}

// validDamageType reports whether t is "" (default) or a known damage type.
func validDamageType(t string) bool { return t == "" || slices.Contains(DamageTypes, t) }

// Attack target kinds.
const (
	TargetGround = "ground"
	TargetAir    = "air"
)

// CanHit reports whether the attack can hit a flying (or a ground) target.
func (a *Attack) CanHit(flying bool) bool {
	if a == nil {
		return false
	}
	if len(a.Targets) == 0 {
		return !flying
	}
	want := TargetGround
	if flying {
		want = TargetAir
	}
	return slices.Contains(a.Targets, want)
}

// Primary is the attack used for timing defaults: melee, else ranged (nil if none).
func (u *Unit) Primary() *Attack {
	if u.Melee != nil {
		return u.Melee
	}
	return u.Ranged
}

// Type returns the attack's damage type with defaults: pierce for projectiles, slash otherwise.
func (a *Attack) Type() string {
	if a.DamageType != "" {
		return a.DamageType
	}
	if a.Projectile != nil {
		return DmgPierce
	}
	return DmgSlash
}

// Class returns the unit's armor class (default medium).
func (u *Unit) Class() string {
	if u.ArmorClass == "" {
		return ArmorMedium
	}
	return u.ArmorClass
}

// IsImmune reports whether the unit ignores a status effect.
func (u *Unit) IsImmune(effect string) bool { return slices.Contains(u.Immune, effect) }

// MaterialOf returns the building's material (default stone).
func (b *Building) MaterialOf() string {
	if b.Material == "" {
		return MaterialStone
	}
	return b.Material
}
