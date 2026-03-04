# bbloom

[![GoDoc](https://pkg.go.dev/badge/github.com/ipfs/bbloom.svg)](https://pkg.go.dev/github.com/ipfs/bbloom)
[![codecov](https://codecov.io/gh/ipfs/bbloom/branch/master/graph/badge.svg)](https://codecov.io/gh/ipfs/bbloom)

A fast bloom filter with a real bitset, JSON serialization, and thread-safe variants.

Forked from [`AndreasBriese/bbloom`](https://github.com/AndreasBriese/bbloom). Uses an inlined SipHash-2-4 for hashing.

## Install

```sh
go get github.com/ipfs/bbloom
```

## Usage

```go
// create a bloom filter for 65536 items and 1% false-positive rate
bf, _ := bbloom.New(float64(1<<16), float64(0.01))

// or specify size and hash locations explicitly
// bf, _ = bbloom.New(650000.0, 7.0)

// add an item
bf.Add([]byte("butter"))

// check membership
bf.Has([]byte("butter"))    // true
bf.Has([]byte("Butter"))    // false

// add only if not already present
bf.AddIfNotHas([]byte("butter"))  // false (already in set)
bf.AddIfNotHas([]byte("buTTer"))  // true  (new entry)

// thread-safe variants: AddTS, HasTS, AddIfNotHasTS
bf.AddTS([]byte("peanutbutter"))
bf.HasTS([]byte("peanutbutter"))  // true

// JSON serialization
data := bf.JSONMarshal()
restored, _ := bbloom.JSONUnmarshal(data)
restored.Has([]byte("butter"))    // true
```

## Benchmarks

See [BENCHMARKS.md](BENCHMARKS.md) for comparison against other bloom filter libraries.

## License

MIT (bbloom) and CC0 (inlined SipHash). See [LICENSE](LICENSE).
