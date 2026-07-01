// Package assets embeds the authoritative Lua game engine into the Go binary so
// the game ships as a single executable with no external file dependency.
package assets

import "embed"

//go:embed lua
var LuaFS embed.FS
