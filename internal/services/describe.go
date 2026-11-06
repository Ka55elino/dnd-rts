package services

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"dnd-rts/internal/data"
	"dnd-rts/internal/fixed"
)

// Tooltip text generated from the content numbers, so it never goes stale: what an ability does
// (its effects, modifiers, aura, modes) and what an upgrade gives.

// num formats a number without trailing zeros.
func num(v fixed.Fixed) string {
	return strconv.FormatFloat(math.Round(v.Float()*100)/100, 'f', -1, 64)
}

func pct(v fixed.Fixed) string { return fmt.Sprintf("%+.0f%%", v.Float()*100) }

// modsText lists what modifiers do, e.g. ["+15% damage", "+2 armor", "invisible"].
func modsText(m data.Modifiers) []string {
	var out []string
	add := func(s string) { out = append(out, s) }
	if m.Damage != 0 {
		add(pct(m.Damage) + " damage")
	}
	if m.AttackSpeed != 0 {
		add(pct(m.AttackSpeed) + " attack speed")
	}
	if m.MoveSpeed != 0 {
		add(pct(m.MoveSpeed) + " speed")
	}
	if m.Range != 0 {
		add(pct(m.Range) + " range")
	}
	if m.Armor != 0 {
		add(fmt.Sprintf("%+g armor", m.Armor.Float()))
	}
	if m.MagicDefense != 0 {
		add(fmt.Sprintf("%+g magic defense", m.MagicDefense.Float()))
	}
	if m.DamageTaken != 0 {
		add(pct(m.DamageTaken) + " damage taken")
	}
	if m.RangedTaken != 0 {
		add(pct(m.RangedTaken) + " damage taken from ranged attacks")
	}
	if m.Invulnerable {
		add("invulnerable")
	}
	if m.StunImmune {
		add("can't be stunned")
	}
	if m.Immune != "" {
		add("immune to " + m.Immune)
	}
	if m.ExtraType != "" && m.ExtraDamage > 0 {
		add(fmt.Sprintf("+%.0f%% of every hit as %s damage", m.ExtraDamage.Float()*100, m.ExtraType))
	}
	if m.AttackType != "" {
		add("attacks deal " + m.AttackType + " damage")
	}
	if m.NoPierce {
		add("piercing attacks lose their armor-ignore chance")
	}
	if m.ResistType != "" && m.Resist > 0 {
		add(fmt.Sprintf("%.0f%% %s resistance", m.Resist.Float()*100, m.ResistType))
	}
	if m.AbilityHaste != 0 {
		add(pct(m.AbilityHaste) + " ability cooldown speed")
	}
	if m.WorkSpeed != 0 {
		add(pct(m.WorkSpeed) + " construction and training speed of own buildings")
	}
	if m.Rooted {
		add("holds its ground (can't be trampled, stops charges)")
	}
	if m.SpeedDamage > 0 {
		add(fmt.Sprintf("enemies running into it take %s piercing damage per m/s of speed", num(m.SpeedDamage)))
	}
	if m.Formation == data.FormationWedge {
		add("wedge formation")
	}
	if m.Charge != 0 {
		add(pct(m.Charge) + " charge damage and knockback")
	}
	if m.Reflect > 0 {
		add(fmt.Sprintf("reflects %.0f%% of the damage taken back to the attacker", m.Reflect.Float()*100))
	}
	if m.NoHeal {
		add("can't be healed")
	}
	switch {
	case m.Invisible && m.InTrees:
		add("invisible near trees")
	case m.Invisible:
		add("invisible")
	case m.Camouflage && m.InTrees:
		add("invisible near trees while standing still")
	case m.Camouflage:
		add("invisible while standing still")
	}
	if m.Invisible || m.Camouflage {
		add("attacking or casting reveals it for 3 s")
	}
	if m.Detect > 0 {
		add(fmt.Sprintf("sees invisible units within %s m", num(m.Detect)))
	}
	if m.TreeGrowth > 0 {
		add(fmt.Sprintf("felled trees regrow %s× faster", num(m.TreeGrowth+fixed.One)))
	}
	return out
}

