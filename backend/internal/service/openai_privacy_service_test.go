//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSelectChatGPTAccountInfo_PrefersPaidOverDefaultFree(t *testing.T) {
	result := map[string]any{
		"accounts": map[string]any{
			"org-free": map[string]any{
				"account": map[string]any{
					"is_default": true,
					"plan_type":  "free",
				},
				"entitlement": map[string]any{
					"expires_at": "2026-04-21T00:00:00Z",
				},
			},
			"org-plus": map[string]any{
				"account": map[string]any{
					"plan_type": "chatgpt_plus",
				},
				"entitlement": map[string]any{
					"expires_at": "2026-05-21T00:00:00Z",
				},
			},
		},
	}

	info := selectChatGPTAccountInfo(result, "")
	require.NotNil(t, info)
	require.Equal(t, "plus", info.PlanType)
	require.Equal(t, "2026-05-21T00:00:00Z", info.SubscriptionExpiresAt)
}

func TestSelectChatGPTAccountInfo_PrefersOrgMatchedPaidAccount(t *testing.T) {
	result := map[string]any{
		"accounts": map[string]any{
			"org-free": map[string]any{
				"account": map[string]any{
					"is_default": true,
					"plan_type":  "free",
				},
				"entitlement": map[string]any{
					"expires_at": "2026-04-21T00:00:00Z",
				},
			},
			"org-plus": map[string]any{
				"account": map[string]any{
					"plan_type": "chatgpt_plus",
				},
				"entitlement": map[string]any{
					"expires_at": "2026-05-21T00:00:00Z",
				},
			},
			"org-pro": map[string]any{
				"account": map[string]any{
					"plan_type": "chatgpt_pro",
				},
				"entitlement": map[string]any{
					"expires_at": "2026-06-21T00:00:00Z",
				},
			},
		},
	}

	info := selectChatGPTAccountInfo(result, "org-plus")
	require.NotNil(t, info)
	require.Equal(t, "plus", info.PlanType)
	require.Equal(t, "2026-05-21T00:00:00Z", info.SubscriptionExpiresAt)
}
