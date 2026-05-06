// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"github.com/jaycho46/shellin-core/protocol"
	"github.com/jaycho46/shellin-core/turn"
)

func issueICE(cfg config, subject string) ([]protocol.ICEServer, error) {
	turnCfg := turn.CloudflareConfig{
		APIBaseURL: cfg.CloudflareTURNBaseURL,
		KeyID:      cfg.CloudflareTURNKeyID,
		APIToken:   cfg.CloudflareTURNToken,
		TTL:        cfg.CloudflareTURNTTL,
	}
	if !turn.CloudflareEnabled(turnCfg) {
		return []protocol.ICEServer{{URLs: []string{turn.CloudflareSTUNURL}}}, nil
	}
	return turn.IssueCloudflareCredentials(turnCfg, subject)
}
