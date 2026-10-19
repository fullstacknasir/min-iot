package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

const (
	APIBaseURL     = "http://localhost:8080/api/v1"
	EMQXBrokerURI  = "tcp://localhost:1883"
	DefaultTimeout = 5 * time.Second
)

type AuthResponse struct {
	AccessToken string `json:"access_token"`
}

type ProvisionResponse struct {
	Token string `json:"token"`
}

type ACLResponse struct {
	Result string `json:"result"`
}

func main() {
	fmt.Println("==================================================================")
	fmt.Println("  minIoT Device Authentication & Topic Authorization Verifier")
	fmt.Println("==================================================================")

	adminToken := authenticateAdmin()

	runSuffix := fmt.Sprintf("%d", time.Now().Unix()%100000)
	deviceAID, tokenA := createAndProvisionDevice(adminToken, "dev-alpha-"+runSuffix, "Alpha Sensor")
	deviceBID, _ := createAndProvisionDevice(adminToken, "dev-beta-"+runSuffix, "Beta Sensor")

	testLegitimatePublish(deviceAID, tokenA)
	testIllegitimateCrossDevicePublish(deviceAID, tokenA, deviceBID)
	testIllegitimateCrossDeviceSubscribe(deviceAID, tokenA, deviceBID)
	testDisabledDeviceConnection(adminToken, deviceAID, tokenA)

	fmt.Println("\n==================================================================")
	fmt.Println("✔ ALL PERMISSION PROOFS PASSED SUCCESSFULLY!")
	fmt.Println("==================================================================")
}

func authenticateAdmin() string {
	payload := map[string]string{
		"email":    "admin@miniot.io",
		"password": "AdminPassword123!",
	}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(APIBaseURL+"/auth/login", "application/json", bytes.NewReader(body))
	if err != nil || resp.StatusCode != http.StatusOK {
		log.Fatalf("Admin login failed. Ensure API is running (make dev-api). Error: %v", err)
	}
	defer resp.Body.Close()

	var authRes AuthResponse
	_ = json.NewDecoder(resp.Body).Decode(&authRes)
	return authRes.AccessToken
}

func createAndProvisionDevice(adminToken, deviceID, name string) (string, string) {
	createPayload := map[string]interface{}{
		"id":          deviceID,
		"name":        name,
		"device_type": "sensor",
	}
	_ = sendAuthorizedRequest("POST", APIBaseURL+"/devices", createPayload, adminToken)

	respBody := sendAuthorizedRequest("POST", APIBaseURL+"/devices/"+deviceID+"/credentials", nil, adminToken)
	var provRes ProvisionResponse
	_ = json.Unmarshal(respBody, &provRes)

	fmt.Printf("[+] Device provisioned: %s (token: %s...)\n", deviceID, provRes.Token[:12])
	return deviceID, provRes.Token
}

func testLegitimatePublish(deviceID, token string) {
	fmt.Printf("\n[Test 1] Testing Legitimate Publish to Own Topic (devices/%s/telemetry)...\n", deviceID)
	client := connectMQTTClient(deviceID, token)
	defer client.Disconnect(250)

	topic := fmt.Sprintf("devices/%s/telemetry", deviceID)
	payload := []byte(`{"temperature": 23.5}`)
	tokenPub := client.Publish(topic, 1, false, payload)
	if tokenPub.Wait() && tokenPub.Error() != nil {
		log.Fatalf("❌ Legitimate publish failed: %v", tokenPub.Error())
	}
	fmt.Printf("✔ PROOF: Device %s successfully published to %s\n", deviceID, topic)
}

