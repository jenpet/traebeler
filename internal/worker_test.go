package internal

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestListenDomains_whenClockFires_shouldQueryDomainsAndTriggerProcessor(t *testing.T) {
	tc := mockClock{}
	sProvider := staticProvider{"lospolloshermanos.com", "api.lospolloshermanos.com", "www.lospolloshermanos.com"}

	called := make(chan bool)
	ctx, cancel := context.WithCancel(context.Background())
	aProcessor := assertingProcessor{ t: t, expectedLen: 3, called: called}

	go func() {
		workDomains(ctx, &aProcessor, sProvider, &tc)
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

type mockClock struct {
	tickerChan chan time.Time
}

func (tc *mockClock) Stop() {}

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