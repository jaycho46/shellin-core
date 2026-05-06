// SPDX-License-Identifier: AGPL-3.0-or-later

package signaling

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/jaycho46/shellin-core/controlplane"
	"github.com/jaycho46/shellin-core/grantauth"
	"github.com/jaycho46/shellin-core/internal/httporigin"
	"github.com/jaycho46/shellin-core/protocol"
)

const MaxSignalMessageBytes = 64 * 1024

type Options struct {
	Validator   *grantauth.Validator
	ReplayStore grantauth.ReplayStore
	// Deprecated: use ReplayStore.
	ReplayGuard grantauth.ReplayStore
	Registry    *controlplane.AgentRegistry
	StaleAfter  time.Duration
	Hub         *controlplane.SignalHub
	ExternalURL string
}

func Handler(opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		HandleWebSocket(w, r, opts)
	}
}

func HandleWebSocket(w http.ResponseWriter, r *http.Request, opts Options) {
	replayStore := opts.replayStore()
	if opts.Validator == nil || replayStore == nil || opts.Registry == nil || opts.Hub == nil {
		http.Error(w, "signaling unavailable", http.StatusServiceUnavailable)
		return
	}

	token := bearerToken(r)
	if token == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	claims, err := opts.Validator.Validate(token)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if err := replayStore.Consume(claims.TokenID, claims.ExpiresAt); err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if claims.Role != string(controlplane.SignalRoleAgent) && claims.Role != string(controlplane.SignalRoleViewer) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if claims.SessionID == "" || claims.Subject == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if claims.Role == string(controlplane.SignalRoleViewer) {
		if _, ok := opts.Registry.Lookup(claims.Subject, claims.SessionID, opts.StaleAfter); !ok {
			http.Error(w, "agent session not found", http.StatusNotFound)
			return
		}
	}

	upgrader := websocket.Upgrader{
		Subprotocols: []string{"mt-signaling"},
		CheckOrigin: func(r *http.Request) bool {
			return OriginAllowedForBaseURL(r, opts.ExternalURL)
		},
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	conn.SetReadLimit(MaxSignalMessageBytes)

	outbound := make(chan protocol.SignalMessage, 128)
	var closeOnce sync.Once
	closed := make(chan struct{})
	closeConn := func(code int, reason string) {
		closeOnce.Do(func() {
			close(closed)
			deadline := time.Now().Add(2 * time.Second)
			_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, reason), deadline)
			_ = conn.Close()
		})
	}
	peer := &controlplane.SignalPeer{
		Subject:   claims.Subject,
		SessionID: claims.SessionID,
		Role:      controlplane.SignalRole(claims.Role),
		Send: func(msg protocol.SignalMessage) bool {
			select {
			case <-closed:
				return false
			case outbound <- msg:
				return true
			default:
				return false
			}
		},
		Close: closeConn,
	}
	opts.Hub.Register(peer)

	writeDone := make(chan struct{})
	go func() {
		defer close(writeDone)
		for {
			select {
			case <-closed:
				return
			case msg := <-outbound:
				_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := conn.WriteJSON(msg); err != nil {
					return
				}
			}
		}
	}()
	defer func() {
		opts.Hub.Unregister(peer)
		closeConn(websocket.CloseNormalClosure, "closing")
		<-writeDone
	}()

	for {
		select {
		case <-closed:
			return
		default:
		}
		_, payload, err := conn.ReadMessage()
		if err != nil {
			break
		}
		var msg protocol.SignalMessage
		if err := json.Unmarshal(payload, &msg); err != nil {
			continue
		}
		_ = opts.Hub.Forward(peer, msg)
	}
}

func OriginAllowed(r *http.Request) bool {
	return OriginAllowedForBaseURL(r, "")
}

func (opts Options) replayStore() grantauth.ReplayStore {
	if opts.ReplayStore != nil {
		return opts.ReplayStore
	}
	return opts.ReplayGuard
}

func OriginAllowedForBaseURL(r *http.Request, externalBaseURL string) bool {
	return httporigin.OriginAllowedForBaseURL(r, externalBaseURL)
}

func bearerToken(r *http.Request) string {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if auth == "" {
		return ""
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(strings.ToLower(auth), strings.ToLower(prefix)) {
		return ""
	}
	return strings.TrimSpace(auth[len(prefix):])
}
