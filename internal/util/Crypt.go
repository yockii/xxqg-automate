package util

import (
	"crypto/hmac"
	"crypto/sha256"
)

func HmacSha256(data, secret []byte) []byte {
	h := hmac.New(sha256.New, secret)
	h.Write(data)
	return h.Sum(nil)
}
