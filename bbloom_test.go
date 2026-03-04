package bbloom

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
)

var (
	wordlist1 [][]byte
	n         = 1 << 16
	bf        Bloom
)

func TestMain(m *testing.M) {
	wordlist1 = make([][]byte, n)
	for i := range wordlist1 {
		wordlist1[i] = []byte("word-" + strconv.Itoa(i))
	}
	fmt.Println("\n###############\nbbloom_test.go")
	fmt.Print("Benchmarks relate to 2**16 OP. --> output/65536 op/ns\n###############\n\n")

	os.Exit(m.Run())
}

func TestM_NumberOfWrongs(t *testing.T) {
	bf, err := New(float64(n*10), float64(7))
	if err != nil {
		t.Fatal(err)
	}

	cnt := 0
	for i := range wordlist1 {
		if !bf.AddIfNotHas(wordlist1[i]) {
			cnt++
		}
	}
	pct := float64(cnt) / float64(n) * 100
	t.Logf("false positives: %d/%d (%.2f%%)", cnt, n, pct)
	if pct > 1.0 {
		t.Errorf("false positive rate too high: %.2f%% (expected <1%%)", pct)
	}
}

func TestM_JSON(t *testing.T) {
	const shallBe = int(1 << 16)

	bf, err := New(float64(n*10), float64(7))
	if err != nil {
		t.Fatal(err)
	}

	cnt := 0
	for i := range wordlist1 {
		if !bf.AddIfNotHas(wordlist1[i]) {
			cnt++
		}
	}

	json := bf.JSONMarshal()
	if err != nil {
		t.Fatal(err)
	}

	// create new bloomfilter from bloomfilter's JSON representation
	bf2, err := JSONUnmarshal(json)
	if err != nil {
		t.Fatal(err)
	}

	cnt2 := 0
	for i := range wordlist1 {
		if !bf2.AddIfNotHas(wordlist1[i]) {
			cnt2++
		}
	}

	if cnt2 != shallBe {
		t.Errorf("FAILED !AddIfNotHas = %v; want %v", cnt2, shallBe)
	}
}
func TestNewWithKeys(t *testing.T) {
	k0 := uint64(0x0123456789abcdef)
	k1 := uint64(0xfedcba9876543210)

	bf1, err := NewWithKeys(k0, k1, float64(n*10), float64(7))
	if err != nil {
		t.Fatal(err)
	}
	bf2, err := New(float64(n*10), float64(7))
	if err != nil {
		t.Fatal(err)
	}

	// same entry should hash to different positions with different keys
	entry := []byte("test-entry")
	l1, h1 := bf1.sipHash(entry)
	l2, h2 := bf2.sipHash(entry)
	if l1 == l2 && h1 == h2 {
		t.Fatal("custom keys produced same hash as default keys")
	}

	// filter should still work correctly with custom keys
	for i := range wordlist1 {
		bf1.Add(wordlist1[i])
	}
	for i := range wordlist1 {
		if !bf1.Has(wordlist1[i]) {
			t.Fatalf("Has(%q) = false after Add", wordlist1[i])
		}
	}
}

func TestNewWithKeysJSON(t *testing.T) {
	k0 := uint64(0x0123456789abcdef)
	k1 := uint64(0xfedcba9876543210)

	bf, err := NewWithKeys(k0, k1, float64(n*10), float64(7))
	if err != nil {
		t.Fatal(err)
	}

	entries := wordlist1[:1000]
	for _, e := range entries {
		bf.Add(e)
	}

	data := bf.JSONMarshal()

	bf2, err := JSONUnmarshal(data)
	if err != nil {
		t.Fatal(err)
	}

	// keys should be preserved
	if bf2.k0 != k0 || bf2.k1 != k1 {
		t.Fatalf("keys not preserved: got k0=%x k1=%x, want k0=%x k1=%x", bf2.k0, bf2.k1, k0, k1)
	}

	for _, e := range entries {
		if !bf2.Has(e) {
			t.Fatalf("custom-key filter lost entry %q after JSON round-trip", e)
		}
	}
}

func TestDefaultKeysOmittedFromJSON(t *testing.T) {
	bf, err := New(float64(512), float64(3))
	if err != nil {
		t.Fatal(err)
	}
	bf.Add([]byte("test"))

	data := bf.JSONMarshal()
	s := string(data)
	if strings.Contains(s, "K0") || strings.Contains(s, "K1") {
		t.Fatalf("default keys should not appear in JSON: %s", s)
	}

	// custom keys should appear
	bf2, err := NewWithKeys(42, 99, float64(512), float64(3))
	if err != nil {
		t.Fatal(err)
	}
	bf2.Add([]byte("test"))

	data2 := bf2.JSONMarshal()
	s2 := string(data2)
	if !strings.Contains(s2, "K0") || !strings.Contains(s2, "K1") {
		t.Fatalf("custom keys should appear in JSON: %s", s2)
	}
}

func TestFillRatio(t *testing.T) {
	bf, err := New(float64(512), float64(7))
	if err != nil {
		t.Fatal(err)
	}
	bf.Add([]byte("test"))
	r := bf.FillRatio()
	if math.Abs(r-float64(7)/float64(512)) > 0.001 {
		t.Error("ratio doesn't work")
	}
}

