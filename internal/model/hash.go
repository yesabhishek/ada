package model

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func HashText(parts ...string) string {
	hash := sha256.New()
	for _, part := range parts {
		hash.Write([]byte(part))
		hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func NormalizeWhitespace(text string) string {
	return strings.Join(strings.Fields(text), " ")
}
