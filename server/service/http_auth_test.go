package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com:it-laborato/MDM_Lab/pkg/mdmlabhttp"
	"github.com:it-laborato/MDM_Lab/server/mdmlab"
	"github.com:it-laborato/MDM_Lab/server/mock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogin(t *testing.T) {
	ds, users, server := setupAuthTest(t)
	loginTests := []struct {
		email    string
		status   int
		password string
	}{
		{
			email:    "admin1@example.com",
			password: testUsers["admin1"].PlaintextPassword,
			status:   http.StatusOK,
		},
		{
			email:    "user1@example.com",
			password: testUsers["user1"].PlaintextPassword,
			status:   http.StatusOK,
		},
		{
			email:    "nosuchuser@example.com",
			password: "nosuchuser",
			status:   http.StatusUnauthorized,
		},
		{
			email:    "admin1@example.com",
			password: "badpassword",
			status:   http.StatusUnauthorized,
		},
	}

	for _, tt := range loginTests {
		// test sessions
		testUser := users[tt.email]

		params := loginRequest{
			Email:    tt.email,
			Password: tt.password,
		}
		j, err := json.Marshal(&params)
		assert.Nil(t, err)

		requestBody := io.NopCloser(bytes.NewBuffer(j))
		resp, err := http.Post(server.URL+"/api/latest/mdmlab/login", "application/json", requestBody)
		require.Nil(t, err)
		assert.Equal(t, tt.status, resp.StatusCode)

		jsn := struct {
			User  *mdmlab.User         `json:"user"`
			Token string              `json:"token"`
			Err   []map[string]string `json:"errors,omitempty"`
		}{}
		err = json.NewDecoder(resp.Body).Decode(&jsn)
		require.Nil(t, err)

		if tt.status != http.StatusOK {
			assert.NotEqual(t, "", jsn.Err)
			continue // skip remaining tests
		}

		require.NotNil(t, jsn.User)
		assert.Equal(t, tt.email, jsn.User.Email)

		// ensure that a session was created for our test user and stored
		sessions, err := ds.ListSessionsForUser(context.Background(), testUser.ID)
		assert.Nil(t, err)
		assert.Len(t, sessions, 1)

		// ensure the session key is not blank
		assert.NotEqual(t, "", sessions[0].Key)

		// test logout
		req, _ := http.NewRequest("POST", server.URL+"/api/latest/mdmlab/logout", nil)
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", jsn.Token))
		client := mdmlabhttp.NewClient()
		resp, err = client.Do(req)
		require.Nil(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode, strconv.Itoa(tt.status))

		_, err = io.ReadAll(resp.Body)
		assert.Nil(t, err)

		// ensure that our user's session was deleted from the store
		sessions, err = ds.ListSessionsForUser(context.Background(), testUser.ID)
		assert.Nil(t, err)
		assert.Len(t, sessions, 0)
	}
}

