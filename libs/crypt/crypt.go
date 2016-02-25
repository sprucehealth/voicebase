package crypt

// Encrypter provides an interface for mchanisms that provide encryption
type Encrypter interface {
	Encrypt(d []byte) ([]byte, error)
}

// PlaintextEncrypter is a convenience no-op encryptor
type PlaintextEncrypter struct{}

// Encrypt performs a no-op
func (e *PlaintextEncrypter) Encrypt(d []byte) ([]byte, error) { return d, nil }

// Decrypter provides an interface for mchanisms that provide decryption
type Decrypter interface {
	Decrypt(d []byte) ([]byte, error)
}

// PlaintextDecrypter is a convenience no-op decryptor
type PlaintextDecrypter struct{}

// Decrypt performs a no-op
func (e *PlaintextDecrypter) Decrypt(d []byte) ([]byte, error) { return d, nil }
