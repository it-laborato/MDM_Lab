package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com:it-laborato/MDM_Lab/server/mdmlab"
	"github.com/stretchr/testify/assert"
)

type foreignKeyError struct{}

func (foreignKeyError) IsForeignKey() bool { return true }
func (foreignKeyError) Error() string      { return "" }

type alreadyExists struct{}

func (alreadyExists) IsExists() bool { return false }
func (alreadyExists) Error() string  { return "" }

type newAndExciting struct{}

func (newAndExciting) Error() string { return "" }

func TestHandlesErrorsCode(t *testing.T) {
	errorTests := []struct {
		name string
		err  error
		code int
	}{
		{
			"validation",
			mdmlab.NewInvalidArgumentError("a", "b"),
			http.StatusUnprocessableEntity,
		},
		{
			"permission",
			mdmlab.NewPermissionError("a"),
			http.StatusForbidden,
		},
		{
			"foreign key",
			foreignKeyError{},
			http.StatusUnprocessableEntity,
		},
		{
			"mail error",
			mailError{},
			http.StatusInternalServerError,
		},
		{
			"osquery error - invalid node",
			&osqueryError{nodeInvalid: true},
			http.StatusUnauthorized,
		},
		{
			"osquery error - valid node",
			&osqueryError{},
			http.StatusInternalServerError,
		},
		{
			"data not found",
			&notFoundError{},
			http.StatusNotFound,
		},
		{
			"already exists",
			alreadyExists{},
			http.StatusConflict,
		},
		{
			"status coder",
			mdmlab.NewAuthFailedError(""),
			http.StatusUnauthorized,
		},
		{
			"default",
			newAndExciting{},
			http.StatusInternalServerError,
		},
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			encodeError(context.Background(), tt.err, recorder)
			assert.Equal(t, recorder.Code, tt.code)
		})
	}
}
