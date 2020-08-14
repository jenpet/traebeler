package traefik

import (
	"errors"
	"fmt"
	"github.com/containous/traefik/v2/pkg/config/dynamic"
	traefik "github.com/containous/traefik/v2/pkg/config/runtime"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestGetEnabledRouterRules_whenSomeRoutersNotEnabled_shouldOnlyReturnEnabledRouterRules(t *testing.T) {
	tp := createTestProvider()
	rules := getEnabledRouterRules(tp.list)
	assert.Len(t, rules, 2, "there should only be listRouters in the result which are enabled")
	assert.Equal(t, "Host(`api.lospolloshermanos.com`,`ww.lospolloshermanos.com`,`lospolloshermanos.com`)", rules[0], "result should contain rules of listRouters")
}

func TestGetEnabledRouterRules_whenAnErrorOccurred_shouldNotReturnAnyRouterRules(t *testing.T) {
	tp := createTestProvider()
	tp.err = errors.New("error stuff")
	rules := getEnabledRouterRules(tp.list)
	assert.Empty(t, rules, "there should be no router rules returned when an error occurs")
}

func TestExtractEffectiveDomains_shouldReturnListWithoutDuplicates(t *testing.T) {
	rules := []string{
		createTestHostRule("lospolloshermanos.com", "api.lospolloshermanos.com", "ww.lospolloshermanos.com", "lospolloshermanos.com"),
		createTestHostRule("lospolloshermanos.com"),
	}
	domains := extractEffectiveDomains(rules)
	assert.Len(t, domains, 3, "there should not be any duplicates in the list")
	assert.Contains(t, domains, "lospolloshermanos.com")
	assert.Contains(t, domains, "api.lospolloshermanos.com")
	assert.Contains(t, domains, "ww.lospolloshermanos.com")
}

func createTestProvider() testProvider {
	return testProvider {
		routerList: []traefik.RouterInfo{
			createTestRouterInfo(traefik.StatusDisabled, []string{}, []string{"veridian-dynamics.com"}),
			createTestRouterInfo(traefik.StatusWarning, []string{"error1"}, []string{"dundermifflinpaper.com"}),
			createTestRouterInfo(traefik.StatusEnabled, []string{}, []string{"api.lospolloshermanos.com", "ww.lospolloshermanos.com", "lospolloshermanos.com"}),
			createTestRouterInfo(traefik.StatusEnabled, []string{}, []string{"bettercallsaul.com"}),
		},
	}
}

func createTestRouterInfo(status string, err []string, hosts []string) traefik.RouterInfo {
	return traefik.RouterInfo {
		Router: &dynamic.Router{
			Service:     "default-service",
			Rule:        createTestHostRule(hosts...),
		},
		Err:    err,
		Status: status,
	}
}

func createTestHostRule(hosts ...string) string {
	for i, host := range hosts {
		hosts[i] = "`" + host + "`"
	}
	return fmt.Sprintf("Host(%v)", strings.Join(hosts, ","))
}

type testProvider struct {
	routerList []traefik.RouterInfo
	err error
}

func (tp testProvider) list() ([]traefik.RouterInfo, error) {
	return tp.routerList, tp.err
}
