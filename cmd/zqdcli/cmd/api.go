package cmd

import (
	"fmt"
	"net/url"

	"github.com/brimsec/zq/zqd/api"
)

// API wraps the client library and adds a few CLI-ish methods
// such as checking if a space exists.
type API struct {
	*api.Connection
}

// newAPI creates a new client API object and parses the URL, returning
// an error if the URL is not valid.  The object does not contact the server
// until Connect is called.
func newAPI(u string) (*API, error) {
	url, err := url.Parse(u)
	if err != nil {
		return nil, err
	}
	if url.Port() == "" {
		url.Host += fmt.Sprintf(":%d", api.DefaultPort)
	}
	c := api.NewConnectionToURL("ZQD-CLI", url)
	return &API{c}, nil
}

func (a API) Native() *api.Connection {
	return a.Connection
}

func (a API) SpaceExists(name string) (bool, error) {
	_, err := a.SpaceInfo(name)
	if err == nil {
		return true, nil
	}
	if err == api.ErrSpaceNotFound {
		return false, nil
	}
	return false, err
}
