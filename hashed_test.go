package bbloom

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"testing"
)

func sha256Digest(i int) []byte {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], uint64(i))
	sum := sha256.Sum256(buf[:])
	return sum[:]
}

func TestSplitDigestPanicsOnShort(t *testing.T) {
	bf, _ := New(float64(1<<16), 0.01)
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for short digest, got none")
		}
	}()
	bf.AddHashed(make([]byte, 15))
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

	for _, d := range digests {
		bf.AddHashed(d)
	}

	if bf.ElementsAdded() != n {
		t.Errorf("ElementsAdded = %d, want %d", bf.ElementsAdded(), n)
	}

	for i, d := range digests {
		if !bf.HasHashed(d) {
			t.Fatalf("HasHashed returned false for inserted item %d", i)
		}
	}

	// Non-members should rarely match (FP rate ~1%).
	fp := 0
	const probes = 100_000
	for i := range probes {
		d := sha256Digest(n + i)
		if bf.HasHashed(d) {
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

	if added := bf.AddIfNotHasHashed(d); !added {
		t.Error("first AddIfNotHasHashed should return true")
	}
	if added := bf.AddIfNotHasHashed(d); added {
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

	d1 := sha256Digest(99)
	bf.AddHashedTS(d1)
	if !bf.HasHashedTS(d1) {
		t.Error("HasHashedTS returned false after AddHashedTS")
	}

	d2 := sha256Digest(100)
	if added := bf.AddIfNotHasHashedTS(d2); !added {
		t.Error("AddIfNotHasHashedTS should return true for new item")
	}
	if added := bf.AddIfNotHasHashedTS(d2); added {
		t.Error("AddIfNotHasHashedTS should return false for existing item")
	}
}

func TestHashedWithCustomKeys(t *testing.T) {
	// Custom SipHash keys should not affect the hashed path.
	bf1, _ := New(float64(1<<16), 0.01)
	bf2, _ := NewWithKeys(0x1234, 0x5678, float64(1<<16), 0.01)

	d := sha256Digest(7)
	bf1.AddHashed(d)
	bf2.AddHashed(d)

	if !bf1.HasHashed(d) || !bf2.HasHashed(d) {
		t.Error("HasHashed should work regardless of SipHash keys")
	}
}

// --- Hash mode enforcement tests ---

func mustPanic(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("%s: expected panic, got none", name)
		}
	}()
	fn()
}

func TestMixingAddThenAddHashedPanics(t *testing.T) {
	bf, _ := New(float64(1<<16), 0.01)
	bf.Add([]byte("x"))
	mustPanic(t, "AddHashed after Add", func() {
		bf.AddHashed(sha256Digest(0))
	})
}

func TestMixingAddHashedThenAddPanics(t *testing.T) {
	bf, _ := New(float64(1<<16), 0.01)
	bf.AddHashed(sha256Digest(0))
	mustPanic(t, "Add after AddHashed", func() {
		bf.Add([]byte("x"))
	})
}

func TestMixingHasThenHasHashedPanics(t *testing.T) {
	bf, _ := New(float64(1<<16), 0.01)
	bf.Has([]byte("x"))
	mustPanic(t, "HasHashed after Has", func() {
		bf.HasHashed(sha256Digest(0))
	})
}

func TestMixingHasHashedThenHasPanics(t *testing.T) {
	bf, _ := New(float64(1<<16), 0.01)
	bf.HasHashed(sha256Digest(0))
	mustPanic(t, "Has after HasHashed", func() {
		bf.Has([]byte("x"))
	})
}

func TestClearResetsHashMode(t *testing.T) {
	bf, _ := New(float64(1<<16), 0.01)
	bf.Add([]byte("x"))
	bf.Clear()
	d := sha256Digest(0)
	bf.AddHashed(d) // must not panic
	if !bf.HasHashed(d) {
		t.Error("HasHashed should find item after Clear + AddHashed")
	}
}

func TestHashModeJSONRoundTrip(t *testing.T) {
	t.Run("SipHash", func(t *testing.T) {
		bf, _ := New(float64(1<<16), 0.01)
		bf.Add([]byte("x"))
		data := bf.JSONMarshal()

		bf2, err := JSONUnmarshal(data)
		if err != nil {
			t.Fatal(err)
		}
		if !bf2.Has([]byte("x")) {
			t.Error("entry lost after round-trip")
		}
		mustPanic(t, "HasHashed on SipHash filter", func() {
			bf2.HasHashed(sha256Digest(0))
		})
	})

	t.Run("Hashed", func(t *testing.T) {
		bf, _ := New(float64(1<<16), 0.01)
		d := sha256Digest(42)
		bf.AddHashed(d)
		data := bf.JSONMarshal()

		bf2, err := JSONUnmarshal(data)
		if err != nil {
			t.Fatal(err)
		}
		if !bf2.HasHashed(d) {
			t.Error("entry lost after round-trip")
		}
		mustPanic(t, "Has on Hashed filter", func() {
			bf2.Has([]byte("x"))
		})
	})
}

func TestHashModeLegacyJSONDefaultsToSipHash(t *testing.T) {
	bf, _ := New(float64(1<<16), 0.01)
	bf.Add([]byte("legacy"))
	data := bf.JSONMarshal()

	// Strip HashMode to simulate pre-hashMode serialization.
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	delete(raw, "HashMode")
	data2, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}

	bf2, err := JSONUnmarshal(data2)
	if err != nil {
		t.Fatal(err)
	}
	if !bf2.Has([]byte("legacy")) {
		t.Error("entry lost")
	}
	mustPanic(t, "HasHashed on legacy filter", func() {
		bf2.HasHashed(sha256Digest(0))
	})
}
