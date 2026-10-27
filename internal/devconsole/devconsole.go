// Package devconsole parses developer console lines into simulation commands
// and session controls. It runs on the session's tick goroutine, so it may
// read the world directly; it never mutates it (changes go through commands,
// which keeps replays deterministic).
package devconsole

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"dnd-rts/internal/data"
	"dnd-rts/internal/fixed"
	"dnd-rts/internal/sim"
)

// Context is what the client knows when the line is sent.
type Context struct {
	Player   sim.PlayerID // who typed it
	Cursor   fixed.Vec2   // world position under the mouse
	Selected []sim.HordeID
}

// Control is the session-level part (not simulation state).
type Control interface {
	Paused() bool
	SetPaused(bool)
	Step(ticks int) // advance n ticks while paused
	Speed() float64
	SetSpeed(float64)
	Local() sim.PlayerID         // the player the client controls
	SetLocal(sim.PlayerID) error // take over another (AI) player
}

// Result of one line.
type Result struct {
	Commands []sim.Command // submit for the next tick
	Output   string
	Err      bool
}

type command struct {
	name, usage, help string
	run               func(e *env, args []string) (string, error)
}

type env struct {
	w   *sim.World
	ctl Control
	ctx Context
	out []sim.Command
}

var commands []command

func init() {
	commands = []command{
		{"help", "help [command]", "list commands or show one command's usage", cmdHelp},
		{"spawn", "spawn <horde> [count] [player]", "spawn hordes at the cursor (player: me | enemy | neutral | <id>)", cmdSpawn},
		{"build", "build <building> [player] [rotation°]", "place a finished building at the cursor (no placement rules)", cmdBuild},
		{"kill", "kill [all [player]]", "kill the selected hordes, or the building under the cursor; 'all' kills every horde (of a player)", cmdKill},
		{"heal", "heal [all]", "restore the selected hordes and the building under the cursor; 'all' heals everything", cmdHeal},
		{"hit", "hit <damageType> <amount>", "hit the selected hordes and the building under the cursor (fire, ice, lightning, acid, poison, necrotic, radiant, pierce, slash, crush)", cmdHit},
		{"xp", "xp <amount>", "give experience to the selected heroes (levels: 150, 400, 800, 1400 total)", cmdXP},
		{"list", "list hordes|buildings|types", "list live hordes, buildings, or the content types you can spawn", cmdList},
		{"pause", "pause", "pause the simulation", func(e *env, _ []string) (string, error) { e.ctl.SetPaused(true); return "paused", nil }},
		{"resume", "resume", "resume the simulation", func(e *env, _ []string) (string, error) { e.ctl.SetPaused(false); return "resumed", nil }},
		{"step", "step [ticks]", "advance a paused simulation by n ticks (default 1)", cmdStep},
		{"speed", "speed <x>", "simulation speed multiplier, 0.1–8 (1 = 20 ticks/s)", cmdSpeed},
		{"control", "control <player>", "take over another player (it stops being AI; yours becomes AI)", cmdControl},
		{"give", "give <resource> <amount> [player]", "add (or take, if negative) resources", cmdGive},
		{"res", "res [player]", "show a player's resources and food", cmdRes},
		{"hash", "hash", "current tick and state hash", func(e *env, _ []string) (string, error) {
			return fmt.Sprintf("tick %d  hash %016x", e.w.Tick, e.w.StateHash()), nil
		}},
	}
}

// Names returns all command names (for completion).
func Names() []string {
	names := make([]string, len(commands))
	for i, c := range commands {
		names[i] = c.name
	}
	return names
}

// Run executes one console line.
func Run(w *sim.World, ctl Control, ctx Context, line string) Result {
	args := strings.Fields(line)
	if len(args) == 0 {
		return Result{}
	}
	name := strings.ToLower(args[0])
	i := slices.IndexFunc(commands, func(c command) bool { return c.name == name })
	if i < 0 {
		return Result{Output: fmt.Sprintf("unknown command %q (try help)", args[0]), Err: true}
	}
	e := &env{w: w, ctl: ctl, ctx: ctx}
	out, err := commands[i].run(e, args[1:])
	if err != nil {
		return Result{Output: fmt.Sprintf("%v\nusage: %s", err, commands[i].usage), Err: true}
	}
	return Result{Commands: e.out, Output: out}
}

