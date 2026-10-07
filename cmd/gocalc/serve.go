package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/rahuldr07/go-calc/internal/server"
)

const defaultAddr = ":8080"

func serve(args []string) (string, int, bool) {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	addr := flags.String("addr", defaultAddr, "")
	origin := flags.String("cors-origin", "", "")
	if err := flags.Parse(args); err != nil || flags.NArg() > 0 {
		return serveUsage, 2, false
	}
	srv := server.New(server.Config{Addr: *addr, CORSOrigin: *origin})
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	failed := make(chan error, 1)
	go func() { failed <- srv.ListenAndServe() }()
	log.Printf("gocalc serve: listening on %s", *addr)
	select {
	case err := <-failed:
		return err.Error(), 1, false
	case <-ctx.Done():
	}
	grace, cancel := context.WithTimeout(context.Background(), server.ShutdownGrace)
	defer cancel()
	if err := srv.Shutdown(grace); err != nil {
		return err.Error(), 1, false
	}
	if err := <-failed; !errors.Is(err, http.ErrServerClosed) {
		return err.Error(), 1, false
	}
	return "gocalc serve: stopped", 0, false
}

const serveUsage = "usage: gocalc serve [--addr :8080] [--cors-origin https://example.com]"
