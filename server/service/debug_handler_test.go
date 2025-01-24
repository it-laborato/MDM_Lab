package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/it-laborato/MDM_Lab/server/config"
	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockService struct {
	mock.Mock
	mdmlab.Service
}

func (m *mockService) GetSessionByKey(ctx context.Context, sessionKey string) (*mdmlab.Session, error) {
	args := m.Called(ctx, sessionKey)
	if ret := args.Get(0); ret != nil {
		return ret.(*mdmlab.Session), nil
	}
	return nil, args.Error(1)
}

func (m *mockService) UserUnauthorized(ctx context.Context, userId uint) (*mdmlab.User, error) {
	args := m.Called(ctx, userId)
	if ret := args.Get(0); ret != nil {
		return ret.(*mdmlab.User), nil
	}
	return nil, args.Error(1)
}

var testConfig = config.MDMlabConfig{
	Auth: config.AuthConfig{},
}

func TestDebugHandlerAuthenticationTokenMissing(t *testing.T) {
	handler := MakeDebugHandler(&mockService{}, testConfig, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "https://mdmlabdm.com/debug/pprof/profile", nil)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)
	assert.Equal(t, http.StatusUnauthorized, res.Code)
}

func TestDebugHandlerAuthenticationSessionInvalid(t *testing.T) {
	svc := &mockService{}
	svc.On(
		"GetSessionByKey",
		mock.Anything,
		"fake_session_key",
	).Return(nil, errors.New("invalid session"))

	handler := MakeDebugHandler(svc, testConfig, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "https://mdmlabdm.com/debug/pprof/profile", nil)
	req.Header.Add("Authorization", "BEARER fake_session_key")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)
	assert.Equal(t, http.StatusUnauthorized, res.Code)
}

func TestDebugHandlerAuthenticationSuccess(t *testing.T) {
	svc := &mockService{}
	svc.On(
		"GetSessionByKey",
		mock.Anything,
		"fake_session_key",
	).Return(&mdmlab.Session{UserID: 42, ID: 1}, nil)
	svc.On(
		"UserUnauthorized",
		mock.Anything,
		uint(42),
	).Return(&mdmlab.User{}, nil)

	handler := MakeDebugHandler(svc, testConfig, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "https://mdmlabdm.com/debug/pprof/cmdline", nil)
	req.Header.Add("Authorization", "BEARER fake_session_key")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)
	assert.Equal(t, http.StatusOK, res.Code)
}