// statusVerb: "Stuns <whom> for 2 s".
var statusVerb = map[string]string{
	data.StatusStun:   "stuns",
	data.StatusSlow:   "slows",
	data.StatusBlind:  "blinds (attacks miss)",
	data.StatusBurn:   "sets on fire",
	data.StatusAcid:   "covers in acid",
	data.StatusGround: "grounds flyers among",
}

func upper(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

var statusText = map[string]string{
	data.StatusStun:   "stuns",
	data.StatusSlow:   "slows",
	data.StatusBlind:  "blinds (attacks miss)",
	data.StatusBurn:   "sets on fire",
	data.StatusAcid:   "covers in acid",
	data.StatusGround: "pulls flyers to the ground",
}

// abilityText describes an ability, one sentence per line.
func (s *ContentService) abilityText(a *data.Ability) []string {
	var out []string
	switch a.Kind {
	case data.AbilityPassive:
		out = append(out, s.modsLine("Always on:", a))
	case data.AbilityToggle:
		out = append(out, s.modsLine("While on:", a))
	case data.AbilityAura:
		who := "allies"
		if a.Affects == data.AffectsEnemies {
			who = "enemies"
		} else if a.Affects == data.AffectsAll {
			who = "everyone"
		}
		if len(a.Tags) > 0 {
			who += " (" + strings.Join(a.Tags, ", ") + ")"
		}
		out = append(out, s.modsLine(fmt.Sprintf("Aura, %s within %s m:", who, num(a.Radius)), a))
	case data.AbilityMode:
		for _, m := range a.Modes {
			line := m.Name
			if t := modsText(m.Modifiers); len(t) > 0 {
				line += ": " + strings.Join(t, ", ")
			}
			out = append(out, line)
		}
	}
	if a.OnTree {
		out = append(out, "Cast on a tree")
	}
	if a.OnBuilding {
		out = append(out, "Cast next to an enemy or neutral building")
	}
	for i := range a.Effects {
		out = append(out, s.effectLines(a, &a.Effects[i], "")...)
	}
	if a.Stealthy {
		out = append(out, "Doesn't reveal an invisible caster")
	}
	return out
}

// modsLine: "<lead> +15% damage, +2 armor" (or per element for ByMode abilities).
func (s *ContentService) modsLine(lead string, a *data.Ability) string {
	if len(a.ByMode) > 0 {
		var parts []string
		for _, mode := range sortedKeys(a.ByMode) {
			parts = append(parts, mode+": "+strings.Join(modsText(a.ByMode[mode]), ", "))
		}
		return lead + " by the chosen element — " + strings.Join(parts, "; ")
	}
	t := modsText(a.Modifiers)
	if len(t) == 0 {
		return lead
	}
	return lead + " " + strings.Join(t, ", ")
}

func sortedKeys(m map[string]data.Modifiers) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}

// whom: "enemies within 5 m", "the target", "allies in a 60° cone, 12 m", "its own horde".
func whom(a *data.Ability, e *data.AbilityEffect) string {
	who := a.AffectsOf(e)
	if who == data.AffectsAll {
		who = "everyone"
	}
	switch {
	case e.Shape == data.ShapeCone:
		return fmt.Sprintf("%s in a %s° cone, %s m", who, num(e.Angle), num(e.Radius))
	case e.Radius > 0:
		return fmt.Sprintf("%s within %s m", who, num(e.Radius))
	case a.Target == data.TargetSelf:
		return "its own horde"
	default:
		return "the target"
	}
}

