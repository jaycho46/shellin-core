// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"

	"github.com/jaycho46/shellin-core/protocol"
)

type liveWebRTCSession struct {
	pc            *webrtc.PeerConnection
	terminalDC    *webrtc.DataChannel
	controlDC     *webrtc.DataChannel
	signalOut     chan protocol.SignalMessage
	signalingLost chan struct{}
	reconnectNow  chan struct{}
	status        *statusPresenter
	onSignal      func(protocol.SignalMessage)
	onSignalLost  func()

	signalMu sync.Mutex
	signal   *signalPump

	closeOnce sync.Once
}

type signalPump struct {
	conn *websocket.Conn
	stop chan struct{}
	done chan struct{}

	suppressLost atomic.Bool
}

func (s *liveWebRTCSession) Close() {
	if s == nil {
		return
	}
	s.closeOnce.Do(func() {
		s.stopSignalPump()
		if s.pc != nil {
			_ = s.pc.Close()
		}
	})
}

func (s *liveWebRTCSession) stopSignalPump() {
	s.signalMu.Lock()
	pump := s.signal
	s.signal = nil
	s.signalMu.Unlock()
	if pump != nil {
		pump.StopAndWait()
	}
}

func (p *signalPump) StopAndWait() {
	if p == nil {
		return
	}
	p.suppressLost.Store(true)
	closeStopChannel(p.stop)
	if p.conn != nil {
		_ = p.conn.Close()
	}
	<-p.done
}

func (s *liveWebRTCSession) ReconnectSignaling(connect protocol.AgentConnectResponse) error {
	if s == nil {
		return errors.New("live session is nil")
	}

	wsConn, err := dialSignalWS(connect.SignalURL, connect.SignalToken)
	if err != nil {
		return err
	}

	s.stopSignalPump()

	pump := &signalPump{
		conn: wsConn,
		stop: make(chan struct{}),
		done: make(chan struct{}),
	}

	s.signalMu.Lock()
	s.signal = pump
	s.signalMu.Unlock()

	if s.status != nil {
		s.status.SetSignalState("normal")
	}

	var wg sync.WaitGroup
	var lostOnce sync.Once
	reportLost := func() {
		if pump.suppressLost.Load() {
			return
		}
		lostOnce.Do(func() {
			closeStopChannel(pump.stop)
			if pump.conn != nil {
				_ = pump.conn.Close()
			}
			if s.status != nil {
				s.status.SetSignalState("closed")
			}
			if s.onSignalLost != nil {
				s.onSignalLost()
			}
		})
	}

	wg.Add(2)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-pump.stop:
				return
			case msg := <-s.signalOut:
				if err := pump.conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
					reportLost()
					return
				}
				if err := pump.conn.WriteJSON(msg); err != nil {
					reportLost()
					return
				}
			}
		}
	}()

	go func() {
		defer wg.Done()
		for {
			_, payload, err := pump.conn.ReadMessage()
			if err != nil {
				reportLost()
				return
			}
			var msg protocol.SignalMessage
			if err := json.Unmarshal(payload, &msg); err != nil {
				continue
			}
			if s.onSignal != nil {
				s.onSignal(msg)
			}
		}
	}()

	go func() {
		wg.Wait()
		close(pump.done)
	}()

	return nil
}

func dialSignalWS(signalURL, token string) (*websocket.Conn, error) {
	header := http.Header{}
	header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		Subprotocols:     []string{"mt-signaling"},
	}
	conn, resp, err := dialer.Dial(strings.TrimSpace(signalURL), header)
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("dial signaling websocket failed: %w (status=%s)", err, resp.Status)
		}
		return nil, fmt.Errorf("dial signaling websocket failed: %w", err)
	}
	if conn.Subprotocol() != "mt-signaling" {
		_ = conn.Close()
		return nil, errors.New("signaling websocket subprotocol negotiation failed")
	}
	return conn, nil
}
