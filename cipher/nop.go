package cipher

var _ Cipher = (*NopCipher)(nil)

type NopCipher struct{}

func NewNopCipher() *NopCipher {
	return &NopCipher{}
}

func (c *NopCipher) Encrypt(bs []byte) ([]byte, error) { return bs, nil }
func (c *NopCipher) Decrypt(bs []byte) ([]byte, error) { return bs, nil }
