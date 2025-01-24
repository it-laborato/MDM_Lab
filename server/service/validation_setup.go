package service

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com:it-laborato/MDM_Lab/server/contexts/ctxerr"
	"github.com:it-laborato/MDM_Lab/server/mdmlab"
)

func (mw validationMiddleware) NewAppConfig(ctx context.Context, payload mdmlab.AppConfig) (*mdmlab.AppConfig, error) {
	invalid := &mdmlab.InvalidArgumentError{}
	var serverURLString string
	if payload.ServerSettings.ServerURL == "" {
		invalid.Append("server_url", "missing required argument")
	} else {
		serverURLString = cleanupURL(payload.ServerSettings.ServerURL)
	}
	if err := validateServerURL(serverURLString); err != nil {
		invalid.Append("server_url", err.Error())
	}
	if invalid.HasErrors() {
		return nil, ctxerr.Wrap(ctx, invalid)
	}
	return mw.Service.NewAppConfig(ctx, payload)
}

func validateServerURL(urlString string) error {
	serverURL, err := url.Parse(urlString)
	if err != nil {
		return err
	}

	if serverURL.Scheme != "https" && !strings.Contains(serverURL.Host, "localhost") {
		return errors.New("url scheme must be https")
	}

	return nil
}
