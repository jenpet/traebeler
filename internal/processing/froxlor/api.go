package froxlor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

const froxlorAPIPath = "/froxlor/api.php"

type froxlorApi struct {
	uri, key, secret string
	action apiAction
}

func (fa froxlorApi) findDomainZones(domain, record string) ([]zone, error) {
	body := zoneListBody{}
	err := fa.post(createFindBodyContent(domain, record), &body)
	return body.Data.List, err
}

func (fa froxlorApi) deleteDomainZone(domain, entryID string) error {
	body := responseBody{}
	return fa.post(createDeleteBodyContent(domain, entryID), &body)
}

func (fa froxlorApi) addDomainZone(domain, record, content, ttl, rtype string) error {
	body := responseBody{}
	return fa.post(createAddBodyContent(domain, record, content, ttl, rtype), &body)
}

func (fa froxlorApi) domainExists(fqn string) (bool, error) {
	body := listBody{}
	err := fa.post(createFindSubDomainBodyContent(fqn), &body)
	return body.Data.Count > 0, err
}

func (fa froxlorApi) addDomain(domain, subdomain string) error {
	body := responseBody{}
	return fa.post(createAddSubDomainContent(domain, subdomain), &body)
}

func (fa froxlorApi) post(content requestBodyContent, responseBody froxlorBody) error {
	body := requestBody{
		Header: requestBodyHeader{
			APIKey: fa.key,
			Secret: fa.secret,
		},
		Body:  content,
	}
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	uri := createURI(fa.uri, froxlorAPIPath)
	resp, err := fa.action(uri, "application/json", bytes.NewBuffer(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// parse response body to extract the "body status code" and the status message if applicable
	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	// there was a body which might be JSON and provide a status
	if len(b) > 0 {
		err = json.Unmarshal(b, &responseBody)
		if err != nil {
			return err
		}

		// check response HTTP status code and body status code
		if resp.StatusCode != http.StatusOK || responseBody.statusCode() != http.StatusOK {
			return fmt.Errorf("froxlor API HTTP response code is '%d' and body response code '%d' with reason '%s'",
				resp.StatusCode, responseBody.statusCode(), responseBody.statusMessage())
		}
	} else if resp.StatusCode != http.StatusNotModified {
		return fmt.Errorf("froxlor API returned no body http status code %d", resp.StatusCode)
	}
	return nil
}

func createURI(baseURI, apiPath string) string {
	uri := baseURI
	if strings.HasSuffix(uri, "/") {
		uri = uri[:len(uri)-1]
	}
	return uri + apiPath
}

type requestBody struct {
	Header requestBodyHeader  `json:"header"`
	Body requestBodyContent `json:"body"`
}

type requestBodyHeader struct {
	APIKey string `json:"apikey"`
	Secret string `json:"secret"`
}

type requestBodyContent struct {
	Command string `json:"command"`
	Params map[string]interface{} `json:"params"`
}

func createFindBodyContent(domain, record string) requestBodyContent {
	return requestBodyContent {
		Command: "DomainZones.listing",
		Params: map[string]interface{}{
			"domainname": domain,
			"sql_search": map[string]interface{}{
				"record": map[string]string{
					"op": "=",
					"value": record,
				},
			},
		},
	}
}

func createDeleteBodyContent(domain, entryID string) requestBodyContent {
	return requestBodyContent{
		Command: "DomainZones.delete",
		Params: map[string]interface{} {
			"domainname": domain,
			"entry_id": entryID,
		},
	}
}

// createAddBodyContent
// domain represents the TLD
// record is the subdomain
// content the IP which has to be mapped
// ttl time to live of the entry
// rtype the type of the record
func createAddBodyContent(domain, record, content, ttl, rtype string) requestBodyContent {
	return requestBodyContent{
		Command: "DomainZones.Add",
		Params:  map[string]interface{}{
			"domainname": domain,
			"record": record,
			"content": content,
			"ttl": ttl,
			"type": rtype,
		},
	}
}

func createFindSubDomainBodyContent(fqn string) requestBodyContent {
	return requestBodyContent{
		Command: "SubDomains.listing",
		Params:  map[string]interface{}{
			"sql_search": map[string]interface{}{
				// a sql join is done internally by froxlor so querying in a sql search requires a prefix due to arbitrary columns
				// Found out by actually reading the PHP code...
				"d.domain": map[string]string{
					"op": "=",
					"value": fqn,
				},
			},
		},
	}
}

func createAddSubDomainContent(domain, subdomain string) requestBodyContent {
	return requestBodyContent{
		Command: "SubDomains.add",
		Params:  map[string]interface{}{
			"domain": domain,
			"subdomain": subdomain,
		},
	}
}

type froxlorBody interface {
	statusCode() int
	statusMessage() string
}

type responseBody struct {
	Status int `json:"status"`
	StatusMessage string `json:"status_message"`
}

func (rb responseBody) statusCode() int {
	return rb.Status
}

func (rb responseBody) statusMessage() string {
	return rb.StatusMessage
}

type zoneListBody struct {
	responseBody
	Data struct{
		Count int `json:"count"`
		List []zone `json:"list"`
	} `json:"data"`
}

type listBody struct {
	responseBody
	Data struct {
		Count int `json:"count"`
	} `json:"data"`
}

type zone struct {
	ID string `json:"id"`
	DomainID string `json:"domain_id"`
	TTL string `json:"ttl"`
	Record string `json:"record"`
	Type string `json:"type"`
	Content string `json:"content"`
}

type apiAction func(url, contentType string, body io.Reader) (resp *http.Response, err error)