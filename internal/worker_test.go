package internal

import (
	"context"
	"github.com/jenpet/traebeler/internal/test"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestProcessDomainsOnTrigger_whenClockFires_shouldQueryDomainsAndTriggerProcessor(t *testing.T) {
	tc := mockClock{}
	sProvider := staticProvider{"lospolloshermanos.com", "api.lospolloshermanos.com", "www.lospolloshermanos.com"}

	called := make(chan bool)
	ctx, cancel := context.WithCancel(context.Background())
	aProcessor := assertingProcessor{ t: t, expectedLen: 3, called: called}

	go func() {
		processDomainsOnTrigger(ctx, &aProcessor, sProvider, &tc)
	}()

	tc.Trigger()
	timeout := time.After(time.Second*1)
	select {
		case <- called:
			cancel()
			return
		case <- timeout:
			assert.Fail(t, "processor did not receive any domains")
			cancel()
			return
	}
}

func TestLoadConfig_whenEnvVarsAreInvalid_shouldPanic(t *testing.T) {
	configTests := []struct {
		name string
		vars map[string]string
	}{
		{"invalid interval value", map[string]string{"TRAEBELER_LOOKUP_INTERVAL":"1-.2"}},
		{"interval leq zero", map[string]string{"TRAEBELER_LOOKUP_INTERVAL": "0"}},
	}

	for _, tt := range configTests {
		t.Run(tt.name, func(t *testing.T) {
			defer test.ClearEnvs(test.SetEnvs(tt.vars))
			assert.Panics(t, func() {
				loadConfig()
			}, "invalid env vars should cause the worker to panic")
		})
	}
}

type mockClock struct {
	tickerChan chan time.Time
}

func (tc *mockClock) Ticker() <-chan time.Time {
	if tc.tickerChan == nil {
		tc.tickerChan = make(chan time.Time)
	}
	return tc.tickerChan
}

func (tc *mockClock) Trigger() {
	if tc.tickerChan == nil {
		tc.tickerChan = make(chan time.Time)
	}
	tc.tickerChan <- time.Now()
}

type staticProvider []string

func (sp staticProvider) GetDomains() []string {
	return sp
}

type assertingProcessor struct {
	t           *testing.T
	expectedLen int
	called      chan bool
}

func (ttp *assertingProcessor) Process(domains []string) {
	assert.Len(ttp.t, domains, ttp.expectedLen, "expected domain slice with certain length")
	ttp.called <- true
}

func (ttp *assertingProcessor) ID() string {
	return "asserting-processor"
}