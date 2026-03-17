// Package bbloom implements a fast bloom filter with a real bitset.
//
// A Bloom filter is a space-efficient probabilistic data structure used to
// test set membership. It may return false positives but never false negatives:
// [Bloom.Has] returning true means the entry was probably added, while false
// means the entry was definitely not added.
//
// This implementation uses an inlined SipHash-2-4 for hashing, rounds the
// bitset up to the next power of two for fast masking, and provides both
// non-thread-safe and mutex-protected (TS-suffixed) variants of all operations.
//
// By default ([New]) the filter uses publicly known SipHash keys. When the
// filter will hold data controlled by untrusted parties, use [NewWithKeys]
// with random secret keys to prevent hash-flooding attacks.
//
// Filters can be serialized to JSON with [Bloom.JSONMarshal] and restored
// with [JSONUnmarshal].
//
// # Pre-hashed keys (Kirsch-Mitzenmacher optimization)
//
// When the input is already a cryptographic hash digest (e.g. SHA2-256),
// the SipHash step can be skipped entirely. [Bloom.AddHashed] and
// [Bloom.HasHashed] read two uint64 values directly from the digest bytes
// and use them as the double-hashing pair. This is based on the result
// by Kirsch and Mitzenmacher ("Less Hashing, Same Performance: Building a
// Better Bloom Filter", 2008, https://doi.org/10.1002/rsa.20208) which
// proves that only two hash values are needed to simulate k independent
// hash functions with no increase in the asymptotic false positive rate.
//
// The pre-hashed path is roughly 4x faster per operation than the SipHash
// path. A filter must use one path consistently for all entries. Mixing
// [Bloom.Add] and [Bloom.AddHashed] on the same filter panics at runtime,
// since they compute different bit positions for the same logical key.
// The chosen mode is preserved across JSON serialization.
package bbloom
