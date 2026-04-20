package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type dashboardAccountConsumptionRepoStub struct {
	service.UsageLogRepository
	items []usagestats.AccountConsumptionItem
	calls int
}

func (s *dashboardAccountConsumptionRepoStub) GetAccountConsumption(
	ctx context.Context,
	startTime, endTime time.Time,
) ([]usagestats.AccountConsumptionItem, error) {
	s.calls++
	return s.items, nil
}

func TestDashboardHandler_GetAccountConsumption_UsesCache(t *testing.T) {
	dashboardAccountConsumptionCache = newSnapshotCache(30 * time.Second)

	repo := &dashboardAccountConsumptionRepoStub{
		items: []usagestats.AccountConsumptionItem{
			{
				AccountID:   9,
				AccountName: "acc-prod-9",
				Requests:    7,
				TotalTokens: 1234,
				AccountCost: 5.67,
			},
		},
	}
	svc := service.NewDashboardService(repo, nil, nil, nil)
	handler := NewDashboardHandler(svc, nil)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/admin/dashboard/accounts", handler.GetAccountConsumption)

	req1 := httptest.NewRequest(http.MethodGet, "/admin/dashboard/accounts?start_date=2026-03-01&end_date=2026-03-07", nil)
	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, req1)

	require.Equal(t, http.StatusOK, rec1.Code)
	require.Equal(t, "miss", rec1.Header().Get("X-Snapshot-Cache"))
	require.Contains(t, rec1.Body.String(), "\"account_id\":9")
	require.Contains(t, rec1.Body.String(), "\"account_name\":\"acc-prod-9\"")
	require.Contains(t, rec1.Body.String(), "\"account_cost\":5.67")
	require.Equal(t, 1, repo.calls)

	req2 := httptest.NewRequest(http.MethodGet, "/admin/dashboard/accounts?start_date=2026-03-01&end_date=2026-03-07", nil)
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)

	require.Equal(t, http.StatusOK, rec2.Code)
	require.Equal(t, "hit", rec2.Header().Get("X-Snapshot-Cache"))
	require.Equal(t, 1, repo.calls)
}
