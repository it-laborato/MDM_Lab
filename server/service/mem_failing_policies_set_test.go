package service

import (
	"testing"

	"github.com/it-laborato/MDM_Lab/server/mdmlab/policytest"
)

func TestMemFailingPolicySet(t *testing.T) {
	m := NewMemFailingPolicySet()
	policytest.RunFailing1000hosts(t, m)
	m = NewMemFailingPolicySet()
	policytest.RunFailingBasic(t, m)
}
