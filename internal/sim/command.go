package sim

import "dnd-rts/internal/fixed"

// CommandKind identifies what a command does.
type CommandKind uint8

const (
	CmdMove CommandKind = iota + 1
	CmdAttack
	CmdCast           // Hordes cast ability Name at Target (point) / TargetHorde (unit) / themselves
	CmdToggle         // Hordes switch toggle ability Name
	CmdAutocast       // Hordes set autocast of Name on (Count = 1) or off (0)
	CmdAttackBuilding // attack TargetBuilding
	CmdBuildOnPlot    // build Name on plot Count of TargetBuilding (a fortress / outpost)
	CmdCancelBuild    // cancel construction of TargetBuilding
	CmdBuildFree      // workers (Hordes) put up Name at Target facing Angle
	CmdConstruct      // workers (Hordes) help build the unfinished TargetBuilding
	CmdTrain          // queue horde type Name in TargetBuilding
	CmdCancelTrain    // remove queue entry Count of TargetBuilding (refund)
	CmdRally          // set TargetBuilding's rally point to Target
	CmdResearch       // research upgrade Name in TargetBuilding
	CmdStop           // Hordes stop and forget their orders
	CmdHold           // Hordes hold position (fight in reach, never chase)
	CmdAttackMove     // Hordes walk to Target, fighting enemies on the way
	CmdPatrol         // Hordes attack-move back and forth between here and Target
	CmdGate           // open (Count 1) or close (Count 0) gate TargetBuilding
	CmdAttackGround   // ranged Hordes shoot at the ground point Target until told otherwise

	// Developer commands (dev console). They go through the same command
	// stream so replays and lockstep stay deterministic.
	CmdDevSpawnHorde    // Name = horde type, Count hordes at Target for Player
	CmdDevSpawnBuilding // Name = building type at Target for Player (Count = rotation in degrees)
	CmdDevKill          // kill Hordes and the Buildings
	CmdDevHeal          // restore Hordes and the Buildings to full health
	CmdDevGive          // add Count of resource Name to Player (negative takes)
	CmdDevHit           // hit every soldier of Hordes (and the Buildings) for Count damage of type Name
	CmdDevXP            // give Count experience to the heroes among Hordes
)

// Command is the only way to change the world. In lockstep multiplayer the
// same command list is applied on every machine at the same tick.
type Command struct {
	Player PlayerID    `json:"player"`
	Kind   CommandKind `json:"kind"`
	Hordes []HordeID   `json:"hordes"`           // hordes the command applies to
	Target fixed.Vec2  `json:"target,omitempty"` // world position (CmdMove)

	TargetHorde    HordeID    `json:"targetHorde,omitempty"`    // enemy horde (CmdAttack)
	TargetBuilding BuildingID `json:"targetBuilding,omitempty"` // enemy building (CmdAttackBuilding)

	Angle fixed.Fixed `json:"angle,omitempty"` // building facing, radians (CmdBuildFree)

	// Developer commands.
	Name      string       `json:"name,omitempty"`
	Count     int          `json:"count,omitempty"`
	Buildings []BuildingID `json:"buildings,omitempty"`
}
