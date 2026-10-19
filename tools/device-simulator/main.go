package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

const (
	DefaultAPIURL       = "http://localhost:8080/api/v1"
	DefaultEMQXBroker   = "tcp://localhost:1883"
	DefaultAdminEmail   = "admin@miniot.io"
	DefaultAdminPass    = "AdminPassword123!"
	DefaultDeviceCount  = 1
	DefaultIntervalSec  = 3
	DefaultDevicePrefix = "sim-sensor"
	HTTPTimeoutDuration = 8 * time.Second
	MQTTConnectTimeout  = 5 * time.Second
	MQTTDisconnectGrace = 250
	BaseTemperature     = 22.0
	TempVarianceRange   = 6.0
	BaseHumidity        = 50.0
	HumidityVariance    = 15.0
	TopicTemplate       = "devices/%s/telemetry"
)

type Config struct {
	APIBaseURL   string
	BrokerURI    string
	AdminEmail   string
	AdminPass    string
	DeviceCount  int
	Interval     time.Duration
	DevicePrefix string
	Duration     time.Duration
}

type DeviceSession struct {
	DeviceID string
	Token    string
	Client   paho.Client
	BaseTemp float64
	BaseHum  float64
}

type AuthResponse struct {
	AccessToken string `json:"access_token"`
}

type ProvisionResponse struct {
	Token string `json:"token"`
}

type TelemetryPayload struct {
	DeviceID  string             `json:"device_id"`
	Timestamp string             `json:"timestamp"`
	Metrics   map[string]float64 `json:"metrics"`
}

func main() {
	cfg := parseFlags()
	fmt.Println("==================================================================")
	fmt.Println("             minIoT Multi-Device Telemetry Simulator              ")
	fmt.Println("==================================================================")
	fmt.Printf("[Config] API URL: %s | Broker: %s\n", cfg.APIBaseURL, cfg.BrokerURI)
	fmt.Printf("[Config] Devices: %d | Interval: %s | Prefix: %s\n", cfg.DeviceCount, cfg.Interval, cfg.DevicePrefix)

	adminToken := authenticateUser(cfg.APIBaseURL, cfg.AdminEmail, cfg.AdminPass)
	fmt.Println("[Auth] Successfully logged in as admin to minIoT API.")

	sessions := provisionAndConnectDevices(cfg, adminToken)
	defer disconnectAllSessions(sessions)

	runSimulation(cfg, sessions)
}

func parseFlags() *Config {
	apiURL := flag.String("api", getEnvOrDefault("API_URL", DefaultAPIURL), "API base URL")
	broker := flag.String("broker", getEnvOrDefault("EMQX_BROKER", DefaultEMQXBroker), "EMQX broker URI")
	email := flag.String("email", getEnvOrDefault("API_EMAIL", DefaultAdminEmail), "Admin login email")
	pass := flag.String("password", getEnvOrDefault("API_PASSWORD", DefaultAdminPass), "Admin login password")
	devices := flag.Int("devices", DefaultDeviceCount, "Number of simulated devices to provision and run")
	flag.IntVar(devices, "n", DefaultDeviceCount, "Shorthand for -devices")
	intervalSec := flag.Int("interval", DefaultIntervalSec, "Telemetry publishing interval in seconds")
	prefix := flag.String("prefix", DefaultDevicePrefix, "Prefix for simulated device IDs")
	durationStr := flag.String("duration", "0s", "Total simulation run duration (e.g. 30s, 5m, or 0s for indefinite)")

	flag.Parse()

	count := *devices
	if count < 1 {
		count = 1
	}

	duration, err := time.ParseDuration(*durationStr)
	if err != nil {
		duration = 0
	}

	return &Config{
		APIBaseURL:   *apiURL,
		BrokerURI:    *broker,
		AdminEmail:   *email,
		AdminPass:    *pass,
		DeviceCount:  count,
		Interval:     time.Duration(*intervalSec) * time.Second,
		DevicePrefix: *prefix,
		Duration:     duration,
	}
}