func setupAuthTest(t *testing.T) (mdmlab.Datastore, map[string]mdmlab.User, *httptest.Server) {
	ds := new(mock.Store)
	var users []*mdmlab.User
	sessions := make(map[string]*mdmlab.Session)
	ds.NewUserFunc = func(ctx context.Context, user *mdmlab.User) (*mdmlab.User, error) {
		users = append(users, user)
		return user, nil
	}
	ds.SessionByKeyFunc = func(ctx context.Context, key string) (*mdmlab.Session, error) {
		return sessions[key], nil
	}
	ds.MarkSessionAccessedFunc = func(ctx context.Context, session *mdmlab.Session) error {
		s := sessions[session.Key]
		s.AccessedAt = time.Now()
		sessions[session.Key] = s
		return nil
	}
	ds.UserByIDFunc = func(ctx context.Context, id uint) (*mdmlab.User, error) {
		for _, user := range users {
			if user.ID == id {
				return user, nil
			}
		}
		return nil, errors.New("user not found")
	}
	ds.ListUsersFunc = func(ctx context.Context, opt mdmlab.UserListOptions) ([]*mdmlab.User, error) {
		return users, nil
	}
	ds.ListSessionsForUserFunc = func(ctx context.Context, id uint) ([]*mdmlab.Session, error) {
		var userSessions []*mdmlab.Session
		for _, session := range sessions {
			if session.UserID == id {
				userSessions = append(userSessions, session)
			}
		}
		return userSessions, nil
	}
	ds.SessionByIDFunc = func(ctx context.Context, id uint) (*mdmlab.Session, error) {
		for _, session := range sessions {
			if session.ID == id {
				return session, nil
			}
		}
		return nil, errors.New("session not found")
	}
	ds.DestroySessionFunc = func(ctx context.Context, session *mdmlab.Session) error {
		delete(sessions, session.Key)
		return nil
	}
	usersMap, server := RunServerForTestsWithDS(t, ds)
	ds.UserByEmailFunc = func(ctx context.Context, email string) (*mdmlab.User, error) {
		user := usersMap[email]
		return &user, nil
	}
	ds.NewSessionFunc = func(ctx context.Context, userID uint, sessionKeySize int) (*mdmlab.Session, error) {
		key := make([]byte, sessionKeySize)
		_, err := rand.Read(key)
		if err != nil {
			return nil, err
		}
		sessionKey := base64.StdEncoding.EncodeToString(key)
		session := &mdmlab.Session{
			UserID:     userID,
			Key:        sessionKey,
			AccessedAt: time.Now(),
		}
		sessions[sessionKey] = session
		return session, nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*mdmlab.AppConfig, error) {
		return &mdmlab.AppConfig{}, nil
	}
	ds.NewActivityFunc = func(
		ctx context.Context, user *mdmlab.User, activity mdmlab.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		return nil
	}
	return ds, usersMap, server
}

func getTestAdminToken(t *testing.T, server *httptest.Server) string {
	return getTestUserToken(t, server, "admin1")
}

func getTestUserToken(t *testing.T, server *httptest.Server, testUserId string) string {
	testUser := testUsers[testUserId]

	params := loginRequest{
		Email:    testUser.Email,
		Password: testUser.PlaintextPassword,
	}
	j, err := json.Marshal(&params)
	assert.Nil(t, err)

	requestBody := io.NopCloser(bytes.NewBuffer(j))
	resp, err := http.Post(server.URL+"/api/latest/mdmlab/login", "application/json", requestBody)
	require.Nil(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	jsn := struct {
		User  *mdmlab.User         `json:"user"`
		Token string              `json:"token"`
		Err   []map[string]string `json:"errors,omitempty"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&jsn)
	require.Nil(t, err)

	return jsn.Token
}

func TestNoHeaderErrorsDifferently(t *testing.T) {
	_, _, server := setupAuthTest(t)

	req, _ := http.NewRequest("GET", server.URL+"/api/latest/mdmlab/users", nil)
	client := mdmlabhttp.NewClient()
	resp, err := client.Do(req)
	require.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	jsn := struct {
		Message string              `json:"message"`
		Errs    []map[string]string `json:"errors,omitempty"`
		UUID    string              `json:"uuid"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&jsn)
	require.NoError(t, err)
	assert.Equal(t, "Authorization header required", jsn.Message)
	require.Len(t, jsn.Errs, 1)
	assert.Equal(t, "base", jsn.Errs[0]["name"])
	assert.Equal(t, "Authorization header required", jsn.Errs[0]["reason"])
	assert.NotEmpty(t, jsn.UUID)

	req, _ = http.NewRequest("GET", server.URL+"/api/latest/mdmlab/users", nil)
	req.Header.Add("Authorization", "Bearer AAAA")
	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	err = json.NewDecoder(resp.Body).Decode(&jsn)
	require.NoError(t, err)
	assert.Equal(t, "Authentication required", jsn.Message)
	require.Len(t, jsn.Errs, 1)
	assert.Equal(t, "base", jsn.Errs[0]["name"])
	assert.Equal(t, "Authentication required", jsn.Errs[0]["reason"])
	assert.NotEmpty(t, jsn.UUID)
}
