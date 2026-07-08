package main

import (
	"errors"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rzaal/ribnip/internal/collection"
	"github.com/rzaal/ribnip/internal/curl"
	"github.com/rzaal/ribnip/internal/tui"
)

func main() {
	if err := curl.CheckAvailable(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "get cwd:", err)
		os.Exit(1)
	}

	coll, err := collection.Load(dir)
	if err != nil {
		var noColl *collection.ErrNoCollection
		if errors.As(err, &noColl) {
			fmt.Printf("No ribnip collection found in %s.\nScaffolding a new one...\n", dir)
			if err := collection.Scaffold(dir); err != nil {
				fmt.Fprintln(os.Stderr, "scaffold:", err)
				os.Exit(1)
			}
			coll, err = collection.Load(dir)
			if err != nil {
				fmt.Fprintln(os.Stderr, "load after scaffold:", err)
				os.Exit(1)
			}
		} else {
			fmt.Fprintln(os.Stderr, "load collection:", err)
			os.Exit(1)
		}
	}

	m := tui.New(coll)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error running program:", err)
		os.Exit(1)
	}
}
