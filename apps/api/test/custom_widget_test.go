package test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"miniot/apps/api/internal/models"
)

func TestCustomWidgetLifecycleAndTenancy(t *testing.T) {
	router, dbClients, _ := setupTestApp(t)
	defer dbClients.Pool.Close()

	suffixA := uuid.New().String()[:6]
	suffixB := uuid.New().String()[:6]

	adminEmailA := fmt.Sprintf("admin_cw_a_%s@org.io", suffixA)
	adminEmailB := fmt.Sprintf("admin_cw_b_%s@org.io", suffixB)
	password := "SecretPassword123!"

	// Register Org A
	resA := executeJSONRequest[models.AuthResponse](t, router, http.MethodPost, "/api/v1/auth/register", models.RegisterRequest{
		OrganizationName: "Widget Org A " + suffixA,
		Email:            adminEmailA,
		Password:         password,
	}, "")
	tokenA := resA.AccessToken

	// Register Org B
	resB := executeJSONRequest[models.AuthResponse](t, router, http.MethodPost, "/api/v1/auth/register", models.RegisterRequest{
		OrganizationName: "Widget Org B " + suffixB,
		Email:            adminEmailB,
		Password:         password,
	}, "")
	tokenB := resB.AccessToken

	// 1. Create Custom Widget in Org A
	createReq := models.CreateCustomWidgetRequest{
		Name:         "Custom Battery Gauge",
		Description:  "ThingsBoard animated battery status indicator",
		WidgetType:   "telemetry",
		TemplateHTML: `<div class="battery-card"><div class="level" id="battery-level"></div></div>`,
		StyleCSS:     `.battery-card { border-radius: 8px; padding: 16px; } .level { height: 24px; background: #22c55e; }`,
		ControllerJS: `self.onInit = function() { console.log('Initialized'); }; self.onDataUpdated = function() { console.log('Updated'); };`,
		SettingsSchema: map[string]interface{}{
			"showPercentage": true,
			"lowThreshold":   20,
		},
	}

	createResp := executeJSONRequest[struct {
		Message string              `json:"message"`
		Data    models.CustomWidget `json:"data"`
	}](t, router, http.MethodPost, "/api/v1/custom-widgets", createReq, tokenA)

	widgetID := createResp.Data.ID
	assert.NotEmpty(t, widgetID)
	assert.Equal(t, "Custom Battery Gauge", createResp.Data.Name)
	assert.Equal(t, `<div class="battery-card"><div class="level" id="battery-level"></div></div>`, createResp.Data.TemplateHTML)

	// 2. Get Custom Widget in Org A
	getResp := executeJSONRequest[struct {
		Data models.CustomWidget `json:"data"`
	}](t, router, http.MethodGet, "/api/v1/custom-widgets/"+widgetID, nil, tokenA)
	assert.Equal(t, widgetID, getResp.Data.ID)
	assert.Equal(t, "Custom Battery Gauge", getResp.Data.Name)

	// 3. Multitenancy verification: Org B cannot access Org A's custom widget
	getRespOrgB := sendRequest(router, http.MethodGet, "/api/v1/custom-widgets/"+widgetID, nil, tokenB)
	assert.Equal(t, http.StatusNotFound, getRespOrgB.Code)

	// 4. Update Custom Widget in Org A
	updatedName := "Enhanced Battery Gauge"
	updatedHTML := `<div class="battery-card-v2"></div>`
	updateReq := models.UpdateCustomWidgetRequest{
		Name:         &updatedName,
		TemplateHTML: &updatedHTML,
	}
	updateResp := executeJSONRequest[struct {
		Message string              `json:"message"`
		Data    models.CustomWidget `json:"data"`
	}](t, router, http.MethodPut, "/api/v1/custom-widgets/"+widgetID, updateReq, tokenA)
	assert.Equal(t, "Enhanced Battery Gauge", updateResp.Data.Name)
	assert.Equal(t, updatedHTML, updateResp.Data.TemplateHTML)

	// Verify update persisted
	getAfterUpdate := executeJSONRequest[struct {
		Data models.CustomWidget `json:"data"`
	}](t, router, http.MethodGet, "/api/v1/custom-widgets/"+widgetID, nil, tokenA)
	assert.Equal(t, "Enhanced Battery Gauge", getAfterUpdate.Data.Name)
	assert.Equal(t, updatedHTML, getAfterUpdate.Data.TemplateHTML)

	// 5. List custom widgets in Org A
	listRespA := executeJSONRequest[struct {
		Data  []models.CustomWidget `json:"data"`
		Count int                   `json:"count"`
	}](t, router, http.MethodGet, "/api/v1/custom-widgets", nil, tokenA)
	assert.GreaterOrEqual(t, listRespA.Count, 1)

	// Org B list should be 0
	listRespB := executeJSONRequest[struct {
		Data  []models.CustomWidget `json:"data"`
		Count int                   `json:"count"`
	}](t, router, http.MethodGet, "/api/v1/custom-widgets", nil, tokenB)
	assert.Equal(t, 0, listRespB.Count)

	// 6. Delete Custom Widget in Org A
	deleteResp := sendRequest(router, http.MethodDelete, "/api/v1/custom-widgets/"+widgetID, nil, tokenA)
	assert.Equal(t, http.StatusOK, deleteResp.Code)

	// Verify deleted
	getAfterDelete := sendRequest(router, http.MethodGet, "/api/v1/custom-widgets/"+widgetID, nil, tokenA)
	assert.Equal(t, http.StatusNotFound, getAfterDelete.Code)
}
