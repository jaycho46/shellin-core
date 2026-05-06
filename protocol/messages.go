// SPDX-License-Identifier: AGPL-3.0-or-later

package protocol

// Transport identifies the connection mode used for an agent session.
type Transport = string

const (
	// TransportWebRTCTURN is the WebRTC data-channel transport with TURN support.
	TransportWebRTCTURN Transport = "webrtc_turn"
)

// ICEPolicy mirrors the WebRTC ICE transport policy returned by the control plane.
type ICEPolicy = string

const (
	ICEPolicyAll   ICEPolicy = "all"
	ICEPolicyRelay ICEPolicy = "relay"
)

// SDPType identifies an SDP offer or answer payload.
type SDPType = string

const (
	SDPTypeOffer  SDPType = "offer"
	SDPTypeAnswer SDPType = "answer"
)

// SignalMessageType identifies a signaling message forwarded between peers.
type SignalMessageType = string

const (
	SignalMessageTypeOffer          SignalMessageType = "offer"
	SignalMessageTypeAnswer         SignalMessageType = "answer"
	SignalMessageTypeCandidate      SignalMessageType = "candidate"
	SignalMessageTypeViewerReady    SignalMessageType = "viewer_ready"
	SignalMessageTypeViewerReplaced SignalMessageType = "viewer_replaced"
	SignalMessageTypeDetach         SignalMessageType = "detach"
	SignalMessageTypeP2PConnected   SignalMessageType = "p2p_connected"
)

// ExchangeAuthRequest exchanges a user key for control-plane auth tokens.
type ExchangeAuthRequest struct {
	UserKey string `json:"user_key"`
}

// ExchangeAuthResponse returns access and refresh tokens for the CLI.
type ExchangeAuthResponse struct {
	AccessToken           string `json:"access_token"`
	AccessTokenExpiresAt  string `json:"access_token_expires_at,omitempty"`
	RefreshToken          string `json:"refresh_token"`
	RefreshTokenExpiresAt string `json:"refresh_token_expires_at,omitempty"`
}

// RefreshAuthRequest exchanges a refresh token for a fresh token pair.
type RefreshAuthRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshAuthResponse returns a renewed access token and refresh token.
type RefreshAuthResponse struct {
	AccessToken           string `json:"access_token"`
	AccessTokenExpiresAt  string `json:"access_token_expires_at,omitempty"`
	RefreshToken          string `json:"refresh_token"`
	RefreshTokenExpiresAt string `json:"refresh_token_expires_at,omitempty"`
}

// IssueLoginKeyRequest asks the control plane to issue a one-time device login key.
type IssueLoginKeyRequest struct {
	TTLSeconds int `json:"ttl_seconds,omitempty"`
}

// IssueLoginKeyResponse returns a human-entered login key and expiry.
type IssueLoginKeyResponse struct {
	LoginKey  string `json:"login_key"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

// DeviceLoginRequest consumes a one-time login key from another device.
type DeviceLoginRequest struct {
	LoginKey string `json:"login_key"`
}

// AgentRegisterRequest announces a CLI session to the control plane.
type AgentRegisterRequest struct {
	SessionID  string    `json:"session_id"`
	AgentLabel string    `json:"agent_label,omitempty"`
	Transport  Transport `json:"transport"`
}

// AgentHeartbeatRequest refreshes the last-seen timestamp for an agent session.
type AgentHeartbeatRequest struct {
	SessionID  string `json:"session_id"`
	AgentLabel string `json:"agent_label,omitempty"`
}

// AgentUnregisterRequest removes a CLI session from the registry.
type AgentUnregisterRequest struct {
	SessionID string `json:"session_id"`
}

// AgentTerminateAllResponse reports how many sessions were terminated.
type AgentTerminateAllResponse struct {
	TerminatedSessions int `json:"terminated_sessions"`
}

// AgentConnectRequest asks the control plane for agent-side signaling bootstrap data.
type AgentConnectRequest struct {
	SessionID  string `json:"session_id"`
	AgentLabel string `json:"agent_label,omitempty"`
}

// AgentConnectResponse returns the agent-side signaling URL, token, and ICE config.
type AgentConnectResponse struct {
	SessionID   string      `json:"session_id"`
	SignalURL   string      `json:"signal_url"`
	SignalToken string      `json:"signal_token"`
	ICEServers  []ICEServer `json:"ice_servers,omitempty"`
	ICEPolicy   ICEPolicy   `json:"ice_policy,omitempty"`
	ExpiresAt   string      `json:"expires_at,omitempty"`
}

// AgentAttachRequest asks to attach a viewer to an existing agent session.
type AgentAttachRequest struct {
	SessionID string `json:"session_id"`
}

// AgentAttachResponse returns viewer-side signaling bootstrap data.
type AgentAttachResponse struct {
	SessionID   string      `json:"session_id"`
	SignalURL   string      `json:"signal_url"`
	SignalToken string      `json:"signal_token"`
	ICEServers  []ICEServer `json:"ice_servers,omitempty"`
	ICEPolicy   ICEPolicy   `json:"ice_policy,omitempty"`
	ExpiresAt   string      `json:"expires_at,omitempty"`
}

// AgentStatus is the public registry view of one agent session.
type AgentStatus struct {
	SessionID  string    `json:"session_id"`
	AgentLabel string    `json:"agent_label,omitempty"`
	Transport  Transport `json:"transport"`
	CreatedAt  string    `json:"created_at,omitempty"`
	LastSeenAt string    `json:"last_seen_at,omitempty"`
}

// ListAgentsResponse returns the active sessions visible to the caller.
type ListAgentsResponse struct {
	Agents []AgentStatus `json:"agents"`
}

// SessionStatusResponse describes the caller's entitlement and current usage.
type SessionStatusResponse struct {
	Subject               string `json:"subject,omitempty"`
	ActiveSessions        int    `json:"active_sessions"`
	MaxSessions           int    `json:"max_sessions"`
	ProductID             string `json:"product_id,omitempty"`
	OriginalTransactionID string `json:"original_transaction_id,omitempty"`
	SubscriptionStatus    string `json:"subscription_status,omitempty"`
	ExpiresAt             string `json:"expires_at,omitempty"`
}

// ICEServer is a sanitized WebRTC ICE server entry sent to clients.
type ICEServer struct {
	URLs       []string `json:"urls"`
	Username   string   `json:"username,omitempty"`
	Credential string   `json:"credential,omitempty"`
}

// SDPPayload carries a WebRTC session description.
type SDPPayload struct {
	Type SDPType `json:"type"`
	SDP  string  `json:"sdp"`
}

// ICECandidate carries one WebRTC ICE candidate from browser or CLI APIs.
type ICECandidate struct {
	Candidate        string  `json:"candidate"`
	SDPMid           *string `json:"sdpMid,omitempty"`
	SDPMLineIndex    *uint16 `json:"sdpMLineIndex,omitempty"`
	UsernameFragment *string `json:"usernameFragment,omitempty"`
}

// SignalMessage is the only message shape accepted by the signaling WebSocket.
type SignalMessage struct {
	Type      SignalMessageType `json:"type"`
	SDP       *SDPPayload       `json:"sdp,omitempty"`
	Candidate *ICECandidate     `json:"candidate,omitempty"`
}
