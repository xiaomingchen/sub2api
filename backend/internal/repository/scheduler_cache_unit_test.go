//go:build unit

package repository

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestBuildSchedulerMetadataAccount_KeepsOpenAIWSFlags(t *testing.T) {
	proxyID := int64(7)
	account := service.Account{
		ID:       42,
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeOAuth,
		ProxyID:  &proxyID,
		Proxy: &service.Proxy{
			ID:       proxyID,
			Name:     "edge-proxy",
			Protocol: "http",
			Host:     "proxy.internal",
			Port:     8080,
			Username: "user",
			Password: "pass",
			Status:   service.StatusActive,
		},
		Credentials: map[string]any{
			"api_key":      "visible",
			"access_token": "hidden",
		},
		Extra: map[string]any{
			"openai_oauth_responses_websockets_v2_enabled": true,
			"openai_oauth_responses_websockets_v2_mode":    service.OpenAIWSIngressModePassthrough,
			"openai_ws_force_http":                         true,
			"mixed_scheduling":                             true,
			"unused_large_field":                           "drop-me",
		},
	}

	got := buildSchedulerMetadataAccount(account)

	require.Equal(t, account.ProxyID, got.ProxyID)
	require.NotNil(t, got.Proxy)
	require.Equal(t, account.Proxy.ID, got.Proxy.ID)
	require.Equal(t, account.Proxy.URL(), got.Proxy.URL())
	require.Equal(t, "visible", got.GetCredential("api_key"))
	require.Empty(t, got.GetCredential("access_token"))
	require.Equal(t, true, got.Extra["openai_oauth_responses_websockets_v2_enabled"])
	require.Equal(t, service.OpenAIWSIngressModePassthrough, got.Extra["openai_oauth_responses_websockets_v2_mode"])
	require.Equal(t, true, got.Extra["openai_ws_force_http"])
	require.Equal(t, true, got.Extra["mixed_scheduling"])
	require.Nil(t, got.Extra["unused_large_field"])
}
