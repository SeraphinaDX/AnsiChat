package scripting

import (
	"testing"

	lua "github.com/yuin/gopher-lua"
)

func TestLegacyLuaAPIStillRegistersCommands(t *testing.T) {
	var printed []string
	r := newRuntime(Host{
		Active: func() Context { return Context{} },
		Print:  func(_, _, text string) { printed = append(printed, text) },
	})
	defer r.L.Close()
	if err := r.L.DoString(`
		assert(ansichat == copperline)
		copperline.command("legacy", function(ctx) ansichat.print("legacy works") end)
		ansichat.command("renamed", function(ctx) copperline.print("renamed works") end)
	`); err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"legacy", "renamed"} {
		if err := r.L.CallByParam(lua.P{Fn: r.commands[command], NRet: 0, Protect: true}); err != nil {
			t.Fatal(err)
		}
	}
	if len(printed) != 2 || printed[0] != "legacy works" || printed[1] != "renamed works" {
		t.Fatalf("unexpected Lua output: %v", printed)
	}
}
