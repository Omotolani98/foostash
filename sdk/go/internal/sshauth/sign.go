// Package sshauth is a copy of the CLI's request-signing primitives adapted
// for external SDK consumers. The canonical source lives at
// github.com/Omotolani98/foostash/internal/sshauth — the wire format must
// match the server exactly.
package sshauth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"golang.org/x/crypto/ssh"
)

const (
	HeaderTimestamp   = "X-Foostash-Timestamp"
	HeaderFingerprint = "X-Foostash-Key-Fingerprint"
	HeaderSignature   = "X-Foostash-Signature"
)

// CanonicalPayload builds the byte slice that is signed. The format is:
//
//	METHOD\nPATH\nTIMESTAMP\nHEX(SHA256(body))
func CanonicalPayload(method, path, timestamp string, body []byte) []byte {
	sum := sha256.Sum256(body)
	out := fmt.Sprintf("%s\n%s\n%s\n%s", method, path, timestamp, hex.EncodeToString(sum[:]))
	return []byte(out)
}

// Sign produces a base64-encoded signature blob over the canonical payload
// and returns the timestamp that was used.
func Sign(signer ssh.Signer, method, path string, body []byte) (signature, timestamp string, err error) {
	timestamp = time.Now().UTC().Format(time.RFC3339)
	payload := CanonicalPayload(method, path, timestamp, body)
	sig, err := signer.Sign(rand.Reader, payload)
	if err != nil {
		return "", "", fmt.Errorf("sign payload: %w", err)
	}
	blob := ssh.Marshal(sig)
	return base64.StdEncoding.EncodeToString(blob), timestamp, nil
}
