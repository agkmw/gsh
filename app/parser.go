package main

import (
	"bytes"
	"slices"
	"strings"
)

func tokenizeInput(input string) []string {
	tokens := make([]string, 0)
	var tokenBuffer bytes.Buffer
	inDoubleQuote := false
	inSingleQuote := false

	input = strings.TrimSpace(input)
	for i := 0; i < len(input); i++ {
		c := input[i]
		switch c {
		case '\\':
			if !inSingleQuote && !inDoubleQuote {
				i++
				tokenBuffer.WriteByte(input[i])
			} else if inDoubleQuote {
				specialChars := []byte{'"', '\\', '$', '`'}
				if slices.Contains(specialChars, input[i+1]) {
					i++
				}
				tokenBuffer.WriteByte(input[i])
			} else {
				tokenBuffer.WriteByte(c)
			}

		case '"':
			if inSingleQuote {
				tokenBuffer.WriteByte(c)
			} else {
				inDoubleQuote = !inDoubleQuote
			}

		case '\'':
			if inDoubleQuote {
				tokenBuffer.WriteByte(c)
			} else {
				inSingleQuote = !inSingleQuote
			}

		case ' ':
			if !inSingleQuote && !inDoubleQuote {
				if tokenBuffer.Len() > 0 {
					tokens = append(tokens, tokenBuffer.String())
					tokenBuffer.Reset()
				}
			} else {
				tokenBuffer.WriteByte(c)
			}

		default:
			tokenBuffer.WriteByte(c)
		}

		if i == len(input)-1 && tokenBuffer.Len() > 0 {
			tokens = append(tokens, tokenBuffer.String())
			tokenBuffer.Reset()
		}
	}

	if tokenBuffer.Len() > 0 {
		tokens = append(tokens, tokenBuffer.String())
		tokenBuffer.Reset()
	}

	return tokens
}
