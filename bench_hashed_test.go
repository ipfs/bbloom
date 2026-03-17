package bbloom

import (
	"crypto/sha256"
	"encoding/binary"
	"testing"
)

// Simulate SHA2-256 multihashes: [0x12, 0x20, <32 bytes digest>].
func makeSHA256Multihashes(n int) (mhs [][]byte, digests [][]byte) {
	mhs = make([][]byte, n)
	digests = make([][]byte, n)
	for i := range n {
		var buf [8]byte
		binary.LittleEndian.PutUint64(buf[:], uint64(i))
		sum := sha256.Sum256(buf[:])

		mh := make([]byte, 2+32)
		mh[0] = 0x12 // SHA2-256
		mh[1] = 0x20 // 32 bytes
		copy(mh[2:], sum[:])

		mhs[i] = mh
		digests[i] = mh[2:]
	}
	return mhs, digests
}

// Pre-generated test data shared across benchmarks.
var (
	benchMHs     [][]byte
	benchDigests [][]byte
	benchH1s     []uint64
	benchH2s     []uint64
)

const benchN = 2_000_000

func initBenchData() {
	if benchMHs != nil {
		return
	}
	benchMHs, benchDigests = makeSHA256Multihashes(benchN)
	benchH1s = make([]uint64, benchN)
	benchH2s = make([]uint64, benchN)
	for i, d := range benchDigests {
		benchH1s[i], benchH2s[i] = SplitDigest(d)
	}
}

func newBenchBloom() *Bloom {
	bf, err := New(float64(benchN), 0.01)
	if err != nil {
		panic(err)
	}
	return bf
}

// --- Per-op benchmarks (ns/op) ---

func BenchmarkPerOp_Add_SipHash(b *testing.B) {
	initBenchData()
	bf := newBenchBloom()
	mh := benchMHs[0]
	b.ResetTimer()
	for range b.N {
		bf.Add(mh)
	}
}

func BenchmarkPerOp_Add_Hashed(b *testing.B) {
	initBenchData()
	bf := newBenchBloom()
	h1, h2 := benchH1s[0], benchH2s[0]
	b.ResetTimer()
	for range b.N {
		bf.AddHashed(h1, h2)
	}
}

func BenchmarkPerOp_Has_SipHash(b *testing.B) {
	initBenchData()
	bf := newBenchBloom()
	mh := benchMHs[0]
	bf.Add(mh)
	b.ResetTimer()
	for range b.N {
		bf.Has(mh)
	}
}

func BenchmarkPerOp_Has_Hashed(b *testing.B) {
	initBenchData()
	bf := newBenchBloom()
	h1, h2 := benchH1s[0], benchH2s[0]
	bf.AddHashed(h1, h2)
	b.ResetTimer()
	for range b.N {
		bf.HasHashed(h1, h2)
	}
}

// --- Bulk benchmarks (2M SHA2-256 CIDs, wall-clock) ---

func BenchmarkBulk2M_Add_SipHash(b *testing.B) {
	initBenchData()
	for range b.N {
		bf := newBenchBloom()
		for _, mh := range benchMHs {
			bf.Add(mh)
		}
	}
}

func BenchmarkBulk2M_Add_Hashed(b *testing.B) {
	initBenchData()
	for range b.N {
		bf := newBenchBloom()
		for _, d := range benchDigests {
			h1, h2 := SplitDigest(d)
			bf.AddHashed(h1, h2)
		}
	}
}

func BenchmarkBulk2M_Has_SipHash(b *testing.B) {
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

func BenchmarkBulk2M_Has_Hashed(b *testing.B) {
	initBenchData()
	bf := newBenchBloom()
	for _, d := range benchDigests {
		h1, h2 := SplitDigest(d)
		bf.AddHashed(h1, h2)
	}
	b.ResetTimer()
	for range b.N {
		for _, d := range benchDigests {
			h1, h2 := SplitDigest(d)
			bf.HasHashed(h1, h2)
		}
	}
}

func BenchmarkBulk2M_AddIfNotHas_SipHash(b *testing.B) {
	initBenchData()
	for range b.N {
		bf := newBenchBloom()
		for _, mh := range benchMHs {
			bf.AddIfNotHas(mh)
		}
	}
}

func BenchmarkBulk2M_AddIfNotHas_Hashed(b *testing.B) {
	initBenchData()
	for range b.N {
		bf := newBenchBloom()
		for _, d := range benchDigests {
			h1, h2 := SplitDigest(d)
			bf.AddIfNotHasHashed(h1, h2)
		}
	}
}
