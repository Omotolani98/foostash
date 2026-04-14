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

// Header names used for signed requests.
const (
	HeaderTimestamp   = "X-Foostash-Timestamp"
	HeaderFingerprint = "X-Foostash-Key-Fingerprint"
	HeaderSignature   = "X-Foostash-Signature"
)

// MaxSkew is the maximum permitted clock skew between client and server.
const MaxSkew = 5 * time.Minute

// CanonicalPayload builds the byte slice that is signed. The format is:
//
//	METHOD\nPATH\nTIMESTAMP\nHEX(SHA256(body))
//
// The body hash line is always present; an empty body hashes to the sha256
// of the empty string so that client and server do not need a special case.
func CanonicalPayload(method, path, timestamp string, body []byte) []byte {
	sum := sha256.Sum256(body)
	out := fmt.Sprintf("%s\n%s\n%s\n%s", method, path, timestamp, hex.EncodeToString(sum[:]))
	return []byte(out)
}

// Sign produces a base64-encoded signature blob over the canonical payload.
// Returns the timestamp that was used so the caller can set the matching
// request header.
func Sign(signer ssh.Signer, method, path string, body []byte) (signature, timestamp string, err error) {
	timestamp = time.Now().UTC().Format(time.RFC3339)
	payload := CanonicalPayload(method, path, timestamp, body)
	sig, err := signer.Sign(rand.Reader, payload)
	if err != nil {
		return "", "", fmt.Errorf("sign payload: %w", err)
	}
	// Encode the full ssh.Signature (including format) so the verifier can
	// reconstruct it. ssh.Marshal handles the wire format.
	blob := ssh.Marshal(sig)
	return base64.StdEncoding.EncodeToString(blob), timestamp, nil
}

// DecodeSignature reverses the base64 + ssh.Marshal dance performed by Sign.
func DecodeSignature(s string) (*ssh.Signature, error) {
	raw, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("decode signature: %w", err)
	}
	var sig ssh.Signature
	if err := ssh.Unmarshal(raw, &sig); err != nil {
		return nil, fmt.Errorf("unmarshal signature: %w", err)
	}
	return &sig, nil
}
