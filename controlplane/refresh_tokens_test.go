// SPDX-License-Identifier: AGPL-3.0-or-later

package controlplane

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRefreshTokenIssueConsume(t *testing.T) {
	store := NewRefreshTokenStore()

	token, _, err := store.Issue("user-1", 5*time.Minute)
	if err != nil {
		t.Fatalf("issue failed: %v", err)
	}

	subject, err := store.Consume(token)
	if err != nil {
		t.Fatalf("consume failed: %v", err)
	}
	if subject != "user-1" {
		t.Fatalf("unexpected subject: %s", subject)
	}

	if _, err := store.Consume(token); err == nil {
		t.Fatal("expected second consume to fail")
	}
}

func TestRefreshTokenPeekDoesNotConsume(t *testing.T) {
	store := NewRefreshTokenStore()

	token, _, err := store.Issue("user-peek", 5*time.Minute)
	if err != nil {
		t.Fatalf("issue failed: %v", err)
	}

	subject, err := store.Peek(token)
	if err != nil {
		t.Fatalf("peek failed: %v", err)
	}
	if subject != "user-peek" {
		t.Fatalf("unexpected subject: %s", subject)
	}

	subject, err = store.Consume(token)
	if err != nil {
		t.Fatalf("consume after peek failed: %v", err)
	}
	if subject != "user-peek" {
		t.Fatalf("unexpected consumed subject: %s", subject)
	}
}

func TestRefreshTokenConsumeIsAtomicUnderConcurrency(t *testing.T) {
	store := NewRefreshTokenStore()
	token, _, err := store.Issue("user-atomic", 5*time.Minute)
	if err != nil {
		t.Fatalf("issue failed: %v", err)
	}

	const workers = 64
	start := make(chan struct{})
	var wg sync.WaitGroup
	var successCount atomic.Int32
	var failureCount atomic.Int32

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if _, err := store.Consume(token); err == nil {
				successCount.Add(1)
				return
			}
			failureCount.Add(1)
		}()
	}

	close(start)
	wg.Wait()

	if got := int(successCount.Load()); got != 1 {
		t.Fatalf("expected exactly 1 successful consume, got %d", got)
	}
	if got := int(failureCount.Load()); got != workers-1 {
		t.Fatalf("expected %d failed consumes, got %d", workers-1, got)
	}
}
