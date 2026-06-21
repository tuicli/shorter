package domain

import (
	"crypto/rand"
	"errors"
	"io"
	"math/big"
)

const CodeAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

type CodeGenerator interface {
	GenerateCode() (string, error)
}

type RandomCodeGenerator struct {
	Length int
	Reader io.Reader
}

func (g RandomCodeGenerator) GenerateCode() (string, error) {
	length := g.Length
	if length == 0 {
		length = 8
	}
	if length < 4 || length > 32 {
		return "", errors.New("code length must be between 4 and 32")
	}

	reader := g.Reader
	if reader == nil {
		reader = rand.Reader
	}

	out := make([]byte, length)
	maxIndex := big.NewInt(int64(len(CodeAlphabet)))
	for i := 0; i < length; i++ {
		n, err := rand.Int(reader, maxIndex)
		if err != nil {
			return "", err
		}
		out[i] = CodeAlphabet[n.Int64()]
	}

	return string(out), nil
}
