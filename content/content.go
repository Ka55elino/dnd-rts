// Package content embeds the game data (units, hordes, factions, buildings, maps) into the binary.
// The JSON files next to this one are the source of truth; the editor edits them.
package content

import "embed"

//go:embed units hordes factions buildings abilities upgrades maps resources.json combat.json plots.json
var FS embed.FS
