package sshauth

import (
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/ssh"
)

var (
	ErrExpiredRequest = errors.New("request timestamp outside allowed skew")
	ErrBadSignature   = errors.New("signature verification failed")
	ErrBadTimestamp   = errors.New("invalid timestamp")
)

// Verify checks that sigB64 is a valid signature by pubKey over the canonical
// payload built from (method, path, timestamp, body). It also enforces that
// timestamp is within MaxSkew of now.
func Verify(pubKey ssh.PublicKey, method, path, timestamp, sigB64 string, body []byte) error {
	ts, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrBadTimestamp, err)
	}
	if diff := time.Since(ts); diff > MaxSkew || diff < -MaxSkew {
		return ErrExpiredRequest
	}
	sig, err := DecodeSignature(sigB64)
	if err != nil {
		return err
	}
	payload := CanonicalPayload(method, path, timestamp, body)
	if err := pubKey.Verify(payload, sig); err != nil {
		return fmt.Errorf("%w: %v", ErrBadSignature, err)
	}
	return nil
}
