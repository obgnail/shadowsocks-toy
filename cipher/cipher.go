package cipher

type Cipher interface {
	Encrypt(bs []byte) ([]byte, error)
	Decrypt(bs []byte) ([]byte, error)
}
