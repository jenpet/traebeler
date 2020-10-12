package main

import (
	"context"
	"github.com/jenpet/traebeler/internal"
	"github.com/jenpet/traebeler/internal/log"
	"os"
	"os/signal"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	listenCancel(cancel)
	internal.Do(ctx)
}

// listenCancel handles a graceful shutdown in case the os receives a cancel signal
func listenCancel(cancel context.CancelFunc) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		osSignal := <-c
		log.Printf("Received os signal '%+v'", osSignal)
		cancel()
	}()
}
