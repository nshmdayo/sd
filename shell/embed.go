// Package shell provides the embedded shell integration scripts.
package shell

import _ "embed"

//go:embed init.bash
var BashInit string

//go:embed init.zsh
var ZshInit string
