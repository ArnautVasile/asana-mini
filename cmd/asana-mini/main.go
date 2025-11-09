package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"context"

	"github.com/ArnautVasile/asana-mini/internal/asana"
	"github.com/ArnautVasile/asana-mini/internal/config"
)

func main() {
	short := flag.Bool("short_interval", false, "run every 30 seconds")
	long := flag.Bool("long_interval", false, "run every 5 minutes")
	flag.Parse()

	if *short && *long {
		fmt.Fprintln(os.Stderr, "choose exactly one: --short_interval OR --long_interval")
		os.Exit(2)
	}
	if !*short && !*long {
		fmt.Fprintln(os.Stderr, "missing interval flag: use --short_interval or --long_interval")
		os.Exit(2)
	}

	interval := 5 * time.Minute
	if *short {
		interval = 30 * time.Second
	}

	cfg, err := config.Load()
	must(err)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = asana.Run(ctx, interval, cfg)
	}()

	<-ctx.Done()
	fmt.Println("signal received, shutting down…")
	<-done
	fmt.Println("stopped")
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
