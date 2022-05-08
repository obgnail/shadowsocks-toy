package cipher

import "encoding/base64"

var _ Cipher = (*Base64Cipher)(nil)

type Base64Cipher struct{}

func NewBase64Cipher() *Base64Cipher {
	return &Base64Cipher{}
}

func (c *Base64Cipher) Encrypt(bs []byte) ([]byte, error) {
	return []byte(base64.RawStdEncoding.EncodeToString(bs)), nil
}
func (c *Base64Cipher) Decrypt(bs []byte) ([]byte, error) {
	return base64.RawStdEncoding.DecodeString(string(bs))
}
