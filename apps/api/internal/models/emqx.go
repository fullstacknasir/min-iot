package models

const (
	EMQXResultAllow = "allow"
	EMQXResultDeny  = "deny"

	EMQXActionPublish   = "publish"
	EMQXActionSubscribe = "subscribe"
)

type EMQXAuthRequest struct {
	ClientID string `json:"clientid"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type EMQXAuthResponse struct {
	Result      string `json:"result"`
	IsSuperuser bool   `json:"is_superuser"`
}

type EMQXACLRequest struct {
	ClientID string `json:"clientid"`
	Username string `json:"username"`
	Action   string `json:"action"`
	Topic    string `json:"topic"`
}

type EMQXACLResponse struct {
	Result string `json:"result"`
}

type EMQXEventPayload struct {
	Event          string      `json:"event"`
	ClientID       string      `json:"clientid"`
	Username       string      `json:"username"`
	ConnectedAt    *int64      `json:"connected_at"`
	DisconnectedAt *int64      `json:"disconnected_at"`
	Reason         string      `json:"reason"`
	Timestamp      interface{} `json:"timestamp"`
}
