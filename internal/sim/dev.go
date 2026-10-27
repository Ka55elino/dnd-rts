package sim

import (
	"dnd-rts/internal/fixed"
)

// devSpawnGap is the spacing between hordes spawned by one dev command.
var devSpawnGap = fixed.FromInt(14)

// applyDev runs a developer command. Invalid arguments are ignored, like any
// other command (the console validates and reports errors before submitting).
func (w *World) applyDev(c Command) {
	switch c.Kind {
	case CmdDevSpawnHorde:
		if _, ok := w.Catalog.Hordes[c.Name]; !ok {
			return
		}
		center, ok := w.walkableTarget(c.Target)
		if !ok {
			return
		}
		facing := center.Neg().Angle()
		forward := fixed.FromAngle(facing)
		right := fixed.Vec2{X: forward.Y, Y: -forward.X}
		n := min(max(c.Count, 1), 50)
		for i := 0; i < n; i++ {
			// A row across the facing direction, centered on the target.
			pos := center.Add(right.Scale(devSpawnGap.MulInt(2*i - (n - 1)).DivInt(2)))
			if p, ok := w.walkableTarget(pos); ok {
				_, _ = w.SpawnHorde(c.Player, c.Name, p, facing)
			}
		}
	case CmdDevSpawnBuilding:
		facing := c.Target.Neg().Angle()
		if c.Count != 0 {
			facing = fixed.FromInt(c.Count).Mul(fixed.Pi).DivInt(180)
		}
		_, _ = w.SpawnBuilding(c.Player, c.Name, c.Target, facing)
	case CmdDevKill:
		for _, id := range c.Hordes {
			if h := w.Horde(id); h != nil {
				for _, s := range h.Soldiers {
					s.HP = 0
				}
			}
		}
		w.removeDead()
		for _, id := range c.Buildings {
			if b := w.Building(id); b != nil {
				b.HP = 0 // removed (and defeat checked) at the end of the tick
			}
		}
	case CmdDevGive:
		if r := w.Catalog.ResourceIndex(c.Name); r >= 0 {
			if e := w.economyOf(c.Player); e != nil {
				e.stock[r] = max(e.stock[r]+int64(c.Count), 0)
			}
		}
	case CmdDevHit:
		w.combatHash.build(w)
		dmg := fixed.FromInt(c.Count)
		for _, id := range c.Hordes {
			if h := w.Horde(id); h != nil {
				for _, s := range h.Soldiers {
					w.hitSoldier(nil, c.Player, s, dmg, c.Name, true)
				}
			}
		}
		for _, id := range c.Buildings {
			if b := w.Building(id); b != nil {
				w.hitBuilding(c.Player, b, dmg, c.Name, true)
			}
		}
		w.removeDead()
	case CmdDevXP:
		for _, id := range c.Hordes {
			if h := w.Horde(id); h != nil {
				h.gainXP(c.Count)
			}
		}
	case CmdDevHeal:
		for _, id := range c.Hordes {
			if h := w.Horde(id); h != nil {
				for _, s := range h.Soldiers {
					s.HP = s.MaxHP()
				}
			}
		}
		for _, id := range c.Buildings {
			if b := w.Building(id); b != nil {
				b.HP = b.Def.Health
			}
		}
	}
}
