package bbloom

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"testing"
)

// Simulate a SHA2-256 multihash: [0x12, 0x20, <32 bytes digest>]
func makeMultihashes(n int) (mhs [][]byte, digests [][]byte) {
	mhs = make([][]byte, n)
	digests = make([][]byte, n)
	for i := range n {
		// Hash a unique input to get a realistic SHA2-256 digest.
		var buf [8]byte
		binary.LittleEndian.PutUint64(buf[:], uint64(i))
		sum := sha256.Sum256(buf[:])

		// Build multihash: varint(0x12) + varint(0x20) + 32-byte digest
		mh := make([]byte, 2+32)
		mh[0] = 0x12 // SHA2-256
		mh[1] = 0x20 // 32 bytes
		copy(mh[2:], sum[:])

		mhs[i] = mh
		digests[i] = mh[2:] // sub-slice, same as multihash.Decode().Digest
	}
	return mhs, digests
}

const benchN = 2_000_000

// Pre-generate test data once and reuse across sub-benchmarks.
var (
	benchMHs     [][]byte
	benchDigests [][]byte
	benchH1s     []uint64
	benchH2s     []uint64
)

func initBenchData() {
	if benchMHs != nil {
		return
	}
	benchMHs, benchDigests = makeMultihashes(benchN)
	benchH1s = make([]uint64, benchN)
	benchH2s = make([]uint64, benchN)
	for i, d := range benchDigests {
		benchH1s[i], benchH2s[i] = SplitDigest(d)
	}
}

// newBenchBloom creates a bloom sized for benchN items at 1% FP rate.
func newBenchBloom() *Bloom {
	bf, err := New(float64(benchN), 0.01)
	if err != nil {
		panic(err)
	}
	return bf
}

// ---------- Add benchmarks ----------

func BenchmarkAdd2M_SipHash(b *testing.B) {
	initBenchData()
	for range b.N {
		bf := newBenchBloom()
		for _, mh := range benchMHs {
			bf.Add(mh)
		}
	}
}

func BenchmarkAdd2M_Hashed(b *testing.B) {
	initBenchData()
	for range b.N {
		bf := newBenchBloom()
		for i := range benchN {
			bf.AddHashed(benchH1s[i], benchH2s[i])
		}
	}
}

func BenchmarkAdd2M_HashedWithSplit(b *testing.B) {
	initBenchData()
	for range b.N {
		bf := newBenchBloom()
		for _, d := range benchDigests {
			h1, h2 := SplitDigest(d)
			bf.AddHashed(h1, h2)
		}
	}
}

// ---------- Has benchmarks (filter pre-populated) ----------

func BenchmarkHas2M_SipHash(b *testing.B) {
	initBenchData()
	bf := newBenchBloom()
	for _, mh := range benchMHs {
		bf.Add(mh)
	}
	b.ResetTimer()
	for range b.N {
		for _, mh := range benchMHs {
			bf.Has(mh)
		}
	}
}

func BenchmarkHas2M_Hashed(b *testing.B) {
	initBenchData()
	bf := newBenchBloom()
	for i := range benchN {
		bf.AddHashed(benchH1s[i], benchH2s[i])
	}
	b.ResetTimer()
	for range b.N {
		for i := range benchN {
			bf.HasHashed(benchH1s[i], benchH2s[i])
		}
	}
}

func BenchmarkHas2M_HashedWithSplit(b *testing.B) {
	initBenchData()
	bf := newBenchBloom()
	for i := range benchN {
		bf.AddHashed(benchH1s[i], benchH2s[i])
	}
	b.ResetTimer()
	for range b.N {
		for _, d := range benchDigests {
			h1, h2 := SplitDigest(d)
			bf.HasHashed(h1, h2)
		}
	}
}

// ---------- AddIfNotHas benchmarks ----------

func BenchmarkAddIfNotHas2M_SipHash(b *testing.B) {
	initBenchData()
	for range b.N {
		bf := newBenchBloom()
		for _, mh := range benchMHs {
			bf.AddIfNotHas(mh)
		}
	}
}

func BenchmarkAddIfNotHas2M_Hashed(b *testing.B) {
	initBenchData()
	for range b.N {
		bf := newBenchBloom()
		for i := range benchN {
			bf.AddIfNotHasHashed(benchH1s[i], benchH2s[i])
		}
	}
}

// ---------- Per-op benchmarks (for ns/op comparison) ----------

