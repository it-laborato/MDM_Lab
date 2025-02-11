//go:build !darwin

package profiles

import "github.com/it-laborato/MDM_Lab/server/mdmlab"

func GetFleetdConfig() (*mdmlab.MDMAppleMDMlabdConfig, error) {
	return nil, ErrNotImplemented
}

func IsEnrolledInMDM() (bool, string, error) {
	return false, "", ErrNotImplemented
}

func CheckAssignedEnrollmentProfile(expectedURL string) error {
	return ErrNotImplemented
}

func GetCustomEnrollmentProfileEndUserEmail() (string, error) {
	return "", ErrNotImplemented
}
