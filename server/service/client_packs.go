package service

import (
	"net/url"

	"github.com/it-laborato/MDM_Lab/server/mdmlab"
)

// ApplyPacks sends the list of Packs to be applied (upserted) to the
// MDMlab instance.
func (c *Client) ApplyPacks(specs []*mdmlab.PackSpec) error {
	req := applyPackSpecsRequest{Specs: specs}
	verb, path := "POST", "/api/latest/mdmlab/spec/packs"
	var responseBody applyPackSpecsResponse
	return c.authenticatedRequest(req, verb, path, &responseBody)
}

// GetPackSpec retrieves information about a pack in apply spec format.
func (c *Client) GetPackSpec(name string) (*mdmlab.PackSpec, error) {
	verb, path := "GET", "/api/latest/mdmlab/spec/packs/"+url.PathEscape(name)
	var responseBody getPackSpecResponse
	err := c.authenticatedRequest(nil, verb, path, &responseBody)
	return responseBody.Spec, err
}

// GetPacksSpecs retrieves the list of all Packs in apply specs format.
func (c *Client) GetPacksSpecs() ([]*mdmlab.PackSpec, error) {
	verb, path := "GET", "/api/latest/mdmlab/spec/packs"
	var responseBody getPackSpecsResponse
	err := c.authenticatedRequest(nil, verb, path, &responseBody)
	return responseBody.Specs, err
}

// ListPacks retrieves the list of all Packs.
func (c *Client) ListPacks() ([]*mdmlab.Pack, error) {
	verb, path := "GET", "/api/latest/mdmlab/packs"
	var responseBody listPacksResponse
	if err := c.authenticatedRequest(nil, verb, path, &responseBody); err != nil {
		return nil, err
	}

	packs := make([]*mdmlab.Pack, 0, len(responseBody.Packs))
	for _, pr := range responseBody.Packs {
		pack := pr.Pack
		packs = append(packs, &pack)
	}
	return packs, nil
}

// DeletePack deletes the pack with the matching name.
func (c *Client) DeletePack(name string) error {
	verb, path := "DELETE", "/api/latest/mdmlab/packs/"+url.PathEscape(name)
	var responseBody deletePackResponse
	return c.authenticatedRequest(nil, verb, path, &responseBody)
}