func ExampleBloom_AddIfNotHas() {
	bf, err := New(float64(512), float64(1))
	if err != nil {
		panic(err)
	}

	fmt.Printf("%v %v %v %v\n", bf.sizeExp, bf.size, bf.setLocs, bf.shift)

	bf.Add([]byte("Manfred"))
	fmt.Println("bf.Add([]byte(\"Manfred\"))")
	fmt.Printf("bf.Has([]byte(\"Manfred\")) -> %v - should be true\n", bf.Has([]byte("Manfred")))
	fmt.Printf("bf.Add([]byte(\"manfred\")) -> %v - should be false\n", bf.Has([]byte("manfred")))
	fmt.Printf("bf.AddIfNotHas([]byte(\"Manfred\")) -> %v - should be false\n", bf.AddIfNotHas([]byte("Manfred")))
	fmt.Printf("bf.AddIfNotHas([]byte(\"manfred\")) -> %v - should be true\n", bf.AddIfNotHas([]byte("manfred")))

	bf.AddTS([]byte("Hans"))
	fmt.Println("bf.AddTS([]byte(\"Hans\")")
	fmt.Printf("bf.HasTS([]byte(\"Hans\")) -> %v - should be true\n", bf.HasTS([]byte("Hans")))
	fmt.Printf("bf.AddTS([]byte(\"hans\")) -> %v - should be false\n", bf.HasTS([]byte("hans")))
	fmt.Printf("bf.AddIfNotHasTS([]byte(\"Hans\")) -> %v - should be false\n", bf.AddIfNotHasTS([]byte("Hans")))
	fmt.Printf("bf.AddIfNotHasTS([]byte(\"hans\")) -> %v - should be true\n", bf.AddIfNotHasTS([]byte("hans")))

	// Output: 9 511 1 55
	// bf.Add([]byte("Manfred"))
	// bf.Has([]byte("Manfred")) -> true - should be true
	// bf.Add([]byte("manfred")) -> false - should be false
	// bf.AddIfNotHas([]byte("Manfred")) -> false - should be false
	// bf.AddIfNotHas([]byte("manfred")) -> true - should be true
	// bf.AddTS([]byte("Hans")
	// bf.HasTS([]byte("Hans")) -> true - should be true
	// bf.AddTS([]byte("hans")) -> false - should be false
	// bf.AddIfNotHasTS([]byte("Hans")) -> false - should be false
	// bf.AddIfNotHasTS([]byte("hans")) -> true - should be true
}

func BenchmarkM_New(b *testing.B) {
	for r := 0; r < b.N; r++ {
		_, _ = New(float64(n*10), float64(7))
	}
}

func BenchmarkM_Clear(b *testing.B) {
	bf, err := New(float64(n*10), float64(7))
	if err != nil {
		b.Fatal(err)
	}
	for i := range wordlist1 {
		bf.Add(wordlist1[i])
	}
	b.ResetTimer()
	for r := 0; r < b.N; r++ {
		bf.Clear()
	}
}

func BenchmarkM_Add(b *testing.B) {
	bf, err := New(float64(n*10), float64(7))
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for r := 0; r < b.N; r++ {
		for i := range wordlist1 {
			bf.Add(wordlist1[i])
		}
	}

}

func BenchmarkM_Has(b *testing.B) {
	b.ResetTimer()
	for r := 0; r < b.N; r++ {
		for i := range wordlist1 {
			bf.Has(wordlist1[i])
		}
	}

}

func BenchmarkM_AddIfNotHasFALSE(b *testing.B) {
	bf, err := New(float64(n*10), float64(7))
	if err != nil {
		b.Fatal(err)
	}
	for i := range wordlist1 {
		bf.Has(wordlist1[i])
	}
	b.ResetTimer()
	for r := 0; r < b.N; r++ {
		for i := range wordlist1 {
			bf.AddIfNotHas(wordlist1[i])
		}
	}
}

func BenchmarkM_AddIfNotHasClearTRUE(b *testing.B) {
	bf, err := New(float64(n*10), float64(7))
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for r := 0; r < b.N; r++ {
		for i := range wordlist1 {
			bf.AddIfNotHas(wordlist1[i])
		}
		bf.Clear()
	}
}

func BenchmarkM_AddTS(b *testing.B) {
	bf, err := New(float64(n*10), float64(7))
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for r := 0; r < b.N; r++ {
		for i := range wordlist1 {
			bf.AddTS(wordlist1[i])
		}
	}

}

func BenchmarkM_HasTS(b *testing.B) {
	b.ResetTimer()
	for r := 0; r < b.N; r++ {
		for i := range wordlist1 {
			bf.HasTS(wordlist1[i])
		}
	}

}

func BenchmarkM_AddIfNotHasTSFALSE(b *testing.B) {
	bf, err := New(float64(n*10), float64(7))
	if err != nil {
		b.Fatal(err)
	}
	for i := range wordlist1 {
		bf.Has(wordlist1[i])
	}
	b.ResetTimer()
	for r := 0; r < b.N; r++ {
		for i := range wordlist1 {
			bf.AddIfNotHasTS(wordlist1[i])
		}
	}
}

func BenchmarkM_AddIfNotHasTSClearTRUE(b *testing.B) {
	bf, err := New(float64(n*10), float64(7))
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for r := 0; r < b.N; r++ {
		for i := range wordlist1 {
			bf.AddIfNotHasTS(wordlist1[i])
		}
		bf.Clear()
	}

}
