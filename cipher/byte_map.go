package cipher

import (
	"math/rand"
	"time"
)

var _ Cipher = (*ByteMapCipher)(nil)

const tableLength = 256

type ByteMapCipher struct {
	table         map[byte]byte
	reversedTable map[byte]byte
}

func NewByteMapCipher() *ByteMapCipher {
	m1, m2 := randomTable()
	return &ByteMapCipher{m1, m2}
}

func (c *ByteMapCipher) Encrypt(bs []byte) ([]byte, error) {
	res := make([]byte, len(bs))
	for idx, b := range bs {
		res[idx] = c.table[b]
	}
	return res, nil
}

func (c *ByteMapCipher) Decrypt(bs []byte) ([]byte, error) {
	res := make([]byte, len(bs))
	for idx, b := range bs {
		res[idx] = c.reversedTable[b]
	}
	return res, nil
}

func randomTable() (map[byte]byte, map[byte]byte) {
	table := make([]byte, tableLength)
	for i := 0; i < tableLength; i++ {
		table[i] = byte(i)
	}

	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(table), func(i, j int) {
		table[i], table[j] = table[j], table[i]
	})

	m1 := make(map[byte]byte, tableLength)
	m2 := make(map[byte]byte, tableLength)
	for i := 0; i < tableLength; i++ {
		m1[byte(i)] = table[i]
		m2[table[i]] = byte(i)
	}
	return m1, m2
}
