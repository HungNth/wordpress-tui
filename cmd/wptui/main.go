package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"wptui/internal/app"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	application := app.New(app.Options{})
	if err := application.RunWithContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