func cmdHelp(_ *env, args []string) (string, error) {
	var b strings.Builder
	for _, c := range commands {
		if len(args) > 0 && c.name != args[0] {
			continue
		}
		fmt.Fprintf(&b, "%-38s %s\n", c.usage, c.help)
	}
	if b.Len() == 0 {
		return "", fmt.Errorf("unknown command %q", args[0])
	}
	return strings.TrimRight(b.String(), "\n"), nil
}

func cmdSpawn(e *env, args []string) (string, error) {
	if len(args) < 1 {
		return "", fmt.Errorf("missing horde type")
	}
	def, ok := e.w.Catalog.Hordes[args[0]]
	if !ok {
		return "", fmt.Errorf("unknown horde %q (list types)", args[0])
	}
	count := 1
	if len(args) > 1 {
		n, err := strconv.Atoi(args[1])
		if err != nil || n < 1 || n > 50 {
			return "", fmt.Errorf("count must be 1–50")
		}
		count = n
	}
	player := e.ctx.Player
	if len(args) > 2 {
		p, err := e.player(args[2])
		if err != nil {
			return "", err
		}
		player = p
	}
	e.out = append(e.out, sim.Command{Player: player, Kind: sim.CmdDevSpawnHorde, Name: def.ID, Count: count, Target: e.ctx.Cursor})
	return fmt.Sprintf("spawned %d × %s for %s at %s", count, def.Name, playerName(player), pos(e.ctx.Cursor)), nil
}

func cmdBuild(e *env, args []string) (string, error) {
	if len(args) < 1 {
		return "", fmt.Errorf("missing building type")
	}
	def, ok := e.w.Catalog.Buildings[args[0]]
	if !ok {
		return "", fmt.Errorf("unknown building %q (list types)", args[0])
	}
	player := e.ctx.Player
	if len(args) > 1 {
		p, err := e.player(args[1])
		if err != nil {
			return "", err
		}
		player = p
	}
	rot := 0
	if len(args) > 2 {
		r, err := strconv.Atoi(args[2])
		if err != nil {
			return "", fmt.Errorf("rotation must be whole degrees")
		}
		rot = r % 360
		if rot == 0 {
			rot = 360 // 0 means "face the map center"
		}
	}
	if !e.w.Nav.Grid().WalkableAt(e.ctx.Cursor) {
		return "", fmt.Errorf("the cursor is on blocked ground")
	}
	e.out = append(e.out, sim.Command{Player: player, Kind: sim.CmdDevSpawnBuilding, Name: def.ID, Count: rot, Target: e.ctx.Cursor})
	return fmt.Sprintf("built %s for %s at %s", def.Name, playerName(player), pos(e.ctx.Cursor)), nil
}

func cmdHit(e *env, args []string) (string, error) {
	if len(args) < 2 {
		return "", fmt.Errorf("missing damage type or amount")
	}
	if !slices.Contains(data.DamageTypes, args[0]) {
		return "", fmt.Errorf("unknown damage type %q (%s)", args[0], strings.Join(data.DamageTypes, ", "))
	}
	n, err := strconv.Atoi(args[1])
	if err != nil || n < 1 || n > 100000 {
		return "", fmt.Errorf("amount must be 1–100000")
	}
	c := sim.Command{Player: e.ctx.Player, Kind: sim.CmdDevHit, Name: args[0], Count: n}
	for _, id := range e.ctx.Selected {
		if e.w.Horde(id) != nil {
			c.Hordes = append(c.Hordes, id)
		}
	}
	if b := e.buildingAt(e.ctx.Cursor); b != nil {
		c.Buildings = append(c.Buildings, b.ID)
	}
	if len(c.Hordes) == 0 && len(c.Buildings) == 0 {
		return "", fmt.Errorf("nothing to hit: select hordes or point at a building")
	}
	e.out = append(e.out, c)
	return fmt.Sprintf("%d %s → %d hordes, %d buildings", n, args[0], len(c.Hordes), len(c.Buildings)), nil
}

