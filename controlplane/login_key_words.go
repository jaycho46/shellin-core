// SPDX-License-Identifier: AGPL-3.0-or-later

package controlplane

import (
	"crypto/rand"
	_ "embed"
	"fmt"
	"math/big"
	"strings"
)

const loginKeyWordCount = 4

// login_key_words.txt is the English BIP-39 wordlist. See docs/wordlists.md for
// source and license details.
//
//go:embed login_key_words.txt
var loginKeyWordsRaw string

var loginKeyWords = mustLoadLoginKeyWords(loginKeyWordsRaw)

func mustLoadLoginKeyWords(raw string) []string {
	words := strings.Fields(raw)
	if len(words) != 2048 {
		panic(fmt.Sprintf("login key word list must contain 2048 words, got %d", len(words)))
	}
	return words
}

func randomWordLoginKey() (string, error) {
	parts := make([]string, loginKeyWordCount)
	max := big.NewInt(int64(len(loginKeyWords)))
	for i := range parts {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		parts[i] = loginKeyWords[int(n.Int64())]
	}
	return normalizeLoginKey(strings.Join(parts, "-")), nil
}

func normalizeLoginKey(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}
