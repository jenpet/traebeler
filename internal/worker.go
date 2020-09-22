package internal

import (
	"context"
	"github.com/jenpet/traebeler/processing"
	"github.com/jenpet/traebeler/traefik"
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
)

// Do will be called from main as an entrypoint.
func Do() {
	ctx, cancel := context.WithCancel(context.Background())
	listenCancel(cancel)
	perform(ctx)
}

// perform triggers the actual work on the provider and the processor.
func perform(ctx context.Context) {
	cfg := loadConfig()
	processor := getProcessor(cfg)
	provider := traefik.Provider()
	timer := configuredClock(cfg)
	workDomains(ctx, processor, provider, timer)
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

// workDomains queries traefik every time the clock's ticker fires. The retrieved domains are passed to the
// given processor not validating for any errors. The domains will be continuously fetched and forwarded until
// the context gets cancelled.
func workDomains(ctx context.Context, processor processor, provider provider, c clock) {
	log.Info("Started listening for domains...")
	for {
		select {
			case <-c.Ticker():
				processDomains(provider, processor)
		case <-ctx.Done():
			log.Info("Stopped listening for domains.")
			return
		}
	}
}

func processDomains(provider provider, processor processor) {
	log.Info("Querying for domains...")
	domains := provider.GetDomains()
	log.Infof("Done querying for domains. Received %v unique domains.", len(domains))
	processor.Process(domains)
}




func loadConfig() config {
	var cfg config
	err := envconfig.Process("traebeler", &cfg)
	if err != nil {
		log.Panicf("failed processing worker environment variables. Error: %s", err)
	}
	if !cfg.valid() {
		log.Panic("invalid worker config, either lookup interval is lte zero or the processor is undefined")
	}
	return cfg
}

func getProcessor(cfg config) processor {
	processor := processing.Repository().GetProcessor(cfg.Processor)
	if processor == nil {
		log.Panicf("failed looking up processor with id '%v'", cfg.Processor)
	}
	if err := processor.Init(); err != nil {
		log.Panicf("failed to initialize processor with ID '%v' due to error: %v", cfg.Processor, err)
	}
	return processor
}
