package utils

import (
	"crypto/rand"
	"encoding/hex"
)

func GenerateUUIDWithoutHyphen() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
