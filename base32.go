// Package base32 provides Crockford-style translation to and from a Base-32
// notation for unsigned integers. Read http://www.crockford.com/wrmg/base32.html
// for more details.
//
// This is not a base-32 ENCODING as you would find in the encoding/base32
// package in the Go standard library. That package encodes arbitrary bytes.
// This package translates base 10 unsigned integers into a base 32 unsigned
// integer.
//
// Limitations and TODOs: This library can't handle hyphens in the encoded value
// (although see FromString). This library has only been tested on a 64-bit
// little-endian machine. Speed of encoding and decoding was a top priority over
// feature completeness and flexibility. Pull requests (with tests and
// benchmarks) welcome.
//
package base32

import (
	"errors"
)

const PackageVersion string = "0.0.3"

// A base-32 number, encoded as a UTF-8/ASCII string.
//
// Valid Base32 digits include:
//
//   0123456789ABCDEFGHJKMNPQRSTVWXYZ
//
// Error corrections are:
//
//   1. Lowercase letters are uppercased.
//   2. Letter 'O' is converted to numeral '0'.
//   3. Letter 'L' is converted to numeral '1'.
//   4. Letter 'I' is converted to numeral '1'.
//   5. Letter 'U' is never a valid Base32 digit.
//
type Base32 string

// A checksum for a base-32 number.
//
// From the spec:
// 		An application may append a check symbol to a symbol string. This check
// 		symbol can be used to detect wrong-symbol and transposed-symbol errors.
// 		This allows for detecting transmission and entry errors early and
// 		inexpensively.
//
// Valid Check digits include all of the valid Base32 digits (see type Base32
// for details) in addition to:
//
//   *~$=U
//
// Error corrections are the same as for Base32, except #5 is removed since the
// letter 'U' is a valid Check digit.
//
type Check rune

// Maximum possible value of the uintXX data types.
const maxUint32Value uint32 = 4294967295
const maxUint64Value uint64 = 18446744073709551615

// The maximum possible value of a Base32 number of a given number of digits.
// For example, the maximum 1-digit Base32 value is "Z", which translates to 31.
//
// See Max7DigitBase32 for more details.
const (
	Max1DigitInt uint32 = 1<<(5*(iota+1)) - 1 // 31
	Max2DigitInt                              // 1023
	Max3DigitInt                              // etc ..
	Max4DigitInt
	Max5DigitInt
	Max6DigitInt
	Max7DigitInt = maxUint32Value
)

// The maximum Base32 value that will fit in a uint32 integer.
//
// Each Base32 digit is 5 bits. Therefore, a uint32 will fit 6 full base32
// digits with two bits left over.
const Max7DigitBase32 Base32 = "3ZZZZZZ"

// An opaque, invalid Base32 value.
const InvalidBase32Value Base32 = ""

// An opaque, invalid checksum value.
const InvalidCheckValue Check = 0

// Encode translates a base-10 number into a base-32 string.
//
// Performance note: fairly fast. 1 memory allocation.
func Encode(num uint32) Base32 {

	// To store the raw result.
	var buffer [7]byte

	const fiveOnes uint32 = 31 // Binary 11111

	// Break the argument into 5-bit bytes, big-end first. Each base 32 digit
	// encodes 5 bits of information. There are 6 5-bit bytes plus 2 bits in a
	// 32 bit unsigned int.
	var bytes = [7]uint8{
		uint8(num >> 30 & fiveOnes), // The >> operator zero-pads the left of the result.
		uint8(num >> 25 & fiveOnes),
		uint8(num >> 20 & fiveOnes),
		uint8(num >> 15 & fiveOnes),
		uint8(num >> 10 & fiveOnes),
		uint8(num >> 5 & fiveOnes),
		uint8(num >> 0 & fiveOnes),
	}

	// We don't want the base-32 result to be zero-padded, so we'll ignore
	// everything up to the first non-zero value. However, special case: if the
	// input argument is 0, then the result should be "0".
	var firstNonZeroIndex int = 6

	// Encode each of the 5-bit bytes into the corresponding base-32 rune.
	for i, byte := range bytes {
		buffer[i] = encodingValue[byte]
		if byte != 0 && firstNonZeroIndex == 6 {
			// Keep track of the index of the first non-zero byte so we can
			// slice off the leading zeros at the end.
			firstNonZeroIndex = i
		}
	}

	// Slice off the leading zeros, and convert the buffer into a Base32-type
	// string.
	return Base32(buffer[firstNonZeroIndex:])
}

