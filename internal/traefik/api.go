package traefik

import (
	"encoding/json"
	"fmt"
	traefik "github.com/containous/traefik/v2/pkg/config/runtime"
	"github.com/containous/traefik/v2/pkg/rules"
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"strings"
)

func Provider() traefikAPI {
	var cfg traefikConfig
	err := envconfig.Process("traefik", &cfg)
	if err != nil {
		log.Panic("Failed loading traefik configuration.")
	}
	return traefikAPI{baseURI: cfg.BaseURI}
}

// GetDomains queries the traefik API for all of its routers and their respective rules
// to return an effective list of domains as strings. All routers which are enabled will be used for domain extraction.
func GetDomains(baseURI string) []string {
	return retrieveDomains(traefikAPI{baseURI: baseURI}.getRouters)
}

type listRouters func() ([]traefik.RouterInfo, error)

type traefikAPI struct {
	baseURI string
}

func (ta traefikAPI) GetDomains() []string {
	return retrieveDomains(ta.getRouters)
}

// improve testing
func (ta traefikAPI) getRouters() (routerInfos []traefik.RouterInfo, err error) {
	uri := fmt.Sprintf("%v/api/http/routers", ta.baseURI)
	res, err := http.Get(uri)
	if err != nil {
		log.Errorf("Failed to communicate with traefik API on destination '%v'. Error: %v", ta.baseURI, err)
		return
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Errorf("Failed to parse traefik response body into byte array. Error: %v", err)
		return
	}

	err = json.Unmarshal(body, &routerInfos)
	if err != nil {
		log.Errorf("Failed to convert traefik response in RouterInfo list. Error: %v", err)
		return
	}

	log.Debugf("Received %v listRouters from traefik.", len(routerInfos))
	return
}

func retrieveDomains(fn listRouters) []string {
	routers := getEnabledRouterRules(fn)
	return extractEffectiveDomains(routers)
}

func getEnabledRouterRules(fn listRouters) (routers []string) {
	routerList, err := fn()

	if err != nil {
		log.Errorf("An error occurred while retrieving listRouters, won't extract any rules. Error: %s", err)
		return
	}

	for _, router := range routerList {
		if router.Status != traefik.StatusEnabled {
			log.Debugf("Won't process rule %s since router for service %s has status %s. Error (optional): %s",
				router.Rule, router.Service, router.Status, strings.Join(router.Err, ","))
			continue
		}
		routers = append(routers, router.Rule)
	}
	return
}

func extractEffectiveDomains(routerRules []string) (domains []string) {
	for _, rule := range routerRules {
		parsed, err := rules.ParseDomains(rule)
		if err != nil {
			log.Errorf("Could not parse domain(s) from rule \"%s\". Error: %s", rule, err)
			continue
		}
		for _, domain := range parsed {
			domains = appendIfNotExists(domains, domain)
		}
	}
	return
}

func appendIfNotExists(haystack []string, needle string) []string {
	for _, element := range haystack {
		if element == needle {
			return haystack
		}
	}
	return append(haystack, needle)
}

type traefikConfig struct {
	BaseURI string `split_words:"true"`
}