func cmdXP(e *env, args []string) (string, error) {
	if len(args) < 1 {
		return "", fmt.Errorf("missing amount")
	}
	n, err := strconv.Atoi(args[0])
	if err != nil || n <= 0 || n > 1_000_000 {
		return "", fmt.Errorf("amount must be a whole number 1–1000000")
	}
	c := sim.Command{Player: e.ctx.Player, Kind: sim.CmdDevXP, Count: n}
	for _, id := range e.ctx.Selected {
		if h := e.w.Horde(id); h != nil && h.IsHero() {
			c.Hordes = append(c.Hordes, id)
		}
	}
	if len(c.Hordes) == 0 {
		return "", fmt.Errorf("select a hero first")
	}
	e.out = append(e.out, c)
	return fmt.Sprintf("+%d XP → %d heroes", n, len(c.Hordes)), nil
}

func cmdKill(e *env, args []string) (string, error) {
	return e.targets(args, sim.CmdDevKill, "killed")
}

func cmdHeal(e *env, args []string) (string, error) {
	return e.targets(args, sim.CmdDevHeal, "healed")
}

// targets builds a kill/heal command for: all hordes (of a player), or the
// selection plus the building under the cursor.
func (e *env) targets(args []string, kind sim.CommandKind, verb string) (string, error) {
	c := sim.Command{Player: e.ctx.Player, Kind: kind}
	if len(args) > 0 && args[0] == "all" {
		filter := -1
		if len(args) > 1 {
			p, err := e.player(args[1])
			if err != nil {
				return "", err
			}
			filter = int(p)
		}
		for _, h := range e.w.Hordes {
			if filter < 0 || int(h.Owner) == filter {
				c.Hordes = append(c.Hordes, h.ID)
			}
		}
		if kind == sim.CmdDevHeal {
			for _, b := range e.w.Buildings {
				if filter < 0 || int(b.Owner) == filter {
					c.Buildings = append(c.Buildings, b.ID)
				}
			}
		}
	} else if len(args) > 0 {
		return "", fmt.Errorf("unexpected %q", args[0])
	} else {
		for _, id := range e.ctx.Selected {
			if e.w.Horde(id) != nil {
				c.Hordes = append(c.Hordes, id)
			}
		}
		if b := e.buildingAt(e.ctx.Cursor); b != nil {
			c.Buildings = append(c.Buildings, b.ID)
		}
	}
	if len(c.Hordes) == 0 && len(c.Buildings) == 0 {
		return "", fmt.Errorf("nothing to target: select hordes or point at a building")
	}
	e.out = append(e.out, c)
	return fmt.Sprintf("%s %d hordes, %d buildings", verb, len(c.Hordes), len(c.Buildings)), nil
}

func cmdList(e *env, args []string) (string, error) {
	what := "hordes"
	if len(args) > 0 {
		what = args[0]
	}
	var b strings.Builder
	switch what {
	case "hordes":
		for _, h := range e.w.Hordes {
			fmt.Fprintf(&b, "#%-4d %-12s %-14s %3d/%-3d %s\n", h.ID, playerName(h.Owner), h.Def.ID, len(h.Soldiers), h.Def.Size, pos(h.Pos))
		}
	case "buildings":
		for _, x := range e.w.Buildings {
			fmt.Fprintf(&b, "#%-4d %-12s %-14s %5d/%-5d %s\n", x.ID, playerName(x.Owner), x.Def.ID, max(x.HP, 0).Int(), x.Def.Health.Int(), pos(x.Pos))
		}
	case "types":
		fmt.Fprintf(&b, "hordes:    %s\n", strings.Join(e.w.Catalog.HordeIDs, " "))
		fmt.Fprintf(&b, "buildings: %s\n", strings.Join(e.w.Catalog.BuildingIDs, " "))
	default:
		return "", fmt.Errorf("list what? hordes, buildings or types")
	}
	if b.Len() == 0 {
		return "(none)", nil
	}
	return strings.TrimRight(b.String(), "\n"), nil
}

func cmdControl(e *env, args []string) (string, error) {
	if len(args) < 1 {
		return "", fmt.Errorf("missing player")
	}
	p, err := e.player(args[0])
	if err != nil {
		return "", err
	}
	if p == e.ctl.Local() {
		return fmt.Sprintf("already controlling %s", playerName(p)), nil
	}
	if err := e.ctl.SetLocal(p); err != nil {
		return "", err
	}
	f := e.w.FactionOf(p)
	name := "?"
	if f != nil {
		name = f.Name
	}
	return fmt.Sprintf("now controlling %s (%s); %s is AI", playerName(p), name, playerName(e.ctx.Player)), nil
}

