package base32

import (
	crypto "crypto/rand"
	"fmt"
	"io"
	"math/rand"
	"strings"
	"testing"
)

func TestEncode(t *testing.T) {

	// encodingTestCases is a comprehensive array of all Base32 numbers from 0
	// to 2ZZ.
	for i, expected := range encodingTestCases {
		var output Base32
		var input uint32

		input = uint32(i)
		output = Encode(input)
		if output != expected {
			t.Fatalf("Expected Encode(%d) to be %q, but got %q.",
				input, expected, output)
		}
	}

	// encodingMap is a sparsely grouped map of int->Base32 test cases.
	for i, expected := range encodingMap {
		var output Base32
		var input uint32

		input = uint32(i)
		output = Encode(input)
		if output != Base32(expected) {
			t.Fatalf("Expected Encode(%d) to be %q, but got %q.",
				input, expected, output)
		}
	}
}

func TestFromString(t *testing.T) {
	var cases = map[string]Base32{
		"0":             Base32("0"),
		"o":             Base32("0"),
		"123":           Base32("123"),
		"ZA0T":          Base32("ZA0T"),
		"abcd1":         Base32("ABCD1"),
		"00ZZZ":         Base32("ZZZ"),
		"AAA-bbb-o-l":   Base32("AAABBB01"),
		"00-Example-00": Base32("EXAMP1E00"),
	}

	for input, expected := range cases {
		t.Log("FromString valid input", input, expected)
		output, err := FromString(input)
		if err != nil {
			t.Errorf("Expected FromString(%q) to be %q, got error %s.",
				input, expected, err)
		} else if output != expected {
			t.Errorf("Expected FromString(%q) to be %q, got %q.",
				input, expected, output)
		}
	}

	var invalid = [...]string{
		"CUT", // U is an invalid character
		"",    // Empty string is an invalid Base32 value.
		"a*b", // * is an invalid character
		"a b", // space is an invalid character
	}

	for _, input := range invalid {
		_, err := FromString(input)
		if err == nil {
			t.Errorf("Expected FromString(%q) to return an error, got nil.", input)
		}
	}
}

func TestBase32_Decode(t *testing.T) {

	for expected, base32 := range encodingTestCases {
		var output uint32
		var err error

		output, err = base32.Decode()
		if err != nil {
			t.Fatalf("Expected Base32.Decode() to be successful, but got error %q.\n", err)
		}
		if output != uint32(expected) {
			t.Fatalf("Expected %q.Decode() to be %d, got %d.\n", base32, expected, output)
		}
	}

	for expected, base32 := range encodingMap {
		var output uint32
		var err error
		output, err = Base32(base32).Decode()

		if err != nil {
			t.Fatalf("Expected %q.Decode() to be successful, but got error %q.\n", base32, err)
		}
		if output != uint32(expected) {
			t.Fatalf("Expected %q.Decode() to be %d, got %d.\n", base32, expected, output)
		}
	}

	for _, base32 := range invalidDecodingTestCases {
		_, err := base32.Decode()
		if err == nil {
			t.Fatalf("Expected invalid Base32 value, %q, to return an error message from Decode(), got nil.\n", base32)
		}
	}

	// Decode needs to be robust against the common errors. The above test cases
	// do not check for that. These do:

	var cases = []struct {
		Encoded  Base32
		Expected uint32
	}{
		{Base32("o"), 0},
		{Base32("o0"), 0},
		{Base32("0l"), 1},
	}

	for _, c := range cases {
		var output uint32
		output, err := c.Encoded.Decode()
		if err != nil || output != c.Expected {
			t.Errorf("Expected Base32(%q).Decode() to return %d, <nil>; got %d, %#v",
				c.Encoded, c.Expected, output, err)
		}
	}

}

func TestBase32_IsValid(t *testing.T) {

	for base32, expected := range isValidTestCases {
		var valid bool
		valid = base32.IsValid(expected)
		if !valid {
			t.Errorf("Expecting %q.IsValid() to be true, got false.\n", base32)
		}

	}
}

// TestEncodeDecode takes a random uint, encodes that value, and then decodes
// it. If the final decoded result is not equal to the original input integer,
// then something is very wrong. This test is performed thousands of times on
// thousands of randomly generated integers.
func TestEncodeDecode(t *testing.T) {
	const n = 100000

	// Test: Encode and then decode, and the resulting output should equal
	// the original input.

	for i := 0; i < n; i++ {
		randInput := rand.Uint32()
		base32 := Encode(randInput)
		base32 = Base32(strings.ToLower(string(base32))) // ToLower() just makes things interesting.
		base10, err := base32.Decode()
		if err != nil {
			t.Errorf("Expected %q.Decode() to succeed, got error %q.", base32, err)
		} else if base10 != randInput {
			t.Errorf("Expected %q.Decode() to be %d, got %d.", base32, randInput, base10)
		}
	}
}

