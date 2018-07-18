package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// CheckHMAC reports whether msgHMAC is a valid HMAC tag for msg.
func CheckHMAC(body, timestamp, msgHMAC, key string) bool {
	msg := "v0:" + timestamp + ":" + body
	hash := hmac.New(sha256.New, []byte(key))
	hash.Write([]byte(msg))

	expectedKey := hash.Sum(nil)
	actualKey, _ := hex.DecodeString(msgHMAC)
	return hmac.Equal(expectedKey, actualKey)
}

// NullComparator is a dummy comparator that allows us to define an
// empty Verify method
type NullComparator struct{}

// Verify always returns true, overriding the default token verification
// method. This is acceptable as we implement a separate check to confirm
// the validity of the request signature.
func (c NullComparator) Verify(string) bool {
	return true
}