func cmdGive(e *env, args []string) (string, error) {
	if len(args) < 2 {
		return "", fmt.Errorf("missing resource or amount")
	}
	r := e.w.Catalog.ResourceIndex(args[0])
	if r < 0 || e.w.Catalog.Resources[args[0]].Kind != data.ResourceStock {
		return "", fmt.Errorf("unknown stock resource %q (%s)", args[0], strings.Join(stockIDs(e.w), ", "))
	}
	n, err := strconv.Atoi(args[1])
	if err != nil || n < -1_000_000 || n > 1_000_000 {
		return "", fmt.Errorf("amount must be a whole number up to ±1000000")
	}
	player := e.ctx.Player
	if len(args) > 2 {
		if player, err = e.player(args[2]); err != nil {
			return "", err
		}
	}
	e.out = append(e.out, sim.Command{Player: player, Kind: sim.CmdDevGive, Name: args[0], Count: n})
	return fmt.Sprintf("%+d %s for %s", n, args[0], playerName(player)), nil
}

func cmdRes(e *env, args []string) (string, error) {
	player := e.ctx.Player
	if len(args) > 0 {
		p, err := e.player(args[0])
		if err != nil {
			return "", err
		}
		player = p
	}
	var parts []string
	for r, id := range e.w.Catalog.ResourceIDs {
		if e.w.Catalog.Resources[id].Kind == data.ResourceCap {
			parts = append(parts, fmt.Sprintf("%s %d/%d", id, e.w.CapUsed(player, r), e.w.CapProvided(player, r)))
		} else {
			parts = append(parts, fmt.Sprintf("%s %d", id, e.w.Stock(player, r)))
		}
	}
	return playerName(player) + ": " + strings.Join(parts, "  "), nil
}

func stockIDs(w *sim.World) []string {
	var out []string
	for _, id := range w.Catalog.ResourceIDs {
		if w.Catalog.Resources[id].Kind == data.ResourceStock {
			out = append(out, id)
		}
	}
	return out
}

func cmdStep(e *env, args []string) (string, error) {
	n := 1
	if len(args) > 0 {
		v, err := strconv.Atoi(args[0])
		if err != nil || v < 1 || v > 1000 {
			return "", fmt.Errorf("ticks must be 1–1000")
		}
		n = v
	}
	e.ctl.SetPaused(true)
	e.ctl.Step(n)
	return fmt.Sprintf("stepping %d ticks", n), nil
}

func cmdSpeed(e *env, args []string) (string, error) {
	if len(args) == 0 {
		return fmt.Sprintf("speed %.2gx", e.ctl.Speed()), nil
	}
	v, err := strconv.ParseFloat(args[0], 64)
	if err != nil || v < 0.1 || v > 8 {
		return "", fmt.Errorf("speed must be 0.1–8")
	}
	e.ctl.SetSpeed(v)
	return fmt.Sprintf("speed %.2gx", v), nil
}

// player parses me | enemy | neutral | <id>.
func (e *env) player(s string) (sim.PlayerID, error) {
	switch s {
	case "me":
		return e.ctx.Player, nil
	case "enemy":
		if e.ctx.Player == 0 {
			return 1, nil
		}
		return 0, nil
	case "neutral":
		return sim.Neutral, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 || n > 254 {
		return 0, fmt.Errorf("player must be me, enemy, neutral or 0–254")
	}
	return sim.PlayerID(n), nil
}

// buildingAt returns the building whose footprint (plus a small margin) contains p.
func (e *env) buildingAt(p fixed.Vec2) *sim.Building {
	for _, b := range e.w.Buildings {
		if b.Covers(p, fixed.FromInt(2)) {
			return b
		}
	}
	return nil
}

func playerName(p sim.PlayerID) string {
	if p == sim.Neutral {
		return "neutral"
	}
	return fmt.Sprintf("player %d", p)
}

func pos(p fixed.Vec2) string {
	return fmt.Sprintf("(%.0f, %.0f)", p.X.Float(), p.Y.Float())
}