// TestMalformed makes sure Base32.Decode() fails explicitly when given an
// invalid input.
func TestMalformed(t *testing.T) {
	const testN = 10000

	for i := 0; i < testN; i++ {
		bytes := generateInvalidBase32Bytes()
		base32 := Base32(bytes)
		_, err := base32.Decode()
		if err == nil {
			t.Errorf("Expected %q to be invalid, but got no error on Decode().", base32)
		}
	}
}

func TestBase32_String(t *testing.T) {
	cases := []struct {
		input    Base32
		expected string
	}{
		{Base32("EXAMP1E"), "EXAMP1E"},
		{InvalidBase32Value, "<invalid>"},
	}

	for _, c := range cases {
		actual := c.input.String()
		if actual != c.expected {
			t.Errorf("Expected Base32(%q).String() to be %q, got %q", c.input, c.expected, actual)
		}
	}
}

func TestCheck_String(t *testing.T) {
	cases := []struct {
		input    Check
		expected string
	}{
		{Check('U'), "U"},
		{InvalidCheckValue, "<invalid>"},
	}

	for _, c := range cases {
		actual := c.input.String()
		if actual != c.expected {
			t.Errorf("Expected Check(%q).String() to be %q, got %q", c.input, c.expected, actual)
		}
	}
}

// Generates a slice of bytes that contains bytes that are not valid Base32
// ASCII-encoded digits. The length of the slice is random, at least 1. The
// contents of the slice are random but guaranteed to have at least one invalid
// byte.
func generateInvalidBase32Bytes() []byte {
	const maxByteLen int = 30
	const lowestValidValue int = '0' // 48 = ASCII "0", the lowest valid Base32 value

	numBytes := rand.Intn(maxByteLen) + 1
	bytes := make([]byte, numBytes)

	n, err := io.ReadFull(crypto.Reader, bytes)
	if n != len(bytes) || err != nil {
		panic("Unable to generate random bytes.")
	}

	// Guarantee an invalid byte stream by injecting a known invalid byte.
	randInvalidByte := byte(rand.Intn(lowestValidValue))
	randIdx := rand.Intn(numBytes)
	bytes[randIdx] = randInvalidByte

	return bytes
}

func TestBase32_Pad(t *testing.T) {
	const width = uint8(5)
	cases := map[Base32]string{
		"Z":      "0000Z",
		"ABC":    "00ABC",
		"ABCDEF": "ABCDEF",
	}
	for input, expected := range cases {
		output := input.Pad(width)
		if string(output) != expected {
			t.Errorf("For %q.Pad(%d), expected %q, got %q.", input, width,
				expected, output)
		}
	}
}

func TestTrim(t *testing.T) {
	cases := map[string]Base32{
		"A":            "A",
		"00A":          "A",
		"0000000A":     "A",
		"00ooOOB":      "B",
		"00-00-00-C":   "C",
		"00-oo-00TEST": "TEST",
	}

	for input, expected := range cases {
		output := Trim(input)
		if output != expected {
			t.Errorf("Expected Trim(%q) to be %q, got %q.",
				input, expected, output)
		}
	}
}

func TestBase32_WillFit(t *testing.T) {
	willFit(t, Base32("Z"))
	willFit(t, Base32("0Z"))
	willFit(t, Base32("ZZ"))
	willFit(t, Base32("ZZZ"))
	willFit(t, Base32("ZZZZ"))
	willFit(t, Base32("ZZZZZ"))
	willFit(t, Base32("ZZZZZZ"))
	willFit(t, Base32("0ZZZZZZ"))
	willFit(t, Base32("1ZZZZZZ"))
	willFit(t, Base32("2ZZZZZZ"))
	willFit(t, Base32("3ZZZZZZ"))

	wontFit(t, Base32("4000000"))
	wontFit(t, Base32("4ZZZZZZ"))
	wontFit(t, Base32("5ZZZZZZ"))
	wontFit(t, Base32("ZZZZZZZ"))
	wontFit(t, Base32("ZZZZZZZZ"))
}

