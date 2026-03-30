package hashx

import (
	"crypto/sha256"
	"encoding/hex"
)

func SumBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func SumString(s string) string {
	return SumBytes([]byte(s))
}
