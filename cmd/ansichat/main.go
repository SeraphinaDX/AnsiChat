package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"syscall"
	"time"

	"ansichat/internal/config"
	"ansichat/internal/irc"
	"ansichat/internal/relay"
	"ansichat/internal/tui"
	buildversion "ansichat/internal/version"
)

func main() {
	configPath := flag.String("config", config.DefaultPath(), "path to AnsiChat TOML configuration")
	memprofile := flag.String("memprofile", "", "write a heap profile and memory counters after 30 seconds")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("AnsiChat", buildversion.Current)
		return
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Fatalf("configuration not found: %s\nCopy config.example.toml there or run with -config=/path/to/config.toml", *configPath)
		}
		log.Fatalf("load configuration: %v", err)
	}

	if *memprofile != "" {
		// Create private output files before starting the terminal UI, so path
		// failures are reported cleanly and profiles never become world-readable.
		heap, err := os.OpenFile(*memprofile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			log.Fatal(err)
		}
		counters, err := os.OpenFile(*memprofile+".txt", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			heap.Close()
			log.Fatal(err)
		}
		go func() {
			defer heap.Close()
			defer counters.Close()
			time.Sleep(30 * time.Second)
			var before, after runtime.MemStats
			runtime.ReadMemStats(&before)
			runtime.GC()
			runtime.ReadMemStats(&after)
			_, _ = fmt.Fprintf(counters, "Before GC: HeapAlloc=%d HeapInuse=%d HeapIdle=%d HeapReleased=%d Sys=%d\nAfter GC: HeapAlloc=%d HeapInuse=%d HeapIdle=%d HeapReleased=%d Sys=%d StackInuse=%d Goroutines=%d\n", before.HeapAlloc, before.HeapInuse, before.HeapIdle, before.HeapReleased, before.Sys, after.HeapAlloc, after.HeapInuse, after.HeapIdle, after.HeapReleased, after.Sys, after.StackInuse, runtime.NumGoroutine())
			if err := pprof.WriteHeapProfile(heap); err != nil {
				_, _ = fmt.Fprintf(counters, "Heap profile failed: %v\n", err)
			}
		}()
	}

	switch cfg.Relay.ModeValue() {
	case "server":
		log.Printf("AnsiChat relay configuration: %s", *configPath)
		r, err := relay.NewServer(cfg)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("AnsiChat relay listening on %s", cfg.Relay.Listen)
		log.Printf("AnsiChat relay host key fingerprint: %s", r.HostKeyFingerprint())
		log.Printf("AnsiChat relay authorized keys: %s", config.ExpandPath(cfg.Relay.AuthorizedKeys))
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		if err := r.Run(ctx); err != nil {
			log.Fatal(err)
		}

	case "client":
		backend, err := relay.NewClient(cfg)
		if err != nil {
			log.Fatal(err)
		}
		app := tui.NewWithBackend(cfg, backend)
		app.SetRelayReconnectFactory(func() (irc.Backend, error) {
			return relay.NewClient(cfg)
		})
		if err := app.Run(); err != nil {
			log.Fatal(err)
		}

	default:
		app := tui.New(cfg)
		if err := app.Run(); err != nil {
			log.Fatal(err)
		}
	}
}
