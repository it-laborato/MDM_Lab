package service

import (
	"fmt"
	"net/url"

	"github.com/it-laborato/MDM_Lab/server/mdmlab"
)

// ApplyQueries sends the list of Queries to be applied (upserted) to the
// MDMlab instance.
func (c *Client) ApplyQueries(specs []*mdmlab.QuerySpec) error {
	req := applyQuerySpecsRequest{Specs: specs}
	verb, path := "POST", "/api/latest/mdmlab/spec/queries"
	var responseBody applyQuerySpecsResponse
	return c.authenticatedRequest(req, verb, path, &responseBody)
}

// GetQuerySpec returns the query spec of a query by its team+name.
func (c *Client) GetQuerySpec(teamID *uint, name string) (*mdmlab.QuerySpec, error) {
	verb, path := "GET", "/api/latest/mdmlab/spec/queries/"+url.PathEscape(name)
	query := url.Values{}
	if teamID != nil {
		query.Set("team_id", fmt.Sprint(*teamID))
	}
	var responseBody getQuerySpecResponse
	err := c.authenticatedRequestWithQuery(nil, verb, path, &responseBody, query.Encode())
	return responseBody.Spec, err
}

// GetQueries retrieves the list of all Queries.
func (c *Client) GetQueries(teamID *uint, name *string) ([]mdmlab.Query, error) {
	verb, path := "GET", "/api/latest/mdmlab/queries"
	query := url.Values{}
	if teamID != nil {
		query.Set("team_id", fmt.Sprint(*teamID))
	}
	if name != nil {
		query.Set("query", *name)
	}
	var responseBody listQueriesResponse
	err := c.authenticatedRequestWithQuery(nil, verb, path, &responseBody, query.Encode())
	if err != nil {
		return nil, err
	}
	return responseBody.Queries, nil
}

// DeleteQuery deletes the query with the matching name.
func (c *Client) DeleteQuery(name string) error {
	verb, path := "DELETE", "/api/latest/mdmlab/queries/"+url.PathEscape(name)
	var responseBody deleteQueryResponse
	return c.authenticatedRequest(nil, verb, path, &responseBody)
}

// DeleteQueries deletes several queries.
func (c *Client) DeleteQueries(ids []uint) error {
	req := deleteQueriesRequest{IDs: ids}
	verb, path := "POST", "/api/latest/mdmlab/queries/delete"
	var responseBody deleteQueriesResponse
	return c.authenticatedRequest(req, verb, path, &responseBody)
}
