package usage

import (
	"net/http"
	"net/url"
	"path"

	"code.cloudfoundry.org/lager"
	"github.com/pkg/errors"

	"code.cloudfoundry.org/clock"
	uaa_go_client "code.cloudfoundry.org/uaa-go-client"
	"code.cloudfoundry.org/uaa-go-client/config"
)

const (
	UsageServiceURLParsingError                     = "error parsing Usage Service URL"
	GetUAAURLError                                  = "error getting UAA URL"
	CreateUAAClientError                            = "error creating UAA client"
	FetchUAATokenError                              = "error fetching UAA token"
	CreateUsageServiceHTTPRequestError              = "error creating HTTP request to usage service endpoint"
	UsageServiceRequestError                        = "error accessing usage service"
	UsageServiceUnexpectedResponseStatusErrorFormat = "unexpected status in usage service response: %d"
)

//go:generate counterfeiter . cfApiClient
type cfApiClient interface {
	GetUAAURL() (string, error)
}

type Collector struct {
	cfApiClient     cfApiClient
	usageServiceURL string
	clientID        string
	clientSecret    string
}

func NewCollector(cfClient cfApiClient, usageServiceURL, clientID, clientSecret string) *Collector {
	return &Collector{cfApiClient: cfClient, usageServiceURL: usageServiceURL, clientID: clientID, clientSecret: clientSecret}
}

func (c *Collector) Collect() error {
	usageURL, err := url.Parse(c.usageServiceURL)
	if err != nil {
		return errors.Wrapf(err, UsageServiceURLParsingError)
	}

	uaaURL, err := c.cfApiClient.GetUAAURL()
	if err != nil {
		return errors.Wrap(err, GetUAAURLError)
	}

	cfg := &config.Config{
		ClientName:       c.clientID,
		ClientSecret:     c.clientSecret,
		UaaEndpoint:      uaaURL,
		SkipVerification: true,
	}

	logger := lager.NewLogger("")
	uaaClient, err := uaa_go_client.NewClient(logger, cfg, clock.NewClock())
	if err != nil {
		return errors.Wrap(err, CreateUAAClientError)
	}

	token, err := uaaClient.FetchToken(true)
	if err != nil {
		return errors.Wrap(err, FetchUAATokenError)
	}

	usageURL.Path = path.Join(usageURL.Path, "system_report", "app_usages")

	req, err := http.NewRequest(http.MethodGet, usageURL.String(), nil)
	if err != nil {
		return errors.Wrap(err, CreateUsageServiceHTTPRequestError)
	}
	req.Header.Set("Authorization", "bearer "+token.AccessToken)

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrap(err, UsageServiceRequestError)
	}
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf(UsageServiceUnexpectedResponseStatusErrorFormat, resp.StatusCode)
	}

	return err
}
