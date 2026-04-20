package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func setupAccountPriorityRouter(adminSvc *stubAdminService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	accountHandler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router.PUT("/api/v1/admin/accounts/:id/group-priority", accountHandler.UpdateGroupPriority)
	return router
}

func TestAccountHandlerUpdateGroupPriority(t *testing.T) {
	adminSvc := newStubAdminService()
	router := setupAccountPriorityRouter(adminSvc)

	body := bytes.NewBufferString(`{"group_id":27,"priority":3}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/accounts/3/group-priority", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(3), adminSvc.lastGroupPriorityUpdate.accountID)
	require.Equal(t, int64(27), adminSvc.lastGroupPriorityUpdate.groupID)
	require.Equal(t, 3, adminSvc.lastGroupPriorityUpdate.priority)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, float64(0), resp["code"])
	data, ok := resp["data"].(map[string]any)
	require.True(t, ok)
	groups, ok := data["account_groups"].([]any)
	require.True(t, ok)
	require.Len(t, groups, 1)
	binding, ok := groups[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(27), binding["group_id"])
	require.Equal(t, float64(3), binding["priority"])
}
