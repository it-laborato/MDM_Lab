package mock

import (
	"context"

	"github.com/it-laborato/MDM_Lab/server/mdmlab"
)

func UserByEmailWithUser(u *mdmlab.User) UserByEmailFunc {
	return func(ctx context.Context, email string) (*mdmlab.User, error) {
		return u, nil
	}
}

func UserWithEmailNotFound() UserByEmailFunc {
	return func(ctx context.Context, email string) (*mdmlab.User, error) {
		return nil, &Error{"not found"}
	}
}

func UserWithID(u *mdmlab.User) UserByIDFunc {
	return func(ctx context.Context, id uint) (*mdmlab.User, error) {
		return u, nil
	}
}
