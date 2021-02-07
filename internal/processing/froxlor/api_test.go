package froxlor

import (
	"bytes"
	"errors"
	"github.com/stretchr/testify/assert"
	"io"
	"io/ioutil"
	"net/http"
	"testing"
)

var mf = mockFroxlor{}

var api = froxlorApi{
	uri:    "froxlor.localhost",
	key:    "key",
	secret: "secret",
	action: mf.mockAction,
}

func TestFind_shouldCallAndReturnListOfZones(t *testing.T) {
	findTests := []struct {
		name string
		mocks func()
		expectedZones []zone
		errorExpected bool
	}{
		{
			"list zones success response",
			func() {mf.mockResponse(http.StatusOK, "domainzone_listing_successful.json", nil) },
			[]zone{{ ID:       "201784", DomainID: "96", TTL:      "18000", Record:   "@", Type:     "A", Content:  "127.0.0.1"}},
			false,
		},
		{
			"list zones success response",
			func() {mf.mockResponse(http.StatusOK, "domainzone_listing_empty.json", nil) },
			[]zone{},
			false,
		},
		{
			"list zones not found response",
			func() {mf.mockResponse(http.StatusNotFound, "domainzone_listing_not_found.json", nil) },
			[]zone{},
			true,
		},
		{
			"http status and body differ",
			func() {mf.mockResponse(http.StatusNotFound, "domainzone_listing_empty.json", nil) },
			[]zone{},
			true,
		},
	}
	for _, tt := range findTests {
		mf.reset()
		tt.mocks()
		t.Run(tt.name, func(t *testing.T) {
			zones, err := api.findDomainZones("foo.bar", "@")
			assert.Equal(t, tt.errorExpected, err != nil, "expected error to be '%v' but was '%v'", tt.errorExpected, err != nil)
			if !tt.errorExpected {
				assert.Len(t, zones, len(tt.expectedZones))
				for _, expectedZone := range tt.expectedZones {
					assert.Contains(t, zones, expectedZone, "Expected zone was not in result set")
				}
			}
		})
	}
}

func TestDelete_shouldReturnErrorInCaseFailed(t *testing.T) {
	deleteTests := []struct{
		name string
		mocks func()
		errorExpected bool
	}{
		{
			name: "deleteDomainZone not modified response",
			mocks: func() {mf.mockResponse(http.StatusNotModified, "", nil) },
			errorExpected: false,
		},
		{
			name: "deleteDomainZone success response",
			mocks: func() {mf.mockResponse(http.StatusOK, "domainzone_delete_success.json", nil) },
			errorExpected: false,
		},
		{
			name: "deleteDomainZone internal server error",
			mocks: func() {mf.mockResponse(http.StatusInternalServerError, "", errors.New("error")) },
			errorExpected: true,
		},
	}

	for _, tt := range deleteTests {
		mf.reset()
		tt.mocks()
		t.Run(tt.name, func(t *testing.T) {
			err := api.deleteDomainZone("foo.bar", "id")
			assert.Equal(t, tt.errorExpected, err != nil, "expected error to be '%v' but was '%v'", tt.errorExpected, err)
		})
	}
}

func TestAdd_shouldReturnErrorInCaseFailed(t *testing.T) {
	addTests := []struct{
		name string
		mocks func()
		errorExpected bool
	}{
		{
			name: "addDomainZone zone success",
			mocks: func() {mf.mockResponse(http.StatusOK, "domainzone_add_success.json", nil) },
			errorExpected: false,
		},
		{
			name: "addDomainZone existing zone error",
			mocks: func() {mf.mockResponse(http.StatusInternalServerError, "domainzone_add_existing_error.json", nil) },
			errorExpected: true,
		},
		{
			name: "addDomainZone none existing domain",
			mocks: func() {mf.mockResponse(http.StatusInternalServerError, "domainzone_add_missing_domain.json", nil) },
			errorExpected: true,
		},
	}
	for _, tt := range addTests {
		mf.reset()
		tt.mocks()
		t.Run(tt.name, func(t *testing.T) {
			err := api.addDomainZone("foo.bar", "record", "127.0.0.1", "18000", "A")
			assert.Equal(t, tt.errorExpected, err != nil, "expected error to be '%v' but was '%v'", tt.errorExpected, err)
		})
	}
}

func TestCreateURI_whenURIHasTrailingSlash_shouldTrim(t *testing.T) {
	uriTests := []struct{
		name string
		baseUri string
	}{
		{
			"has slash suffix",
			"https://froxlor.com/",
		},
		{
			"does not have a slash suffix",
			"https://froxlor.com",
		},
	}

	for _, tt := range uriTests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, "https://froxlor.com/froxlor/api.php", createURI(tt.baseUri, froxlorAPIPath), "created froxlor URI is invalid")
		})
	}
}

type mockFroxlor struct {
	requests []actionRequest
	responses []actionResponse
}

func (mf *mockFroxlor) mockAction(url, contentType string, body io.Reader) (resp *http.Response, err error) {
	if mf.requests == nil {
		mf.requests = []actionRequest{}
	}
	mf.requests = append(mf.requests, actionRequest{
		contentType: contentType,
		body:        body,
	})
	reqID := len(mf.requests)-1

	if reqID <= len(mf.responses)-1 {
		return mf.responses[reqID].resp, mf.responses[reqID].err
	}
	return nil, nil
}

func (mf *mockFroxlor) mockResponse(statusCode int, bodyFile string, err error) {
	if mf.responses == nil {
		mf.responses = []actionResponse{}
	}
	b := []byte("")
	if bodyFile != "" {
		b, _ = ioutil.ReadFile("testdata/" + bodyFile)
	}
	resp := actionResponse {
		resp: &http.Response{StatusCode: statusCode, Body: ioutil.NopCloser(bytes.NewReader(b))},
		err:  err,
	}
	mf.responses = append(mf.responses, resp)
}

type actionRequest struct {
	contentType string
	body io.Reader
}

type actionResponse struct {
	resp *http.Response
	err error
}

func (mf *mockFroxlor) reset() {
	mf.requests = []actionRequest{}
	mf.responses = []actionResponse{}
}