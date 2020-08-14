package internal

import "time"

// clock is used to have a configuredClock based ticker channel and a possibility to stop it.
type clock interface {
	Ticker() <-chan time.Time
	Stop()
}

// provider provides a list of domains which can be used for processing.
type provider interface {
	GetDomains() []string
}

// processor works on a list of domains and identifies itself via an ID.
type processor interface {
	Process(domains []string)
	ID() string
}

// realClock implements the clock interface having a time.Ticker internally for re-occurring signals
type realClock struct {
	ticker *time.Ticker
}

func (rt *realClock) Ticker() <- chan time.Time {
	return rt.ticker.C
}

func (rt *realClock) Stop() {
	rt.ticker.Stop()
}

// config holds the general configuration for the overall application relying on "github.com/kelseyhightower/envconfig"
// by utilizing annotations for default values
type config struct {
	LookupInterval int `split_words:"true" default:"30"` // default of 30 secs
	Processor string `default:""`
}

func (wc config) valid() bool {
	return wc.LookupInterval > 0 && wc.Processor != ""
}

// configuredClock returns a clock based on the provided traebeler configuration
func configuredClock(cfg config) clock {
	return &realClock{ticker: time.NewTicker(time.Second * time.Duration(cfg.LookupInterval))}
}