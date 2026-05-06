// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jaycho46/shellin-core/controlplane"
)

func main() {
	cfg, err := parseConfig(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	store := controlplane.NewUserKeyStore()
	if err := seedUserKeys(cfg, store); err != nil {
		log.Fatalf("failed to seed user keys: %v", err)
	}
	if store.Count() == 0 {
		log.Fatal("no user keys configured")
	}

	deps, err := newServerDeps(cfg, store)
	if err != nil {
		log.Fatal(err)
	}
	startCleanupLoop(cfg, deps)

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           NewServer(cfg, deps),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("shellin control plane listening on %s (keys=%d)", cfg.Addr, store.Count())
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
