package mysql

import (
	"context"
	"testing"

	"github.com:it-laborato/MDM_Lab/server/ptr"

	"github.com:it-laborato/MDM_Lab/server/mdmlab"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmailChanges(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"Confirm", testEmailChangesConfirm},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testEmailChangesConfirm(t *testing.T, ds *Datastore) {
	user := &mdmlab.User{
		Password:   []byte("foobar"),
		Email:      "bob@bob.com",
		GlobalRole: ptr.String(mdmlab.RoleObserver),
	}
	user, err := ds.NewUser(context.Background(), user)
	require.Nil(t, err)
	err = ds.PendingEmailChange(context.Background(), user.ID, "xxxx@yyy.com", "abcd12345")
	require.Nil(t, err)
	newMail, err := ds.ConfirmPendingEmailChange(context.Background(), user.ID, "abcd12345")
	require.Nil(t, err)
	assert.Equal(t, "xxxx@yyy.com", newMail)
	user, err = ds.UserByID(context.Background(), user.ID)
	require.Nil(t, err)
	assert.Equal(t, "xxxx@yyy.com", user.Email)
	// this should fail because it doesn't exist
	_, err = ds.ConfirmPendingEmailChange(context.Background(), user.ID, "abcd12345")
	assert.NotNil(t, err)

	// test that wrong user can't confirm e-mail change
	err = ds.PendingEmailChange(context.Background(), user.ID, "other@bob.com", "uniquetoken")
	require.Nil(t, err)
	otheruser, err := ds.NewUser(context.Background(), &mdmlab.User{
		Password:   []byte("supersecret"),
		Email:      "other@bobcom",
		GlobalRole: ptr.String(mdmlab.RoleObserver),
	})
	require.Nil(t, err)
	_, err = ds.ConfirmPendingEmailChange(context.Background(), otheruser.ID, "uniquetoken")
	assert.NotNil(t, err)
}
