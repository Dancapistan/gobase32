A Base32-to-Decimal Converter for Go
====================================

A Go (golang) implementation of
[Crockford-style Base32 numbers](http://www.crockford.com/wrmg/base32.html).

Provides Encode and Decode functions and some useful utility functions. See the
godoc output for API details and examples.

Examples
--------

You can encode a decimal value using the `Encode` function:

    b32 := gobase32.Encode(90) //=> "2T"

You can decode a Base32 value using the `Decode` method:

    i, err := b32.Decode() //=> 90, nil

The `Encode` and `Decode` functions are generally pretty fast and resistant 
to common input errors, like using the letter `O` in place of the numeral 
`0`. However, speed and memeory efficiency were priorities, so use the 
`FromString` function for a more robust and spec-compliant method of conversion.

You can generate a checksum using the `GenerateCheck` function, and use the 
checksum to check the validity of an existing Base32 value:

    c := gobase32.GenerateCheck(90)
    valid := b32.IsValid(c) //=> true

Use the `FromString` function to normalize raw user input:

    b32, err := gobase32.FromString(rawInputString)

The godoc output has examples for all of the API functions.