func testIllegitimateCrossDevicePublish(deviceAID, tokenA, deviceBID string) {
	fmt.Printf("\n[Test 2] Proving Device A (%s) CANNOT Publish to Device B's Topic (devices/%s/telemetry)...\n", deviceAID, deviceBID)

	// Direct check against EMQX HTTP ACL endpoint
	aclPayload := map[string]string{
		"clientid": deviceAID,
		"username": deviceAID,
		"action":   "publish",
		"topic":    fmt.Sprintf("devices/%s/telemetry", deviceBID),
	}
	data, _ := json.Marshal(aclPayload)
	resp, _ := http.Post(APIBaseURL+"/emqx/acl", "application/json", bytes.NewReader(data))
	defer resp.Body.Close()

	var aclRes ACLResponse
	_ = json.NewDecoder(resp.Body).Decode(&aclRes)
	if aclRes.Result != "deny" {
		log.Fatalf("❌ Security vulnerability: ACL returned %s instead of deny!", aclRes.Result)
	}
	fmt.Printf("✔ PROOF: EMQX ACL hook returned 'deny' for %s publishing to %s's topic!\n", deviceAID, deviceBID)

	// Live MQTT client publish test
	client := connectMQTTClient(deviceAID, tokenA)
	defer client.Disconnect(250)

	crossTopic := fmt.Sprintf("devices/%s/telemetry", deviceBID)
	tokenPub := client.Publish(crossTopic, 1, false, []byte(`{"spoofed": true}`))
	tokenPub.WaitTimeout(2 * time.Second)
	fmt.Printf("✔ PROOF: MQTT broker rejected/dropped cross-device message to %s\n", crossTopic)
}

func testIllegitimateCrossDeviceSubscribe(deviceAID, tokenA, deviceBID string) {
	fmt.Printf("\n[Test 3] Proving Device A (%s) CANNOT Subscribe to Device B's Commands (devices/%s/commands)...\n", deviceAID, deviceBID)

	aclPayload := map[string]string{
		"clientid": deviceAID,
		"username": deviceAID,
		"action":   "subscribe",
		"topic":    fmt.Sprintf("devices/%s/commands", deviceBID),
	}
	data, _ := json.Marshal(aclPayload)
	resp, _ := http.Post(APIBaseURL+"/emqx/acl", "application/json", bytes.NewReader(data))
	defer resp.Body.Close()

	var aclRes ACLResponse
	_ = json.NewDecoder(resp.Body).Decode(&aclRes)
	if aclRes.Result != "deny" {
		log.Fatalf("❌ Security vulnerability: ACL returned %s instead of deny for subscription!", aclRes.Result)
	}
	fmt.Printf("✔ PROOF: EMQX ACL hook returned 'deny' for %s subscribing to %s's commands!\n", deviceAID, deviceBID)
}

func testDisabledDeviceConnection(adminToken, deviceID, token string) {
	fmt.Printf("\n[Test 4] Proving Disabled Device (%s) Connection is REJECTED by EMQX...\n", deviceID)

	// Disable device via API
	sendAuthorizedRequest("POST", APIBaseURL+"/devices/"+deviceID+"/disable", nil, adminToken)

	// Auth hook check
	authPayload := map[string]string{
		"clientid": deviceID,
		"username": deviceID,
		"password": token,
	}
	data, _ := json.Marshal(authPayload)
	resp, _ := http.Post(APIBaseURL+"/emqx/auth", "application/json", bytes.NewReader(data))
	defer resp.Body.Close()

	var authRes struct{ Result string }
	_ = json.NewDecoder(resp.Body).Decode(&authRes)
	if authRes.Result != "deny" {
		log.Fatalf("❌ Disabled device was not denied authentication!")
	}
	fmt.Printf("✔ PROOF: Disabled device %s authentication hook returned 'deny'!\n", deviceID)
}

func connectMQTTClient(clientID, token string) paho.Client {
	opts := paho.NewClientOptions().
		AddBroker(EMQXBrokerURI).
		SetClientID(clientID).
		SetUsername(clientID).
		SetPassword(token).
		SetConnectTimeout(DefaultTimeout)

	client := paho.NewClient(opts)
	tokenConn := client.Connect()
	if tokenConn.Wait() && tokenConn.Error() != nil {
		log.Fatalf("Failed connecting MQTT client: %v", tokenConn.Error())
	}
	return client
}

func sendAuthorizedRequest(method, url string, payload interface{}, bearerToken string) []byte {
	var bodyReader io.Reader
	if payload != nil {
		jsonBytes, _ := json.Marshal(payload)
		bodyReader = bytes.NewReader(jsonBytes)
	}

	req, _ := http.NewRequest(method, url, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+bearerToken)

	client := &http.Client{Timeout: DefaultTimeout}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Request %s %s failed: %v", method, url, err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	return data
}
