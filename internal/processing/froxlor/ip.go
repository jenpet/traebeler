package froxlor

import (
	"io/ioutil"
	"net/http"
)

const url = "https://api.ipify.org?format=text"

type ipifyApi struct {}

func (api ipifyApi) ipv4() (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	ip, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(ip), nil
}