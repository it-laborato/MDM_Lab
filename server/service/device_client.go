package service

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/it-laborato/MDM_Lab/pkg/retry"
	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	"github.com/it-laborato/MDM_Lab/server/ptr"
	"github.com/rs/zerolog/log"
)

// Device client is used consume the `device/...` endpoints and meant to be used by MDMlab Desktop
type DeviceClient struct {
	*baseClient

	// mdmlabAlternativeBrowserHost is an alternative host to use for the MDMlab Desktop URLs generated for the browser.
	//
	// This is needed when the host that Orbit will connect to is different from the host that will connect via the browser.
	mdmlabAlternativeBrowserHost string

	// if set and a request fails with ErrUnauthenticated, the client will call
	// this function to get a fresh token and retry if it returns a different,
	// non-empty token.
	invalidTokenRetryFunc func() string
}

// NewDeviceClient instantiates a new client to perform requests against device endpoints.
func NewDeviceClient(addr string, insecureSkipVerify bool, rootCA string, mdmlabClientCrt *tls.Certificate, mdmlabAlternativeBrowserHost string) (*DeviceClient, error) {
	capabilities := mdmlab.CapabilityMap{}
	baseClient, err := newBaseClient(addr, insecureSkipVerify, rootCA, "", mdmlabClientCrt, capabilities)
	if err != nil {
		return nil, err
	}

	return &DeviceClient{
		baseClient:                  baseClient,
		mdmlabAlternativeBrowserHost: mdmlabAlternativeBrowserHost,
	}, nil
}

// WithInvalidTokenRetry sets the function to call if a request fails with
// ErrUnauthenticated. The client will call this function to get a fresh token
// and retry if it returns a different, non-empty token.
func (dc *DeviceClient) WithInvalidTokenRetry(fn func() string) {
	log.Debug().Msg("setting invalid token retry hook")
	dc.invalidTokenRetryFunc = fn
}

// request performs the request, resolving the pathFmt that should contain a %s
// verb to be replaced with the token, or no verb at all if the token is "-"
// (the pathFmt is used as-is as path). It will retry if the request fails due
// to an invalid token and the invalidTokenRetryFunc field is set.
func (dc *DeviceClient) request(verb, pathFmt, token, query string, params interface{}, responseDest interface{}) error {
	const maxAttempts = 4
	var attempt int
	for {
		attempt++

		path := pathFmt
		if token != "-" {
			path = fmt.Sprintf(pathFmt, token)
		}
		reqErr := dc.requestAttempt(verb, path, query, params, responseDest)
		if attempt >= maxAttempts || dc.invalidTokenRetryFunc == nil || token == "-" || !errors.Is(reqErr, ErrUnauthenticated) {
			// no retry possible, return the result
			if reqErr != nil {
				log.Debug().Msgf("not retrying API error; attempt=%d, hook set=%t, token unset=%t, error is auth=%t",
					attempt, dc.invalidTokenRetryFunc != nil, token == "-", errors.Is(reqErr, ErrUnauthenticated))
			}
			return reqErr
		}

		delay := time.Duration(attempt) * time.Second
		log.Debug().Msgf("retrying API error in %s", delay)
		time.Sleep(delay)
		newToken := dc.invalidTokenRetryFunc()
		log.Debug().Msgf("retrying API error; token is different=%t", newToken != "" && newToken != token)
		if newToken != "" {
			token = newToken
		}
	}
}

func (dc *DeviceClient) requestAttempt(verb string, path string, query string, params interface{}, responseDest interface{}) error {
	var bodyBytes []byte
	var err error
	if params != nil {
		bodyBytes, err = json.Marshal(params)
		if err != nil {
			return fmt.Errorf("making request json marshalling : %w", err)
		}
	}
	request, err := http.NewRequest(
		verb,
		dc.url(path, query).String(),
		bytes.NewBuffer(bodyBytes),
	)
	if err != nil {
		return err
	}

	dc.setClientCapabilitiesHeader(request)
	response, err := dc.http.Do(request)
	if err != nil {
		return fmt.Errorf("%s %s: %w", verb, path, err)
	}
	defer response.Body.Close()

	return dc.parseResponse(verb, path, response, responseDest)
}