func BenchmarkAddSingle_SipHash(b *testing.B) {
	initBenchData()
	bf := newBenchBloom()
	mh := benchMHs[0]
	b.ResetTimer()
	for i := range b.N {
		_ = i
		bf.Add(mh)
	}
}

func BenchmarkAddSingle_Hashed(b *testing.B) {
	initBenchData()
	bf := newBenchBloom()
	h1, h2 := benchH1s[0], benchH2s[0]
	b.ResetTimer()
	for range b.N {
		bf.AddHashed(h1, h2)
	}
}

func BenchmarkHasSingle_SipHash(b *testing.B) {
	initBenchData()
	bf := newBenchBloom()
	mh := benchMHs[0]
	bf.Add(mh)
	b.ResetTimer()
	for range b.N {
		bf.Has(mh)
	}
}

func BenchmarkHasSingle_Hashed(b *testing.B) {
	initBenchData()
	bf := newBenchBloom()
	h1, h2 := benchH1s[0], benchH2s[0]
	bf.AddHashed(h1, h2)
	b.ResetTimer()
	for range b.N {
		bf.HasHashed(h1, h2)
	}
}

// ---------- Summary: wall-clock for 2M CIDs ----------

func BenchmarkSummary2M(b *testing.B) {
	initBenchData()

	b.Run("Add_SipHash", func(b *testing.B) {
		for range b.N {
			bf := newBenchBloom()
			for _, mh := range benchMHs {
				bf.Add(mh)
			}
		}
	})

	b.Run("Add_Hashed", func(b *testing.B) {
		for range b.N {
			bf := newBenchBloom()
			for _, d := range benchDigests {
				h1, h2 := SplitDigest(d)
				bf.AddHashed(h1, h2)
			}
		}
	})

	b.Run("AddIfNotHas_SipHash", func(b *testing.B) {
		for range b.N {
			bf := newBenchBloom()
			for _, mh := range benchMHs {
				bf.AddIfNotHas(mh)
			}
		}
	})

	b.Run("AddIfNotHas_Hashed", func(b *testing.B) {
		for range b.N {
			bf := newBenchBloom()
			for _, d := range benchDigests {
				h1, h2 := SplitDigest(d)
				bf.AddIfNotHasHashed(h1, h2)
			}
		}
	})

	// Pre-populate for Has benchmarks
	bfSip := newBenchBloom()
	for _, mh := range benchMHs {
		bfSip.Add(mh)
	}
	bfHash := newBenchBloom()
	for _, d := range benchDigests {
		h1, h2 := SplitDigest(d)
		bfHash.AddHashed(h1, h2)
	}

	b.Run("Has_SipHash", func(b *testing.B) {
		for range b.N {
			for _, mh := range benchMHs {
				bfSip.Has(mh)
			}
		}
	})

	b.Run("Has_Hashed", func(b *testing.B) {
		for range b.N {
			for _, d := range benchDigests {
				h1, h2 := SplitDigest(d)
				bfHash.HasHashed(h1, h2)
			}
		}
	})

	// Print fill ratios to sanity-check both paths behave similarly
	b.Logf("FillRatio SipHash=%.4f  Hashed=%.4f  (items: %d)",
		bfSip.FillRatio(), bfHash.FillRatio(), benchN)

	// Check FP rates on non-members to confirm Hashed path is sound
	fpSip, fpHash := 0, 0
	_, extra := makeMultihashes(100_000)
	// Use indices offset by benchN so these are guaranteed non-members
	extraMHs := make([][]byte, 100_000)
	for i := range 100_000 {
		var buf [8]byte
		binary.LittleEndian.PutUint64(buf[:], uint64(benchN+i))
		sum := sha256.Sum256(buf[:])
		mh := make([]byte, 34)
		mh[0] = 0x12
		mh[1] = 0x20
		copy(mh[2:], sum[:])
		extraMHs[i] = mh
	}
	for _, mh := range extraMHs {
		if bfSip.Has(mh) {
			fpSip++
		}
		h1, h2 := SplitDigest(mh[2:])
		if bfHash.HasHashed(h1, h2) {
			fpHash++
		}
	}
	b.Logf("FP rate (100k non-members): SipHash=%.4f%%  Hashed=%.4f%%",
		float64(fpSip)/1000, float64(fpHash)/1000)
	_ = extra
	_ = fmt.Sprint // keep fmt imported
}
