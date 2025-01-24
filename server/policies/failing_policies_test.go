package policies

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/it-laborato/MDM_Lab/server/contexts/ctxerr"
	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	"github.com/it-laborato/MDM_Lab/server/mock"
	"github.com/it-laborato/MDM_Lab/server/ptr"
	"github.com/it-laborato/MDM_Lab/server/service"
	kitlog "github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

func TestTriggerFailingPolicies(t *testing.T) {
	ds := new(mock.Store)

	// test that every configuration works - global for specific policy IDs,
	// per-team for specific policy IDs, different configurations for different
	// teams:
	//
	// pol-global-{1-3}: global policies (only 1 and 2 is enabled), ids 1-2-3
	// pol-teamA-{4-6}: team A policies (only 4 and 5 is enabled), ids 4-5-6
	// pol-teamB-{7-9}: team B policies (only 7 and 8 is enabled), ids 7-8-9
	// pol-teamC-10: team C policy, team does not exist, id 10
	// pol-unknown-11: policy that does not exist anymore, id 11
	// pol-teamD-{12-14}: team D policies (only 12 and 13 is enabled), ids 12-13-14
	// pol-teamE-15: team E policy, integration does not exist at the global level
	//
	// Global config uses the webhook, team A a Jira integration, team B a
	// Zendesk integration, team D a webhook.

	pols := map[uint]*mdmlab.PolicyData{
		1:  {ID: 1, Name: "pol-global-1"},
		2:  {ID: 2, Name: "pol-global-2"},
		3:  {ID: 3, Name: "pol-global-3"},
		4:  {ID: 4, Name: "pol-teamA-4", TeamID: ptr.Uint(1)},
		5:  {ID: 5, Name: "pol-teamA-5", TeamID: ptr.Uint(1)},
		6:  {ID: 6, Name: "pol-teamA-6", TeamID: ptr.Uint(1)},
		7:  {ID: 7, Name: "pol-teamB-7", TeamID: ptr.Uint(2)},
		8:  {ID: 8, Name: "pol-teamB-8", TeamID: ptr.Uint(2)},
		9:  {ID: 9, Name: "pol-teamB-9", TeamID: ptr.Uint(2)},
		10: {ID: 10, Name: "pol-teamC-10", TeamID: ptr.Uint(3)},
		12: {ID: 12, Name: "pol-teamD-12", TeamID: ptr.Uint(4)},
		13: {ID: 13, Name: "pol-teamD-13", TeamID: ptr.Uint(4)},
		14: {ID: 14, Name: "pol-teamD-14", TeamID: ptr.Uint(4)},
		15: {ID: 15, Name: "pol-teamE-15", TeamID: ptr.Uint(5)},
	}
	ds.PolicyFunc = func(ctx context.Context, id uint) (*mdmlab.Policy, error) {
		pd, ok := pols[id]
		if !ok {
			return nil, ctxerr.Wrap(ctx, sql.ErrNoRows)
		}
		return &mdmlab.Policy{PolicyData: *pd}, nil
	}

	teams := map[uint]*mdmlab.Team{
		1: {ID: 1, Name: "teamA", Config: mdmlab.TeamConfig{
			WebhookSettings: mdmlab.TeamWebhookSettings{
				FailingPoliciesWebhook: mdmlab.FailingPoliciesWebhookSettings{
					PolicyIDs: []uint{4, 5},
				},
			},
			Integrations: mdmlab.TeamIntegrations{
				Jira: []*mdmlab.TeamJiraIntegration{
					{URL: "http://j.com", ProjectKey: "A", EnableFailingPolicies: true},
				},
			},
		}},
		2: {ID: 2, Name: "teamB", Config: mdmlab.TeamConfig{
			WebhookSettings: mdmlab.TeamWebhookSettings{
				FailingPoliciesWebhook: mdmlab.FailingPoliciesWebhookSettings{
					PolicyIDs: []uint{7, 8},
				},
			},
			Integrations: mdmlab.TeamIntegrations{
				Zendesk: []*mdmlab.TeamZendeskIntegration{
					{URL: "http://z.com", GroupID: 1, EnableFailingPolicies: true},
				},
			},
		}},
		4: {ID: 4, Name: "teamD", Config: mdmlab.TeamConfig{
			WebhookSettings: mdmlab.TeamWebhookSettings{
				FailingPoliciesWebhook: mdmlab.FailingPoliciesWebhookSettings{
					Enable:    true,
					PolicyIDs: []uint{12, 13},
				},
			},
			Integrations: mdmlab.TeamIntegrations{
				Zendesk: []*mdmlab.TeamZendeskIntegration{
					{URL: "http://z.com", GroupID: 1, EnableFailingPolicies: false},
				},
				Jira: []*mdmlab.TeamJiraIntegration{
					{URL: "http://j.com", ProjectKey: "A", EnableFailingPolicies: false},
				},
			},
		}},
		5: {ID: 5, Name: "teamE", Config: mdmlab.TeamConfig{
			WebhookSettings: mdmlab.TeamWebhookSettings{
				FailingPoliciesWebhook: mdmlab.FailingPoliciesWebhookSettings{
					PolicyIDs: []uint{15},
				},
			},
			Integrations: mdmlab.TeamIntegrations{
				Zendesk: []*mdmlab.TeamZendeskIntegration{
					{URL: "http://notexist", GroupID: 999, EnableFailingPolicies: true},
				},
			},
		}},
	}
	ds.TeamFunc = func(ctx context.Context, id uint) (*mdmlab.Team, error) {
		tm, ok := teams[id]
		if !ok {
			return nil, ctxerr.Wrap(ctx, sql.ErrNoRows)
		}
		return tm, nil
	}

	// globally enable the webhook automation
	ac := &mdmlab.AppConfig{
		WebhookSettings: mdmlab.WebhookSettings{
			FailingPoliciesWebhook: mdmlab.FailingPoliciesWebhookSettings{
				Enable:    true,
				PolicyIDs: []uint{1, 2},
			},
		},
		Integrations: mdmlab.Integrations{
			Jira: []*mdmlab.JiraIntegration{
				{URL: "http://j.com", ProjectKey: "A", Username: "jirauser", APIToken: "secret"},
			},
			Zendesk: []*mdmlab.ZendeskIntegration{
				{URL: "http://z.com", GroupID: 1, Email: "zendesk@z.com", APIToken: "secret"},
			},
		},
		ServerSettings: mdmlab.ServerSettings{
			ServerURL: "https://mdmlab.example.com",
		},
	}

	ds.AppConfigFunc = func(ctx context.Context) (*mdmlab.AppConfig, error) {
		return ac, nil
	}

	// add a failing policy host for every known policy
	failingPolicySet := service.NewMemFailingPolicySet()
	for polID := range pols {
		err := failingPolicySet.AddHost(polID, mdmlab.PolicySetHost{
			ID:       polID, // use policy ID as host ID, does not matter in the test
			Hostname: fmt.Sprintf("host%d.example", polID),
		})
		require.NoError(t, err)
	}
	// add a failing policy for the unknown one
	err := failingPolicySet.AddHost(11, mdmlab.PolicySetHost{
		ID:       11,
		Hostname: "host11.example",
	})
	require.NoError(t, err)

	type policyAutomation struct {
		polID      uint
		automation FailingPolicyAutomationType
	}
	var triggerCalls []policyAutomation
	err = TriggerFailingPoliciesAutomation(context.Background(), ds, kitlog.NewNopLogger(), failingPolicySet, func(pol *mdmlab.Policy, cfg FailingPolicyAutomationConfig) error {
		triggerCalls = append(triggerCalls, policyAutomation{pol.ID, cfg.AutomationType})

		hosts, err := failingPolicySet.ListHosts(pol.ID)
		require.NoError(t, err)
		err = failingPolicySet.RemoveHosts(pol.ID, hosts)
		require.NoError(t, err)

		return nil
	})
	require.NoError(t, err)

	wantCalls := []policyAutomation{
		{1, FailingPolicyWebhook},
		{2, FailingPolicyWebhook},
		{4, FailingPolicyJira},
		{5, FailingPolicyJira},
		{7, FailingPolicyZendesk},
		{8, FailingPolicyZendesk},
		{12, FailingPolicyWebhook},
		{13, FailingPolicyWebhook},
	}
	// order of calls is undefined
	require.ElementsMatch(t, wantCalls, triggerCalls)

	// failing policies set is now cleared
	polSets, err := failingPolicySet.ListSets()
	require.NoError(t, err)
	var remainingHosts []uint
	for _, set := range polSets {
		hosts, err := failingPolicySet.ListHosts(set)
		require.NoError(t, err)
		for _, h := range hosts {
			remainingHosts = append(remainingHosts, h.ID)
		}
	}
	// there's one remaining host ID in the failing policy sets, and it's the one
	// with the invalid integration (it did not remove the failing policy set so
	// that it can retry once the integration is fixed).
	require.Len(t, remainingHosts, 1)
	require.Equal(t, remainingHosts[0], uint(15)) // host id used is the same as the policy id

	// trigger it again, should cause the same calls as the first time, but all
	// policy sets should be empty (no host to process).
	var countHosts int
	triggerCalls = triggerCalls[:0]
	err = TriggerFailingPoliciesAutomation(context.Background(), ds, kitlog.NewNopLogger(), failingPolicySet, func(pol *mdmlab.Policy, cfg FailingPolicyAutomationConfig) error {
		hosts, err := failingPolicySet.ListHosts(pol.ID)
		require.NoError(t, err)
		countHosts += len(hosts)
		triggerCalls = append(triggerCalls, policyAutomation{pol.ID, cfg.AutomationType})

		return nil
	})
	require.NoError(t, err)

	// order of calls is undefined
	require.ElementsMatch(t, wantCalls, triggerCalls)
	require.Zero(t, countHosts)
}