func authenticateUser(apiURL, email, password string) string {
	payload := map[string]string{
		"email":    email,
		"password": password,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", apiURL+"/auth/login", bytes.NewReader(body))
	if err != nil {
		log.Fatalf("Failed creating auth request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: HTTPTimeoutDuration}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		log.Fatalf("Login failed for %s. Ensure API is running (make dev-api). Status: %v, Err: %v", email, respStatusCode(resp), err)
	}
	defer resp.Body.Close()

	var authRes AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authRes); err != nil {
		log.Fatalf("Failed decoding auth response: %v", err)
	}
	return authRes.AccessToken
}

func provisionAndConnectDevices(cfg *Config, adminToken string) []*DeviceSession {
	sessions := make([]*DeviceSession, 0, cfg.DeviceCount)
	fmt.Printf("\n[Provisioning] Registering & provisioning %d devices...\n", cfg.DeviceCount)

	for i := 1; i <= cfg.DeviceCount; i++ {
		deviceID := fmt.Sprintf("%s-%03d", cfg.DevicePrefix, i)
		deviceName := fmt.Sprintf("Environmental Sensor #%d", i)

		token := registerAndProvisionDevice(cfg.APIBaseURL, adminToken, deviceID, deviceName)
		mqttClient := connectDeviceMQTT(cfg.BrokerURI, deviceID, token)

		session := &DeviceSession{
			DeviceID: deviceID,
			Token:    token,
			Client:   mqttClient,
			BaseTemp: BaseTemperature + float64(i%5)*1.2,
			BaseHum:  BaseHumidity + float64(i%4)*2.5,
		}
		sessions = append(sessions, session)
		fmt.Printf("  ✔ Device %s provisioned and connected to MQTT broker.\n", deviceID)
	}

	return sessions
}

func registerAndProvisionDevice(apiURL, adminToken, deviceID, name string) string {
	createPayload := map[string]interface{}{
		"id":          deviceID,
		"name":        name,
		"device_type": "environmental_sensor",
		"metadata": map[string]interface{}{
			"simulated": true,
			"sensors":   []string{"temperature", "humidity"},
		},
	}

	// Device may already exist from a previous run; safe to ignore creation conflict
	_ = requestAuthorizedJSON(apiURL+"/devices", "POST", createPayload, adminToken, true)
	credResp := requestAuthorizedJSON(apiURL+"/devices/"+deviceID+"/credentials", "POST", nil, adminToken, false)

	var provRes ProvisionResponse
	if err := json.Unmarshal(credResp, &provRes); err != nil || provRes.Token == "" {
		log.Fatalf("Failed to extract device token for %s: %v", deviceID, err)
	}

	return provRes.Token
}

func connectDeviceMQTT(brokerURI, deviceID, token string) paho.Client {
	opts := paho.NewClientOptions().
		AddBroker(brokerURI).
		SetClientID(deviceID).
		SetUsername(deviceID).
		SetPassword(token).
		SetConnectTimeout(MQTTConnectTimeout).
		SetAutoReconnect(true).
		SetKeepAlive(60 * time.Second)

	client := paho.NewClient(opts)
	connToken := client.Connect()
	if connToken.Wait() && connToken.Error() != nil {
		log.Fatalf("Failed to connect MQTT client %s: %v", deviceID, connToken.Error())
	}
	return client
}

func runSimulation(cfg *Config, sessions []*DeviceSession) {
	fmt.Printf("\n[Simulation] Publishing telemetry every %s (Ctrl+C to stop)...\n", cfg.Interval)

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	var durationTimer <-chan time.Time
	if cfg.Duration > 0 {
		durationTimer = time.After(cfg.Duration)
		fmt.Printf("[Simulation] Scheduled to run for %s\n", cfg.Duration)
	}

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	// Initial publish
	publishTelemetryFromAllDevices(sessions)

	for {
		select {
		case <-ticker.C:
			publishTelemetryFromAllDevices(sessions)
		case <-durationTimer:
			fmt.Printf("\n[Simulation] Configured duration (%s) completed.\n", cfg.Duration)
			return
		case <-stopChan:
			fmt.Println("\n[Simulation] Termination signal received. Stopping simulator...")
			return
		}
	}
}

func publishTelemetryFromAllDevices(sessions []*DeviceSession) {
	var wg sync.WaitGroup
	for _, s := range sessions {
		wg.Add(1)
		go func(sess *DeviceSession) {
			defer wg.Done()
			publishDeviceTelemetry(sess)
		}(s)
	}
	wg.Wait()
}

func publishDeviceTelemetry(session *DeviceSession) {
	temp := roundTwoDecimals(session.BaseTemp + (rand.Float64()*TempVarianceRange - TempVarianceRange/2))
	hum := roundTwoDecimals(session.BaseHum + (rand.Float64()*HumidityVariance - HumidityVariance/2))

	payload := TelemetryPayload{
		DeviceID:  session.DeviceID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Metrics: map[string]float64{
			"temperature": temp,
			"humidity":    hum,
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[Error] Failed formatting JSON for %s: %v", session.DeviceID, err)
		return
	}

	topic := fmt.Sprintf(TopicTemplate, session.DeviceID)
	token := session.Client.Publish(topic, 1, false, data)
	token.Wait()

	fmt.Printf("  [%s] Published to %s -> temp: %.2f°C | hum: %.2f%%\n",
		time.Now().Format("15:04:05"), topic, temp, hum)
}

func disconnectAllSessions(sessions []*DeviceSession) {
	fmt.Println("[Shutdown] Disconnecting all simulated MQTT devices...")
	for _, s := range sessions {
		if s.Client != nil && s.Client.IsConnected() {
			s.Client.Disconnect(MQTTDisconnectGrace)
		}
	}
	fmt.Println("✔ All simulated devices disconnected cleanly.")
}

func requestAuthorizedJSON(endpoint, method string, payload interface{}, token string, ignoreError bool) []byte {
	var bodyReader io.Reader
	if payload != nil {
		data, _ := json.Marshal(payload)
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(context.Background(), method, endpoint, bodyReader)
	if err != nil {
		log.Fatalf("Failed creating request for %s: %v", endpoint, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: HTTPTimeoutDuration}
	resp, err := client.Do(req)
	if err != nil {
		if ignoreError {
			return nil
		}
		log.Fatalf("Request %s %s failed: %v", method, endpoint, err)
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= http.StatusBadRequest && !ignoreError {
		log.Fatalf("Endpoint %s returned error code %d: %s", endpoint, resp.StatusCode, string(respBytes))
	}
	return respBytes
}

func roundTwoDecimals(val float64) float64 {
	return math.Round(val*100) / 100
}

func getEnvOrDefault(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func respStatusCode(resp *http.Response) string {
	if resp == nil {
		return "nil"
	}
	return resp.Status
}
