package test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"miniot/apps/api/internal/models"
)

func TestDashboardManagementAndTenancy(t *testing.T) {
	router, dbClients, _ := setupTestApp(t)
	defer dbClients.Pool.Close()

	suffixA := uuid.New().String()[:6]
	suffixB := uuid.New().String()[:6]

	adminEmailA := fmt.Sprintf("admin_dash_a_%s@org.io", suffixA)
	adminEmailB := fmt.Sprintf("admin_dash_b_%s@org.io", suffixB)
	password := "SecretPassword123!"

	// Register Org A
	resA := executeJSONRequest[models.AuthResponse](t, router, http.MethodPost, "/api/v1/auth/register", models.RegisterRequest{
		OrganizationName: "Dashboard Org A " + suffixA,
		Email:            adminEmailA,
		Password:         password,
	}, "")
	tokenA := resA.AccessToken

	// Register Org B
	resB := executeJSONRequest[models.AuthResponse](t, router, http.MethodPost, "/api/v1/auth/register", models.RegisterRequest{
		OrganizationName: "Dashboard Org B " + suffixB,
		Email:            adminEmailB,
		Password:         password,
	}, "")
	tokenB := resB.AccessToken

	// Register a device in Org A to target in widget config
	deviceID := "dev-gauge-" + suffixA
	sendRequest(router, http.MethodPost, "/api/v1/devices", models.CreateDeviceRequest{
		ID:         deviceID,
		Name:       "Boiler Sensor",
		DeviceType: "boiler",
	}, tokenA)

	// 1. Create Dashboard in Org A
	createReq := models.CreateDashboardRequest{
		Title:       "Boiler Monitoring Canvas",
		Description: "Real-time thermal monitoring and controls",
		IsDefault:   true,
		Configuration: &models.DashboardConfiguration{
			Widgets: []models.WidgetConfig{
				{
					ID:         "w-1",
					Type:       "gauge",
					Title:      "Boiler Pressure",
					DeviceID:   deviceID,
					DeviceName: "Boiler Sensor",
					MetricName: "pressure",
					ColSpan:    1,
					RowSpan:    1,
					Settings: map[string]interface{}{
						"min":  0,
						"max":  100,
						"unit": "bar",
					},
				},
				{
					ID:         "w-2",
					Type:       "chart",
					Title:      "Temperature Trends",
					DeviceID:   deviceID,
					DeviceName: "Boiler Sensor",
					MetricName: "temperature",
					ColSpan:    2,
					RowSpan:    1,
				},
			},
		},
	}

	createResp := executeJSONRequest[struct {
		Message string           `json:"message"`
		Data    models.Dashboard `json:"data"`
	}](t, router, http.MethodPost, "/api/v1/dashboards", createReq, tokenA)

	dashboardID := createResp.Data.ID
	assert.NotEmpty(t, dashboardID)
	assert.Equal(t, "Boiler Monitoring Canvas", createResp.Data.Title)
	assert.True(t, createResp.Data.IsDefault)
	assert.Len(t, createResp.Data.Configuration.Widgets, 2)

	// 2. List Dashboards for Org A
	listResp := executeJSONRequest[struct {
		Data  []models.Dashboard `json:"data"`
		Count int                `json:"count"`
	}](t, router, http.MethodGet, "/api/v1/dashboards", nil, tokenA)

	assert.GreaterOrEqual(t, listResp.Count, 1)
	assert.Equal(t, dashboardID, listResp.Data[0].ID)

	// 3. Multi-Tenant Isolation: Org B cannot access Org A's dashboard
	getRespB := sendRequest(router, http.MethodGet, "/api/v1/dashboards/"+dashboardID, nil, tokenB)
	assert.Equal(t, http.StatusNotFound, getRespB.Code)

	// 4. Update Dashboard in Org A (add a new widget)
	updatedTitle := "Updated Boiler Center"
	updatedConfig := models.DashboardConfiguration{
		Widgets: append(createReq.Configuration.Widgets, models.WidgetConfig{
			ID:         "w-3",
			Type:       "digital_card",
			Title:      "Current Steam Flow",
			DeviceID:   deviceID,
			MetricName: "steam_flow",
			ColSpan:    1,
			RowSpan:    1,
		}),
	}
	updateReq := models.UpdateDashboardRequest{
		Title:         &updatedTitle,
		Configuration: &updatedConfig,
	}

	updateResp := executeJSONRequest[struct {
		Message string           `json:"message"`
		Data    models.Dashboard `json:"data"`
	}](t, router, http.MethodPut, "/api/v1/dashboards/"+dashboardID, updateReq, tokenA)

	assert.Equal(t, "Updated Boiler Center", updateResp.Data.Title)
	assert.Len(t, updateResp.Data.Configuration.Widgets, 3)

	// 5. Set Default Dashboard
	setDefaultResp := sendRequest(router, http.MethodPost, "/api/v1/dashboards/"+dashboardID+"/default", nil, tokenA)
	assert.Equal(t, http.StatusOK, setDefaultResp.Code)

	// 6. Delete Dashboard
	deleteResp := sendRequest(router, http.MethodDelete, "/api/v1/dashboards/"+dashboardID, nil, tokenA)
	assert.Equal(t, http.StatusOK, deleteResp.Code)

	// Verify it's gone
	getAfterDelete := sendRequest(router, http.MethodGet, "/api/v1/dashboards/"+dashboardID, nil, tokenA)
	assert.Equal(t, http.StatusNotFound, getAfterDelete.Code)
}
