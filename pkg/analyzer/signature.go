package analyzer

import (
	"crypto/sha256"
	"encoding/hex"
)

const signatureLength = 12

// Signature returns a short stable fingerprint for a failure, built from the
// testFile and the canonicalised exit message. Two failures with the same
// signature represent (with high probability) the same underlying defect.
func Signature(testFile, canonicalMessage string) string {
	h := sha256.Sum256([]byte(testFile + "|" + canonicalMessage))
	return hex.EncodeToString(h[:])[:signatureLength]
}
