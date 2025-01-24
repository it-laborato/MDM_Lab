package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/it-laborato/MDM_Lab/server/contexts/ctxerr"
	"github.com/it-laborato/MDM_Lab/server/mdmlab"
)

const (
	SecretVariablePrefix = "FLEET_SECRET_"
	SecretVariableMaxLen = 255
)

// //////////////////////////////////////////////////////////////////////////////
// Secret variables
// //////////////////////////////////////////////////////////////////////////////

type secretVariablesRequest struct {
	DryRun          bool                   `json:"dry_run"`
	SecretVariables []mdmlab.SecretVariable `json:"secrets"`
}

type secretVariablesResponse struct {
	Err error `json:"error,omitempty"`
}

func (r secretVariablesResponse) error() error { return r.Err }

func secretVariablesEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*secretVariablesRequest)
	err := svc.CreateSecretVariables(ctx, req.SecretVariables, req.DryRun)
	return secretVariablesResponse{Err: err}, nil
}

func (svc *Service) CreateSecretVariables(ctx context.Context, secretVariables []mdmlab.SecretVariable, dryRun bool) error {
	// Do authorization check first so that we don't have to worry about it later in the flow.
	if err := svc.authz.Authorize(ctx, &mdmlab.SecretVariable{}, mdmlab.ActionWrite); err != nil {
		return err
	}

	privateKey := svc.config.Server.PrivateKey
	if testSetEmptyPrivateKey {
		privateKey = ""
	}

	if len(privateKey) == 0 {
		return ctxerr.Wrap(ctx,
			&mdmlab.BadRequestError{Message: "Couldn't save secret variables. Missing required private key. Learn how to configure the private key here: https://mdmlabdm.com/learn-more-about/mdmlab-server-private-key"})
	}

	// Preprocess: strip FLEET_SECRET_ prefix from variable names
	for i, secretVariable := range secretVariables {
		secretVariables[i].Name = mdmlab.Preprocess(strings.TrimPrefix(secretVariable.Name, SecretVariablePrefix))
	}

	// Validation
	for _, secretVariable := range secretVariables {
		if len(secretVariable.Name) == 0 {
			return ctxerr.Wrap(ctx,
				mdmlab.NewInvalidArgumentError("name", "secret variable name cannot be empty"))
		}
		if len(secretVariable.Name) > SecretVariableMaxLen {
			return ctxerr.Wrap(ctx,
				mdmlab.NewInvalidArgumentError("name", fmt.Sprintf("secret variable name is too long: %s", secretVariable.Name)))
		}
	}

	if dryRun {
		return nil
	}

	if err := svc.ds.UpsertSecretVariables(ctx, secretVariables); err != nil {
		return ctxerr.Wrap(ctx, err, "saving secret variables")
	}
	return nil
}
