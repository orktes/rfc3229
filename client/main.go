package main

import (
	"github.com/gopherjs/gopherjs/js"
	"github.com/orktes/rfc3229/client/patch"
)

func main() {
	js.Global.Set("BSDiff", map[string]interface{}{
		"Patch":      patch.PatchSlice,
		"MultiPatch": patch.MultiPatchSlice,
	})
}