// effectLines describes one effect (nested zone / chain effects are folded into its line).
func (s *ContentService) effectLines(a *data.Ability, e *data.AbilityEffect, inZone string) []string {
	prefix := ""
	if e.Mode != "" {
		prefix = strings.ToUpper(e.Mode[:1]) + e.Mode[1:] + ": "
	}
	line := ""
	switch e.Type {
	case data.EffDamage:
		line = fmt.Sprintf("%s %s damage", num(e.Amount), e.DamageType)
		if inZone == "" {
			line = "Deals " + line + " to " + whom(a, e)
		}
	case data.EffHeal:
		line = fmt.Sprintf("heals %s HP", num(e.Amount))
		if inZone == "" {
			line = fmt.Sprintf("Heals %s for %s HP", whom(a, e), num(e.Amount))
		}
	case data.EffStatus:
		line = fmt.Sprintf("%s for %s s", statusText[e.Status], num(e.Duration))
		if inZone == "" {
			line = fmt.Sprintf("%s %s for %s s", upper(statusVerb[e.Status]), whom(a, e), num(e.Duration))
		}
	case data.EffKnockback:
		line = "Knocks back " + whom(a, e)
	case data.EffDispel:
		line = "Removes negative effects from " + whom(a, e)
	case data.EffBuff:
		line = fmt.Sprintf("%s for %s s", strings.Join(modsText(e.Modifiers), ", "), num(e.Duration))
		if inZone == "" {
			line = fmt.Sprintf("For %s s, %s: %s", num(e.Duration), whom(a, e), strings.Join(modsText(e.Modifiers), ", "))
		}
	case data.EffZone:
		var inner []string
		for i := range e.Effects {
			inner = append(inner, s.effectLines(a, &e.Effects[i], "zone")...)
		}
		line = fmt.Sprintf("An area of %s m for %s s", num(e.Radius), num(e.Duration))
		if len(inner) > 0 {
			line += fmt.Sprintf(": every %s s %s", num(e.Interval), strings.Join(inner, ", "))
		}
		if e.Smoke {
			line += "; hides the hordes inside from everyone outside"
		}
		out := []string{prefix + line}
		if e.Raise != "" {
			every := e.RaiseEvery
			name := e.Raise
			if h, ok := s.catalog.Hordes[e.Raise]; ok {
				name = h.Name
				if every <= 0 {
					every = h.Size
				}
			}
			r := fmt.Sprintf("Every %d enemies who die inside rise as %s", every, name)
			if e.RaiseDuration > 0 {
				r += fmt.Sprintf(" for %s s", num(e.RaiseDuration))
			}
			out = append(out, r)
		}
		if e.Channel {
			out = append(out, "Channelled: ends if the caster moves or dies")
		}
		return out
	case data.EffSummon:
		line = s.describeEffect(e)
		if e.PerTree {
			line += ", one per tree nearby"
		}
		if e.MaxAlive > 0 {
			line += fmt.Sprintf(" (up to %d at a time)", e.MaxAlive)
		}
		if u := s.summonedTrap(e.Horde); u != "" {
			line += "; " + u
		}
	case data.EffTree:
		n := max(e.Count, 1)
		line = fmt.Sprintf("Plants %d tree", n)
		if n > 1 {
			line += "s"
		}
	case data.EffReveal:
		line = fmt.Sprintf("Reveals %s m around the point for %s s", num(e.Radius), num(e.Duration))
	case data.EffBuild:
		name := e.Building
		if b, ok := s.catalog.Buildings[e.Building]; ok {
			name = b.Name
		}
		line = "Digs a tunnel entrance (" + name + ") next to the building, joined to the own tunnels"
	default:
		line = s.describeEffect(e)
	}
	if line == "" {
		return nil
	}
	return []string{prefix + line}
}

// summonedTrap: for a trap horde, what it does when stepped on.
func (s *ContentService) summonedTrap(horde string) string {
	h, ok := s.catalog.Hordes[horde]
	if !ok {
		return ""
	}
	u := s.catalog.Units[h.Unit]
	if u == nil || u.Trap == nil {
		return ""
	}
	blast := s.catalog.Abilities[u.Trap.Ability]
	var parts []string
	for i := range blast.Effects {
		for _, l := range s.effectLines(blast, &blast.Effects[i], "") {
			parts = append(parts, strings.ToLower(l[:1])+l[1:])
		}
	}
	return "an invisible trap: when an enemy steps on it, " + strings.Join(parts, ", ")
}
