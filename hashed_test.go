package bbloom

import (
	"crypto/sha256"
	"encoding/binary"
	"testing"
)

func sha256Digest(i int) []byte {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], uint64(i))
	sum := sha256.Sum256(buf[:])
	return sum[:]
}

func TestSplitDigest(t *testing.T) {
	digest := make([]byte, 32)
	for i := range digest {
		digest[i] = byte(i + 1)
	}

	h1, h2 := SplitDigest(digest)

	wantH1 := binary.LittleEndian.Uint64(digest[:8])
	wantH2 := binary.LittleEndian.Uint64(digest[8:16])

	if h1 != wantH1 {
		t.Errorf("h1 = %x, want %x", h1, wantH1)
	}
	if h2 != wantH2 {
		t.Errorf("h2 = %x, want %x", h2, wantH2)
	}
}

func TestSplitDigestPanicsOnShort(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for short digest, got none")
		}
	}()
	SplitDigest(make([]byte, 15))
}

func TestAddHashedAndHasHashed(t *testing.T) {
	bf, err := New(float64(1<<16), 0.01)
	if err != nil {
		t.Fatal(err)
	}

	const n = 10_000
	digests := make([][]byte, n)
	for i := range n {
		digests[i] = sha256Digest(i)
	}

	// Insert all.
	for _, d := range digests {
		h1, h2 := SplitDigest(d)
		bf.AddHashed(h1, h2)
	}

	if bf.ElementsAdded() != n {
		t.Errorf("ElementsAdded = %d, want %d", bf.ElementsAdded(), n)
	}

	// All inserted items must be found.
	for i, d := range digests {
		h1, h2 := SplitDigest(d)
		if !bf.HasHashed(h1, h2) {
			t.Fatalf("HasHashed returned false for inserted item %d", i)
		}
	}

	// Non-members should rarely match (FP rate ~1%).
	fp := 0
	const probes = 100_000
	for i := range probes {
		d := sha256Digest(n + i)
		h1, h2 := SplitDigest(d)
		if bf.HasHashed(h1, h2) {
			fp++
		}
	}
	fpRate := float64(fp) / float64(probes)
	if fpRate > 0.02 {
		t.Errorf("false positive rate %.4f exceeds 2%% threshold", fpRate)
	}
}

func TestAddIfNotHasHashed(t *testing.T) {
	bf, err := New(float64(1<<16), 0.01)
	if err != nil {
		t.Fatal(err)
	}

	d := sha256Digest(42)
	h1, h2 := SplitDigest(d)

	added := bf.AddIfNotHasHashed(h1, h2)
	if !added {
		t.Error("first AddIfNotHasHashed should return true")
	}

	added = bf.AddIfNotHasHashed(h1, h2)
	if added {
		t.Error("second AddIfNotHasHashed should return false")
	}

	if bf.ElementsAdded() != 1 {
		t.Errorf("ElementsAdded = %d, want 1", bf.ElementsAdded())
	}
}

func TestHashedTSVariants(t *testing.T) {
	bf, err := New(float64(1<<16), 0.01)
	if err != nil {
		t.Fatal(err)
	}

	d := sha256Digest(99)
	h1, h2 := SplitDigest(d)

	bf.AddHashedTS(h1, h2)

	if !bf.HasHashedTS(h1, h2) {
		t.Error("HasHashedTS returned false after AddHashedTS")
	}

	d2 := sha256Digest(100)
	h1b, h2b := SplitDigest(d2)

	added := bf.AddIfNotHasHashedTS(h1b, h2b)
	if !added {
		t.Error("AddIfNotHasHashedTS should return true for new item")
	}
	added = bf.AddIfNotHasHashedTS(h1b, h2b)
	if added {
		t.Error("AddIfNotHasHashedTS should return false for existing item")
	}
}

func TestHashedH2ForcedOdd(t *testing.T) {
	// Verify that an even h2 still works (the method forces it odd).
	bf, err := New(float64(1<<16), 0.01)
	if err != nil {
		t.Fatal(err)
	}

	h1 := uint64(0xAAAA)
	h2 := uint64(0xBB00) // even

	bf.AddHashed(h1, h2)
	if !bf.HasHashed(h1, h2) {
		t.Error("HasHashed should find item added with even h2")
	}
}

func TestHashedWithCustomKeys(t *testing.T) {
	// Custom SipHash keys should not affect the hashed path.
	bf1, _ := New(float64(1<<16), 0.01)
	bf2, _ := NewWithKeys(0x1234, 0x5678, float64(1<<16), 0.01)

	d := sha256Digest(7)
	h1, h2 := SplitDigest(d)

	bf1.AddHashed(h1, h2)
	bf2.AddHashed(h1, h2)

	if !bf1.HasHashed(h1, h2) || !bf2.HasHashed(h1, h2) {
		t.Error("HasHashed should work regardless of SipHash keys")
	}
}
