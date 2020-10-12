package froxlor

import (
	"errors"
	"gopkg.in/h2non/gock.v1"
	"gotest.tools/assert"
	"net/http"
	"testing"
)

func TestIPv4(t *testing.T) {
	defer gock.Off()
	ipTests := []struct{
		name, uri string
		reply func(*gock.Response)
		expected string
		errorExpected bool
	}{
		{
			"successful query",
			"https://api.ipify.org",
			func(response *gock.Response) {
				response.BodyString("127.0.0.1")
				response.Status(http.StatusOK) },
			"127.0.0.1",
			false,
		},
		{
			"failed query",
			"https://api.ipify.org",
			func(response *gock.Response) { response.Error = errors.New("foo") },
			"",
			true,
		},
	}
	defer gock.Off()
	api := ipifyApi{}
	for _, tt := range ipTests {
		t.Run(tt.name, func(t *testing.T) {
			gockIP(tt.uri, tt.reply)
			ip, err := api.ipv4()
			assert.Equal(t, tt.expected, ip, "result of ipv4 did not match")
			assert.Equal(t, tt.errorExpected, err != nil, "expected error did not match with returned error")
		})
	}
}

func gockIP(uri string, reply func(*gock.Response)) {
	gock.Clean()
	gock.New(uri).
		Get("/").
		MatchParam("format", "text").
		ReplyFunc(reply)
}