func willFit(t *testing.T, base32 Base32) {
	if base32.WillFit() == false {
		t.Errorf("Expected %q.WillFit() to be true, got false.", base32)
	}
}

func wontFit(t *testing.T, base32 Base32) {
	if base32.WillFit() == true {
		t.Errorf("Expected %q.WillFit() to be false, got true.", base32)
	}
}

var isValidTestCases = map[Base32]Check{
	"z": 'Z',
	// TODO: More!
}

var invalidDecodingTestCases = [...]Base32{
	"fun",
	"",
	"nothing goes here",
	"BEEF!",
	"\u0000",
}

// BenchmarkEncode  2000000         807   ns/op                                   # string concat uint32
// BenchmarkEncode 10000000         132   ns/op                                   # [7]byte buffer uint32
// BenchmarkEncode 10000000         202   ns/op                                   # Adds Check retval
// BenchmarkEncode 10000000         134   ns/op         8 B/op        1 allocs/op
// BenchmarkEncode 20000000          93.1 ns/op         8 B/op        1 allocs/op # Go 1.2
func BenchmarkEncode(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Encode(123123123)
	}
}

// BenchmarkDecode 20000000          79.8 ns/op # bit shifting
// BenchmarkDecode 20000000          80.1 ns/op         0 B/op        0 allocs/op # Uses validBase32Digit map to check for valid rune.
// BenchmarkDecode 50000000          35.8 ns/op         0 B/op        0 allocs/op # Do manual check on rune to see if its valid.
// BenchmarkDecode 50000000          56.5 ns/op         0 B/op        0 allocs/op # Add checks for 'o' and 'O'.
// BenchmarkDecode 50000000          52.0 ns/op         0 B/op        0 allocs/op # Adds invalidDecodeRune
// BenchmarkDecode 50000000          54.0 ns/op         0 B/op        0 allocs/op # Go 1.2
func BenchmarkDecode(b *testing.B) {
	var base32 Base32

	base32 = Base32("N0NoN0")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = base32.Decode()
	}
}

// BenchmarkWillFit  2000000000           1.56 ns/op        0 B/op        0 allocs/op # 6-digit
// BenchmarkWillFit  1000000000           2.01 ns/op        0 B/op        0 allocs/op # 8-digit
// BenchmarkWillFit   100000000          14.5 ns/op         0 B/op        0 allocs/op # 7-digit, string compare
// BenchmarkWillFit  1000000000           2.39 ns/op        0 B/op        0 allocs/op # 7-digit, rune compare
// BenchmarkWillFit  500000000            6.44 ns/op        0 B/op        0 allocs/op # Go 1.2
func BenchmarkWillFit(b *testing.B) {
	var base32 Base32

	base32 = Base32("3ZZZZZZ") // worst case
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = base32.WillFit()
	}
}

// BenchmarkFromString_easy  50000000          68.0 ns/op         0 B/op        0 allocs/op # For an easy input value
// BenchmarkFromString_easy  50000000          64.6 ns/op         0 B/op        0 allocs/op # delete using copy()
// BenchmarkFromString_easy  50000000          62.9 ns/op         0 B/op        0 allocs/op # Manual validity check
// BenchmarkFromString_easy  50000000          61.7 ns/op         0 B/op        0 allocs/op # Go 1.2
func BenchmarkFromString_easy(b *testing.B) {
	input := "EXAMP1E" // An easy input value
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = FromString(input)
	}
}

// BenchmarkFromString_hard   1000000        1470 ns/op         180 B/op        7 allocs/op # For a hard input value
// BenchmarkFromString_hard   5000000         672 ns/op          32 B/op        2 allocs/op # Hard input, more manual fiddling
// BenchmarkFromString_hard   5000000         616 ns/op          32 B/op        2 allocs/op # delete using copy()
// BenchmarkFromString_hard   5000000         473 ns/op          32 B/op        2 allocs/op # Manual validity check
// BenchmarkFromString_hard   5000000         443 ns/op          32 B/op        2 allocs/op # Copy over bytes manually
// BenchmarkFromString_hard   5000000         406 ns/op          32 B/op        2 allocs/op # More manual fiddling.
// BenchmarkFromString_hard   5000000         417 ns/op          24 B/op        2 allocs/op # Interior hyphen count
// BenchmarkFromString_hard   5000000         367 ns/op          24 B/op        2 allocs/op # Go 1.2
func BenchmarkFromString_hard(b *testing.B) {
	input := "AAA-bbb-o-l" // A hard input value
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = FromString(input)
	}
}