// BrowserTransparencyURL returns a URL for the browser that the server
// will use to redirect to the transparency URL configured by the user.
func (dc *DeviceClient) BrowserTransparencyURL(token string) string {
	transparencyURL := dc.baseClient.url("/api/latest/mdmlab/device/"+token+"/transparency", "")
	if dc.mdmlabAlternativeBrowserHost != "" {
		transparencyURL.Host = dc.mdmlabAlternativeBrowserHost
	}
	return transparencyURL.String()
}

// BrowserSelfServiceURL returns the "Self-service" URL for the browser.
func (dc *DeviceClient) BrowserSelfServiceURL(token string) string {
	selfServiceURL := dc.baseClient.url("/device/"+token+"/self-service", "")
	if dc.mdmlabAlternativeBrowserHost != "" {
		selfServiceURL.Host = dc.mdmlabAlternativeBrowserHost
	}
	return selfServiceURL.String()
}

// BrowserDeviceURL returns the "My device" URL for the browser.
func (dc *DeviceClient) BrowserDeviceURL(token string) string {
	deviceURL := dc.baseClient.url("/device/"+token, "")
	if dc.mdmlabAlternativeBrowserHost != "" {
		deviceURL.Host = dc.mdmlabAlternativeBrowserHost
	}
	return deviceURL.String()
}

// CheckToken checks if a token is valid by making an authenticated request to
// the server
func (dc *DeviceClient) CheckToken(token string) error {
	verb, path := "HEAD", "/api/latest/mdmlab/device/%s/ping"
	err := dc.request(verb, path, token, "", nil, nil)

	if errors.As(err, &notFoundErr{}) {
		// notFound is ok, it means an old server without the auth ping endpoint,
		// so we fall back to previously-used endpoint
		_, err = dc.DesktopSummary(token)
	}
	return err
}

// Ping sends a ping to the server using the device/ping endpoint
func (dc *DeviceClient) Ping() error {
	verb, path := "HEAD", "/api/mdmlab/device/ping"
	err := dc.request(verb, path, "-", "", nil, nil)

	if err == nil || errors.Is(err, notFoundErr{}) {
		// notFound is ok, it means an old server without the ping endpoint +
		// capabilities header
		return nil
	}

	return err
}

func (dc *DeviceClient) getListDevicePolicies(token string) ([]*mdmlab.HostPolicy, error) {
	verb, path := "GET", "/api/latest/mdmlab/device/%s/policies"
	var responseBody listDevicePoliciesResponse
	err := dc.request(verb, path, token, "", nil, &responseBody)
	return responseBody.Policies, err
}

func (dc *DeviceClient) getMinDesktopPayload(token string) (mdmlabDesktopResponse, error) {
	verb, path := "GET", "/api/latest/mdmlab/device/%s/desktop"
	var r mdmlabDesktopResponse
	err := dc.request(verb, path, token, "", nil, &r)
	return r, err
}

func (dc *DeviceClient) DesktopSummary(token string) (*mdmlabDesktopResponse, error) {
	r, err := dc.getMinDesktopPayload(token)
	if err == nil {
		r.FailingPolicies = ptr.Uint(uintValueOrZero(r.FailingPolicies))
		return &r, nil
	}

	if errors.Is(err, notFoundErr{}) {
		policies, err := dc.getListDevicePolicies(token)
		if err != nil {
			return nil, err
		}

		var failingPolicies uint
		for _, policy := range policies {
			if policy.Response != "pass" {
				failingPolicies++
			}
		}
		return &mdmlabDesktopResponse{
			DesktopSummary: mdmlab.DesktopSummary{
				FailingPolicies: ptr.Uint(failingPolicies),
			},
		}, nil
	}

	return nil, err
}

func (dc *DeviceClient) MigrateMDM(token string) error {
	verb, path := "POST", "/api/latest/mdmlab/device/%s/migrate_mdm"
	return dc.request(verb, path, token, "", nil, nil)
}

func (dc *DeviceClient) ReportError(token string, mdmlabdErr mdmlab.MDMlabdError) error {
	verb, path := "POST", "/api/latest/mdmlab/device/%s/debug/errors"
	req := mdmlabdErrorRequest{MDMlabdError: mdmlabdErr}
	return retry.Do(
		func() error {
			err := dc.request(verb, path, token, "", req, nil)
			if err != nil {
				return err
			}

			return nil
		},
		retry.WithMaxAttempts(3),
		retry.WithInterval(15*time.Second),
	)
}