// FromString converts a base32-like string into a valid Base32 value, if
// possible. It normalizes the characters (lowercase to uppercase, convert O to
// 0, removes hyphens). It can't handle otherwise invalid base-32 values,
// though, and will return an error.
//
// Performance note: This function is very fast for already-valid Base32 Values,
// and for totally invalid values. 0 memory allocations. Only when the input is
// technically valid but totally non-normalized does this method get crazy (~2
// allocations).
//
func FromString(base32String string) (Base32, error) {

	var inputLength = len(base32String)

	if inputLength == 0 {
		return InvalidBase32Value, decodeEmptyString
	}

	// First, check the string to see if it is already a valid Base32 value.
	var standard bool = true
	for _, byte := range base32String {
		isNumber := byte >= '0' && byte <= '9'
		isValidUpper := byte >= 'A' && byte <= 'Z' &&
			!(byte == 'I' || byte == 'O' || byte == 'L' || byte == 'U')
		if !isNumber && !isValidUpper {
			standard = false
			break
		}
	}

	// If it already looks fine; nothing to do.
	if standard && base32String[0] != '0' {
		return Base32(base32String), nil
	}

	// Check for invalid characters.
	for _, rune := range base32String {

		isNumber := rune >= '0' && rune <= '9'
		isUpper := rune >= 'A' && rune <= 'Z' && rune != 'U'
		isLower := rune >= 'a' && rune <= 'z' && rune != 'u'
		isHyphen := rune == '-'

		isValid := isNumber || isUpper || isLower || isHyphen

		if !isValid {
			return InvalidBase32Value, decodeInvalidDigit
		}
	}

	// Find the first non-zero character so we can trim off any zero padding.
	firstNonZeroCharIndex := 0
	for i, char := range base32String {
		isZero := char == '0' || char == 'o' || char == 'O'
		isHyphen := char == '-'
		if !isZero && !isHyphen {
			firstNonZeroCharIndex = i
			break
		}
	}

	// Count all hyphens in the string that occur AFTER the first non-zero
	// character. These will have to be deleted later on.
	interiorHyphenCount := 0
	for i := firstNonZeroCharIndex + 1; i < inputLength; i++ {
		if base32String[i] == '-' {
			interiorHyphenCount++
		}
	}

	// Mutate the characters in the result string into normalized digits. For
	// example, convert lowercase letters into uppercase, etc.

	var lenResult = inputLength - firstNonZeroCharIndex - interiorHyphenCount
	var result = make([]byte, lenResult)
	var inputIndex = firstNonZeroCharIndex
	var destIndex = 0

	for inputIndex < inputLength {

		char := base32String[inputIndex]
		inputIndex++

		// Convert letter O to numeral 0.
		if char == 'o' || char == 'O' {
			result[destIndex] = '0'
			destIndex++
			continue
		}

		// Convert letters L and I into numeral 1.
		if char == 'l' || char == 'L' || char == 'i' || char == 'I' {
			result[destIndex] = '1'
			destIndex++
			continue
		}

		// Uppercase the characters, ASCII hack.
		if char >= 'a' && char <= 'z' {
			result[destIndex] = char - 32
			destIndex++
			continue
		}

		if char == '-' {
			// Skip hyphen.
			continue
		}

		result[destIndex] = char
		destIndex++
	}

	return Base32(result), nil
}

var (
	decodeEmptyString  error = errors.New("Cannot decode empty Base32 string")
	decodeTooBig32     error = errors.New("Base 32 value is too big for a 32-bit unsigned integer")
	decodeInvalidDigit error = errors.New("Invalid Base32 digit")
)

// Decode translates a base-32 number into a base-10 integer. The letter values
// in the supplied string are case insensitive. This function is robust against
// common errors in the input, by design; for example 1, I, and l are assumed to
// be the same character: 1. Same with O and 0.
//
// An error will be returned if there was a fatal problem decoding the value.
// Possible decode errors are:
//
// - The empty string Base32("") is an invalid value and distinct from
// Base32("0").
//
// - The base-32 value is too big for the uint32 datatype. See WillFit() for
// details.
//
// - The base-32 string has invalid digits.
//
// Performance: This method is quite fast and does 0 allocations.
//
func (num Base32) Decode() (result uint32, err error) {

	// num can be any number of digits, at least 1 digit. Don't assume a fixed
	// number of digits.

	// `shift` is the number of 5-bit bytes we want to move the value over. Since
	// we're starting at the most significant digit, we'll start at the biggest
	// shift value and work down.
	var shift = (len(num) - 1) * 5

	if shift < 0 {
		err = decodeEmptyString
		return
	}

	if !num.WillFit() {
		err = decodeTooBig32
		return
	}

	// For each base-32 character, convert that into its decoding bits
	// and add it to the result.
	var width = uint(shift)
	for _, rn := range num {

		// Check for invalid rune. This is only half a check. We check to make
		// sure the rune is not too big, or else it will cause an array index
		// out of bounds error when we get the decodingValue.
		if rn > decodeMaxRune || rn < decodeMinRune {
			err = decodeInvalidDigit
			return
		}

		// Convert the character into its byte value.
		val := decodingValue[rn]

		// Second half of the valid rune check. An invalid rune will return a
		// value of invalidDecodeValue.
		//
		if val == invalidDecodeValue {
			err = decodeInvalidDigit
			return
		}

		// Add it to the result.
		result = result | (val << width)

		// Move on to the next 5-bit byte.
		width -= 5
	}

	return
}