// BenchmarkFromString_invalid  1000000        1106 ns/op         103 B/op        6 allocs/op # For an invalid input value
// BenchmarkFromString_invalid 10000000         283 ns/op           0 B/op        0 allocs/op # Invalid input, do check before alloc
// BenchmarkFromString_invalid 10000000         280 ns/op           0 B/op        0 allocs/op # delete using copy()
// BenchmarkFromString_invalid 10000000         150 ns/op           0 B/op        0 allocs/op # Manual validity check
// BenchmarkFromString_invalid 10000000         156 ns/op           0 B/op        0 allocs/op # Go 1.2
func BenchmarkFromString_invalid(b *testing.B) {
	input := "--00invalid***/" // An invalid input value
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = FromString(input)
	}
}

// BenchmarkTrim 20000000          71.2 ns/op         0 B/op        0 allocs/op # Baseline for input "00-oo-00TEST"
// BenchmarkTrim 100000000         11.0 ns/op         0 B/op        0 allocs/op # For input "TEST"
// BenchmarkTrim 20000000          68.4 ns/op         0 B/op        0 allocs/op # Go 1.2 for input "00-oo-00TEST"
func BenchmarkTrim(b *testing.B) {
	input := "00-oo-00TEST"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Trim(input)
	}
}

// BenchmarkPad  10000000         273 ns/op        16 B/op      2 allocs/op # Return type = string
//
// BenchmarkPad  10000000         162 ns/op         8 B/op      1 allocs/op # Changed return type to []byte, width = 6
// BenchmarkPad   5000000         435 ns/op       257 B/op      1 allocs/op # Same but width = 255
//
// BenchmarkPad  10000000         163 ns/op         8 B/op      1 allocs/op # Generalized algorithm, width = 6.
// BenchmarkPad  20000000          85.2 ns/op       8 B/op      1 allocs/op # Same but width = 2
// BenchmarkPad  10000000         170 ns/op        16 B/op      1 allocs/op # Same but width = 12
// BenchmarkPad   5000000         360 ns/op        64 B/op      1 allocs/op # Same but width = 60
// BenchmarkPad   1000000        1092 ns/op       257 B/op      1 allocs/op # Same but width = 255
//
// BenchmarkPad  20000000         109 ns/op         8 B/op      1 allocs/op # Simple for loop, width = 8
// BenchmarkPad  10000000         232 ns/op        64 B/op      1 allocs/op # Same but width = 60
//
// BenchmarkPad  20000000          95.3 ns/op         8 B/op        1 allocs/op # Go 1.2, width = 8, input = "ABC"
func BenchmarkPad(b *testing.B) {
	const width = uint8(8)
	base32 := Base32("ABC")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = base32.Pad(width)
	}
}

func ExampleEncode() {
	fmt.Println(Encode(90)) // Moderately fast. 100s ns/op; 1 alloc.
	fmt.Println(Encode(8730))
	// Output:
	// 2T
	// 8GT
}

func ExampleFromString() {

	// FromString() is pretty fast for already valid Base32 input values (10s of
	// ns per op; 0 allocs). It's pretty slow if there is lots of clean-up to do
	// (100s ns/op; 2 allocs).

	base32, _ := FromString("00-Example")
	fmt.Println(base32)
	// Output:
	// EXAMP1E
}

func ExampleBase32_Decode() {

	example := Base32("2T")
	decimal, err := example.Decode() // Fast. 10s of ns/op; 0 allocs

	if err != nil {
		fmt.Println("Unable to decode base 32 example value.")
		return
	}
	fmt.Println(decimal)
	// Output:
	// 90
}

func ExampleBase32_IsValid() {

	input := uint32(12)
	base32 := Encode(input)
	check := GenerateCheck(input)

	fmt.Println(base32.IsValid(check))
	// Output:
	// true
}

func ExampleBase32_Pad() {
	var base32 Base32

	// Padding is pretty slow; 100s of nanoseconds per op; requires 1 alloc.

	base32 = Base32("A")
	fmt.Println(string(base32.Pad(5)))

	base32 = Base32("ABCD")
	fmt.Println(string(base32.Pad(3)))
	// Output:
	// 0000A
	// ABCD
}

func ExampleTrim() {
	fmt.Println(Trim("00A")) // Trim is pretty fast by default. 0 allocs.
	fmt.Println(Trim("A"))   // No trims needed, Trim() is very fast.
	// Output:
	// A
	// A
}
