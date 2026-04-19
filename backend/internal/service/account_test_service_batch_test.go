//go:build unit

package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/stretchr/testify/require"
)

type batchTestHTTPUpstream struct {
	mu        sync.Mutex
	requests  map[int64]*http.Request
	responses map[int64]*http.Response
}

func (u *batchTestHTTPUpstream) Do(_ *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	return nil, fmt.Errorf("unexpected Do call")
}

func (u *batchTestHTTPUpstream) DoWithTLS(req *http.Request, _ string, accountID int64, _ int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	if u.requests == nil {
		u.requests = make(map[int64]*http.Request)
	}
	u.requests[accountID] = req

	if u.responses == nil {
		return nil, fmt.Errorf("no mocked responses configured")
	}
	resp, ok := u.responses[accountID]
	if !ok {
		return nil, fmt.Errorf("no mocked response for account %d", accountID)
	}
	return resp, nil
}

func TestAccountTestService_RunBatchTests_Persists429AndRecoversSuccess(t *testing.T) {
	repo := &openAIAccountTestRepo{
		mockAccountRepoForGemini: mockAccountRepoForGemini{
			accountsByID: map[int64]*Account{
				1: {
					ID:          1,
					Platform:    PlatformOpenAI,
					Type:        AccountTypeOAuth,
					Concurrency: 1,
					Credentials: map[string]any{
						"access_token": "token-1",
					},
				},
				2: {
					ID:          2,
					Platform:    PlatformOpenAI,
					Type:        AccountTypeOAuth,
					Concurrency: 1,
					Credentials: map[string]any{
						"access_token": "token-2",
					},
				},
			},
		},
	}

	successResp := newJSONResponse(http.StatusOK, "")
	successResp.Body = io.NopCloser(strings.NewReader(`data: {"type":"response.completed"}

`))
	successResp.Header.Set("x-codex-primary-used-percent", "88")
	successResp.Header.Set("x-codex-primary-reset-after-seconds", "604800")
	successResp.Header.Set("x-codex-primary-window-minutes", "10080")
	successResp.Header.Set("x-codex-secondary-used-percent", "42")
	successResp.Header.Set("x-codex-secondary-reset-after-seconds", "18000")
	successResp.Header.Set("x-codex-secondary-window-minutes", "300")

	rateLimitedResp := newJSONResponse(http.StatusTooManyRequests, `{"error":{"type":"usage_limit_reached","message":"limit reached"}}`)
	rateLimitedResp.Header.Set("x-codex-primary-used-percent", "100")
	rateLimitedResp.Header.Set("x-codex-primary-reset-after-seconds", "604800")
	rateLimitedResp.Header.Set("x-codex-primary-window-minutes", "10080")
	rateLimitedResp.Header.Set("x-codex-secondary-used-percent", "100")
	rateLimitedResp.Header.Set("x-codex-secondary-reset-after-seconds", "18000")
	rateLimitedResp.Header.Set("x-codex-secondary-window-minutes", "300")

	upstream := &batchTestHTTPUpstream{
		responses: map[int64]*http.Response{
			1: successResp,
			2: rateLimitedResp,
		},
	}

	svc := &AccountTestService{
		accountRepo:  repo,
		httpUpstream: upstream,
	}

	var recoveredMu sync.Mutex
	var recovered []int64
	results, err := svc.RunBatchTests(context.Background(), []int64{1, 2}, "gpt-5.4", func(_ context.Context, accountID int64) error {
		recoveredMu.Lock()
		recovered = append(recovered, accountID)
		recoveredMu.Unlock()
		return nil
	})

	require.NoError(t, err)
	require.Len(t, results, 2)
	require.Equal(t, int64(1), results[0].AccountID)
	require.Equal(t, "success", results[0].Status)
	require.Equal(t, int64(2), results[1].AccountID)
	require.Equal(t, "failed", results[1].Status)
	require.Equal(t, []int64{1}, recovered)
	require.Equal(t, int64(2), repo.rateLimitedID)
	require.NotNil(t, repo.rateLimitedAt)
	require.WithinDuration(t, time.Now().Add(7*24*time.Hour), *repo.rateLimitedAt, 2*time.Second)
}
