package internal

import (
	"context"
	"github.com/jenpet/traebeler/internal/log"
	"github.com/jenpet/traebeler/internal/processing"
	"github.com/jenpet/traebeler/internal/traefik"
	"github.com/kelseyhightower/envconfig"
)

// Do will be called from main as an entrypoint.
func Do(ctx context.Context) {
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

// workDomains initially queries traefik for a first set of domains. The retrieved domains are passed to the
// given processor _not_ validating for any errors. Subsequently it will query traefik every time the clock's ticker fires.
// The domains will be continuously fetched and forwarded until the context gets cancelled.
func workDomains(ctx context.Context, processor processor, provider provider, c clock) {
	log.Info("Started listening for domains...")
	processDomains(provider, processor)
	processDomainsOnTrigger(ctx, processor, provider, c)
}

func processDomainsOnTrigger(ctx context.Context, processor processor, provider provider, c clock) {
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

// processDomains retrieves a list of domains from the given provider and forwards it to the processor.
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
