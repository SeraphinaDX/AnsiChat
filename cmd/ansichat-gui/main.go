// AnsiChat GUI is the shared desktop/Android relay frontend.
package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"

	"ansichat/internal/gui"
	buildversion "ansichat/internal/version"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

// Icon.png is the single source for the desktop window icon and Android/desktop
// package icon, so runtime and launcher artwork cannot drift apart.
//
//go:embed Icon.png
var iconData []byte

var icon = fyne.NewStaticResource("ansichat.png", iconData)

func main() {
	configPath := flag.String("config", "", "optional desktop relay TOML file (otherwise use GUI app storage)")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("AnsiChat GUI", buildversion.Current)
		return
	}

	// Keep Android upgrade identity and the existing private config/key storage.
	a := app.NewWithID("ca.cerberusgames.copperline")
	g, err := gui.New(a, *configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "AnsiChat GUI:", err)
		os.Exit(1)
	}
	g.SetIcon(icon)
	g.Run()
}
