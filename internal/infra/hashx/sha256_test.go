package hashx

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestSumBytes_ReturnsSHA256Hex(t *testing.T) {
	input := []byte("skillc")
	wantSum := sha256.Sum256(input)
	want := hex.EncodeToString(wantSum[:])

	got := SumBytes(input)
	assert.Eq(t, want, got)
}

func TestSumString_MatchesByteHash(t *testing.T) {
	got := SumString("config")
	assert.Eq(t, SumBytes([]byte("config")), got)
}
