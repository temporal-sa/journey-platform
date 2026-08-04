package experiments

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

// VersionedKey formats the HMAC salt with the experiment version number.
// Example: "exp-onboarding:v1"
func VersionedKey(salt string, version int) string {
	if version <= 0 {
		version = 1
	}
	return fmt.Sprintf("%s:v%d", salt, version)
}

// BucketUnit computes a deterministic bucket integer in range [0, 9999] using HMAC SHA-256.
// It matches Go, TypeScript, and SQL semantics:
// Go: binary.BigEndian.Uint64(hmacSHA256(key, unitID)[:8]) % 10000
// TypeScript: Number(crypto.createHmac('sha256', key).update(unitID).digest().readBigUInt64BE(0) % 10000n)
// SQL: (('0x' || substring(encode(hmac(unit_id, key, 'sha256'), 'hex') from 1 for 16))::bit(64)::numeric % 10000)
func BucketUnit(versionedKey, unitID string) uint16 {
	mac := hmac.New(sha256.New, []byte(versionedKey))
	mac.Write([]byte(unitID))
	digest := mac.Sum(nil)

	val := binary.BigEndian.Uint64(digest[:8])
	return uint16(val % 10000)
}

// SelectVariant selects a variant for the given unit ID based on experiment variant weights (basis points).
func SelectVariant(exp *Experiment, unitID string) (*Variant, uint16, error) {
	if err := ValidateExperiment(exp); err != nil {
		return nil, 0, fmt.Errorf("cannot bucket invalid experiment: %w", err)
	}

	vk := VersionedKey(exp.Salt, exp.Version)
	bucket := BucketUnit(vk, unitID)

	var cumulative int
	for i := range exp.Variants {
		cumulative += exp.Variants[i].WeightBasisPoints
		if int(bucket) < cumulative {
			return &exp.Variants[i], bucket, nil
		}
	}

	// Fallback to last variant in case of rounding (should not occur since sum is 10000)
	lastIndex := len(exp.Variants) - 1
	return &exp.Variants[lastIndex], bucket, nil
}
