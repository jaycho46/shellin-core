// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"strings"

	"github.com/pion/webrtc/v4"

	"github.com/jaycho46/shellin-core/protocol"
)

func toWebRTCICEServers(in []protocol.ICEServer) []webrtc.ICEServer {
	out := make([]webrtc.ICEServer, 0, len(in))
	for _, server := range in {
		urls := make([]string, 0, len(server.URLs))
		for _, rawURL := range server.URLs {
			rawURL = strings.TrimSpace(rawURL)
			if rawURL == "" {
				continue
			}
			urls = append(urls, rawURL)
		}
		if len(urls) == 0 {
			continue
		}
		out = append(out, webrtc.ICEServer{
			URLs:       urls,
			Username:   strings.TrimSpace(server.Username),
			Credential: strings.TrimSpace(server.Credential),
		})
	}
	if len(out) == 0 {
		return []webrtc.ICEServer{{URLs: []string{"stun:stun.cloudflare.com:3478"}}}
	}
	return out
}

func iceServersAreTURNOnly(in []protocol.ICEServer) bool {
	if len(in) == 0 {
		return false
	}
	for _, server := range in {
		if len(server.URLs) == 0 {
			return false
		}
		for _, rawURL := range server.URLs {
			rawURL = strings.ToLower(strings.TrimSpace(rawURL))
			if !strings.HasPrefix(rawURL, "turn:") && !strings.HasPrefix(rawURL, "turns:") {
				return false
			}
		}
	}
	return true
}