// IsValid checks a base 32 number against a checksum.
func (num Base32) IsValid(check Check) bool {
	var base10 uint32
	var err error
	var validCheck Check

	base10, err = num.Decode()
	validCheck = GenerateCheck(base10)

	return err == nil && check == validCheck
}

// String implements the Stringer interface for Base32 types.
func (num Base32) String() string {
	if num == InvalidBase32Value {
		return "<invalid>"
	}
	return string(num)
}

// String implements the Stringer interface for Check types.
func (check Check) String() string {
	return string(check)
}

// Pad left-pads the argument with 0s until the resulting string is at least `n`
// characters wide.
//
// See also Trim() for the opposite function.
//
// The input value must be valid or the result of this method is undefined.
func (num Base32) Pad(n uint8) []byte {
	finalWidth := int(n)
	inputLength := len(num)

	// If we're already at least n characters wide, nothing to do here.
	if inputLength >= finalWidth {
		return []byte(num)
	}

	// start is where the base32 digits start in the result's byte slice.
	var start = int(finalWidth - inputLength)
	var result = make([]byte, finalWidth)

	for i := 0; i < finalWidth; i++ {
		if i < start {
			result[i] = '0'
		} else {
			result[i] = num[i-start]
		}
	}

	return result
}

// Trim removes zeros from the beginning of the argument and returns the
// result as a Base32 value.
//
// See also Pad() for the opposite function.
//
// The input value must be an otherwise valid Base32 value, or else the result
// of this function is undefined. (This function does treat the letters
// 'o' and 'O' and the hyphen as zeros.)
func Trim(padded string) Base32 {
	firstNonZeroIdx := 0
	for i, char := range padded {
		var isZero = char == '0' || char == 'o' || char == 'O' || char == '-'
		if !isZero {
			firstNonZeroIdx = i
			break
		}
	}
	return Base32(padded[firstNonZeroIdx:])
}

// WillFit returns true if the Base32 value can be decoded into a uint32
// integer, or false if the value is too big for a uint32 integer.
//
// It is assumed `num` is valid. If not, the behavior of this method is
// undefined. Also, `num` should not be left-padded with zeros.
func (num Base32) WillFit() bool {

	var numDigits = len(num)
	// len() returns number of bytes, but that is the same as the number of
	// characters for our use-case since all possible (legal) values are
	// 7-bit ASCII compliant.

	// Any six digit Base32 value will fit for sure.
	if numDigits < 7 {
		return true
	}

	// Any Base32 value with more than 7 digits definitely cannot fit into a
	// uint32.
	if numDigits > 7 {
		return false
	}

	// A 7-digit Base32 value will fit if the most significant digit is 3 or
	// under.
	var msd = num[0]
	return msd == '3' || msd == '2' || msd == '1' || msd == '0'
}

// GenerateCheck returns the checksum byte for a given argument. It will be one
// of 0-9, the valid Base32 values of A-Z, or *, ~, $, =, or U.
func GenerateCheck(num uint32) Check {
	const checksumPrime = 37
	return Check(encodingValue[num%checksumPrime])
}

var (
	invalidCheckLength = errors.New("A check string must be exactly 1 character long")
	invalidCheckDigit  = errors.New("The input value is not a valid checksum digit")
)

