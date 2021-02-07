package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jenpet/traebeler/internal/test"
	"github.com/stretchr/testify/assert"
	"gopkg.in/h2non/gock.v1"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"
)

var envVars = map[string]string{
	"TRAEFIK_BASE_URI": "http://traefik.io",
	"TRAEBELER_LOG_LEVEL": "DEBUG",
	"TRAEBELER_LOOKUP_INTERVAL": "1",
	"TRAEBELER_PROCESSOR": "froxlor",
	"TRAEBELER_PROCESSOR_FROXLOR_URI": "http://froxlor.com",
	"TRAEBELER_PROCESSOR_FROXLOR_KEY": "FROXLOR-KEY",
	"TRAEBELER_PROCESSOR_FROXLOR_SECRET": "FROXLOR-SECRET",
}

func TestMain(m *testing.M) {
	defer test.ClearEnvs(test.SetEnvs(envVars))
	code := m.Run()
	os.Exit(code)
}

func TestIntegrationFroxlorProcessor_shouldQueryTraefikAndUpdateFroxlor(t *testing.T) {
	defer gock.Off()

	gock.New("http://traefik.io").
		Get("/api/http/routers").
		Reply(200).
		File("test/data/traefik/http_routers_response.json")

	gock.New("https://api.ipify.org").
		Get("/").
		MatchParam("format", "text").
		Reply(http.StatusOK).
		BodyString("127.0.0.1")

	gockFroxlor("DomainZones.listing", "test/data/froxlor/domainzone_listing_successful.json")
	gockFroxlor("DomainZones.delete", "test/data/froxlor/domainzone_delete_success.json")
	gockFroxlor("SubDomains.listing", "test/data/froxlor/subdomain_listing_successful.json")
	froxlorAdd := gockFroxlor("DomainZones.Add", "test/data/froxlor/domainzone_add_success.json")

	done := make(chan bool)

	go func() {
		for true {
			if froxlorAdd.Done() == true {
				done <- true
				return
			}
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())

	// perform listening and Froxlor triggering
	go func() {
		Do(ctx)
	}()

	for {
		select {
			case <-done:
				cancel()
				return
			case <-time.After(time.Second*30):
				cancel()
				assert.Fail(t, "integration test timed out")
		}
	}
}

func filterFroxlorCommand(command string) gock.FilterRequestFunc {
	return func(request *http.Request) bool {
		// body is read multiple times by the different matchers. GetBody does not clear the body byte array.
		reqBody, _ := request.GetBody()
		b, _ := ioutil.ReadAll(reqBody)
		var body struct{
			Header map[string]string `json:"header"`
			Body map[string]interface {
			} `json:"body"`
		}
		err := json.Unmarshal(b, &body)
		if err != nil {
			panic(err)
		}
		return fmt.Sprint(body.Body["command"]) == command
	}
}

func gockFroxlor(command, filename string) *gock.Response {
	return gock.New("http://froxlor.com").
		Post("/froxlor/api.php").
		Filter(filterFroxlorCommand(command)).
		Reply(http.StatusOK).
		File(filename)
}