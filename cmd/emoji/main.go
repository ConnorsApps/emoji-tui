package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/atotto/clipboard"
)

func runCLI(query string) int {
	results := filter(emojiData, buildSource(emojiData), query)
	if len(results) == 0 {
		fmt.Fprintln(os.Stderr, "no results")
		return 1
	}
	best := results[0].Emoji
	if err := clipboard.WriteAll(best.Char); err != nil {
		fmt.Fprintln(os.Stderr, "clipboard error:", err)
		return 1
	}
	fmt.Printf("copied %s (%s)\n", best.Char, best.Name)
	return 0
}

func main() {
	if len(os.Args) > 1 {
		os.Exit(runCLI(strings.Join(os.Args[1:], " ")))
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	p := tea.NewProgram(initialModel(), tea.WithContext(ctx))
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