// CheckFromString converts the input string into a valid Check value if possible.
//
// This function is robust by design against common input errors, like the
// letter 'O' in place of the numeral '0'.
//
// Possible failure cases are:
//
// - The input string must be exactly 1 character long to be a valid Check value.
//
// - The input character must be a valid Check value. See type Check for a
// list of valid Check digits and corresponding error corrections.
//
// TODO Add tests
func CheckFromString(input string) (result Check, err error) {

	if len(input) != 1 {
		return InvalidCheckValue, invalidCheckLength
	}

	char := rune(input[0])
	validBase32Digit := validBase32Digit[char]
	validChecksumDigit := char == '*' || char == '~' || char == '$' || char == '=' || char == 'u' || char == 'U'

	if !validBase32Digit && !validChecksumDigit {
		return InvalidCheckValue, invalidCheckDigit
	}

	// Capitalize the value if needed. ASCII hack.
	if char >= 'a' && char <= 'z' {
		char = char - 32
	}

	// Normalize common error values
	if char == 'O' {
		char = '0'
	} else if char == 'I' || char == 'L' {
		char = '1'
	}

	return Check(char), nil
}

var encodingValue = [...]byte{
	'0',
	'1',
	'2',
	'3',
	'4',
	'5',
	'6',
	'7',
	'8',
	'9',
	'A',
	'B',
	'C',
	'D',
	'E',
	'F',
	'G',
	'H',
	'J',
	'K',
	'M',
	'N',
	'P',
	'Q',
	'R',
	'S',
	'T',
	'V',
	'W',
	'X',
	'Y',
	'Z',
	'*', // ONLY USED FOR CHECKSUM
	'~', // ONLY USED FOR CHECKSUM
	'$', // ONLY USED FOR CHECKSUM
	'=', // ONLY USED FOR CHECKSUM
	'U', // ONLY USED FOR CHECKSUM
}

// Todo: Delete this map and remove uses of it.
var validBase32Digit = map[rune]bool{
	'0': true,
	'1': true,
	'2': true,
	'3': true,
	'4': true,
	'5': true,
	'6': true,
	'7': true,
	'8': true,
	'9': true,
	'A': true,
	'B': true,
	'C': true,
	'D': true,
	'E': true,
	'F': true,
	'G': true,
	'H': true,
	'I': true,
	'J': true,
	'K': true,
	'L': true,
	'M': true,
	'N': true,
	'O': true,
	'P': true,
	'Q': true,
	'R': true,
	'S': true,
	'T': true,
	'V': true,
	'W': true,
	'X': true,
	'Y': true,
	'Z': true,
	'a': true,
	'b': true,
	'c': true,
	'd': true,
	'e': true,
	'f': true,
	'g': true,
	'h': true,
	'i': true,
	'j': true,
	'k': true,
	'l': true,
	'm': true,
	'n': true,
	'o': true,
	'p': true,
	'q': true,
	'r': true,
	's': true,
	't': true,
	'v': true,
	'w': true,
	'x': true,
	'y': true,
	'z': true,
}

const decodeMaxRune = 'z'
const decodeMinRune = '0'
const invalidDecodeValue = 99 // 31 is the maximum valid value

var decodingValue = [...]uint32{
	'0': 0, // 48 = 0x30
	'1': 1,
	'2': 2,
	'3': 3,
	'4': 4,
	'5': 5,
	'6': 6,
	'7': 7,
	'8': 8,
	'9': 9, // 57 = 0x39
	58:  invalidDecodeValue,
	59:  invalidDecodeValue,
	60:  invalidDecodeValue,
	61:  invalidDecodeValue,
	62:  invalidDecodeValue,
	63:  invalidDecodeValue,
	64:  invalidDecodeValue,
	'A': 10, // 65 = 0x40
	'B': 11,
	'C': 12,
	'D': 13,
	'E': 14,
	'F': 15,
	'G': 16,
	'H': 17,
	'I': 1,
	'J': 18,
	'K': 19,
	'L': 1,
	'M': 20,
	'N': 21,
	'O': 0,
	'P': 22,
	'Q': 23,
	'R': 24,
	'S': 25,
	'T': 26,
	'U': invalidDecodeValue,
	'V': 27,
	'W': 28,
	'X': 29,
	'Y': 30,
	'Z': 31, // 90 = 0x5A
	91:  invalidDecodeValue,
	92:  invalidDecodeValue,
	93:  invalidDecodeValue,
	94:  invalidDecodeValue,
	95:  invalidDecodeValue,
	96:  invalidDecodeValue,
	'a': 10, // 97 = 0x61
	'b': 11,
	'c': 12,
	'd': 13,
	'e': 14,
	'f': 15,
	'g': 16,
	'h': 17,
	'i': 1,
	'j': 18,
	'k': 19,
	'l': 1,
	'm': 20,
	'n': 21,
	'o': 0,
	'p': 22,
	'q': 23,
	'r': 24,
	's': 25,
	't': 26,
	'u': invalidDecodeValue,
	'v': 27,
	'w': 28,
	'x': 29,
	'y': 30,
	'z': 31,
}
