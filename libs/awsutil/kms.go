package awsutil

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"io"
	"sync"

	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/kms/kmsiface"
	"github.com/sprucehealth/backend/libs/crypt"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/ptr"
)

// KMSEncryptedData provides a struct wrapper
type kmsEncryptionWrapper struct {
	CipherSpec string `json:"cipher_spec"`
	KeyID      string `json:"key_id"`
	CipherKey  []byte `json:"cipher_key"`
	Nonce      []byte `json:"nonce"`
	Data       []byte `json:"data"`
}

type kmsEncrypter struct {
	// The aead used to encrypt data
	aead cipher.AEAD
	// The unique identifier that maps to the encryption key backing the cipher
	keyID string
	// The encryption key encrypted with the remote master key
	// The purpose of this key is to attach it alongside the encrypted data
	// The receiver utilizes the keyID to decrypt the cipher key which can in turn
	//   be used to decrypt the wrapped data
	cipherKey []byte

	kmsC kmsiface.KMSAPI
}

// Why not 256? 128 is fine for us/99% of use cases.
const keySpec = "AES_128"

// NewKMSEncrypter returns an initialized instance of kmsEncrypter
// TODO: Decouple this from the kinda hacky crypt package
func NewKMSEncrypter(masterKeyARN string, kmsC kmsiface.KMSAPI) (crypt.Encrypter, error) {
	// Generate a data key for us to use: http://docs.aws.amazon.com/kms/latest/APIReference/API_GenerateDataKey.html
	// TODO: There are lots of clever KMS attributes we could use
	resp, err := kmsC.GenerateDataKey(&kms.GenerateDataKeyInput{
		KeyId:   ptr.String(masterKeyARN),
		KeySpec: ptr.String(keySpec),
	})
	if err != nil {
		return nil, errors.Trace(err)
	}
	cipherBlock, err := aes.NewCipher(resp.Plaintext)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(cipherBlock)
	if err != nil {
		return nil, err
	}
	return &kmsEncrypter{
		aead:      aead,
		keyID:     *resp.KeyId,
		cipherKey: resp.CiphertextBlob,
		kmsC:      kmsC,
	}, nil
}

// Encrypt performs 128 bit AES encryption on the provided data
// Shamelessly used as reference: http://stackoverflow.com/questions/18817336/golang-encrypting-a-string-with-aes-and-base64
func (k *kmsEncrypter) Encrypt(d []byte) ([]byte, error) {
	ed, nonce, err := k.encrypt(d)
	if err != nil {
		return nil, err
	}
	wd, err := k.wrap(ed, nonce)
	if err != nil {
		return nil, err
	}

	return wd, nil
}

func (k *kmsEncrypter) wrap(d, nonce []byte) ([]byte, error) {
	wd, err := json.Marshal(&kmsEncryptionWrapper{
		CipherSpec: keySpec,
		CipherKey:  k.cipherKey,
		Nonce:      nonce,
		KeyID:      k.keyID,
		Data:       d,
	})
	if err != nil {
		return nil, err
	}
	return wd, nil
}

func (k *kmsEncrypter) encrypt(d []byte) ([]byte, []byte, error) {
	nonce := make([]byte, k.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}
	cipherText := make([]byte, 0, k.aead.Overhead()+len(d))
	return k.aead.Seal(cipherText, nonce, d, nil), nonce, nil
}

type kmsDecrypter struct {
	masterKeyARN string
	keyCache     map[string][]byte
	cacheMutex   sync.RWMutex
	kmsC         kmsiface.KMSAPI
}

// NewKMSDecrypter returns an initialized instance of kmsDeryptor
// TODO: Decouple this from the kinda hacky crypt package
func NewKMSDecrypter(masterKeyARN string, kmsC kmsiface.KMSAPI) crypt.Decrypter {
	// TODO: We should check the key ARN somehow here to fail fast in the event of bad input
	return &kmsDecrypter{
		masterKeyARN: masterKeyARN,
		keyCache:     make(map[string][]byte),
		kmsC:         kmsC,
	}
}

// Decrypt performs decryption of messaged properly wrapped in kmsEncryptionWrapper
func (k *kmsDecrypter) Decrypt(d []byte) ([]byte, error) {
	wd, err := k.unwrap(d)
	if err != nil {
		return nil, err
	}
	key, err := k.key(wd.KeyID, wd.CipherKey)
	if err != nil {
		return nil, err
	}
	dd, err := k.decrypt(key, wd.Data, wd.Nonce)
	if err != nil {
		return nil, err
	}
	return dd, nil
}

func (k *kmsDecrypter) unwrap(d []byte) (*kmsEncryptionWrapper, error) {
	w := &kmsEncryptionWrapper{}
	if err := json.Unmarshal(d, w); err != nil {
		return nil, err
	}
	return w, nil
}

func (k *kmsDecrypter) key(keyID string, cipherKey []byte) ([]byte, error) {
	k.cacheMutex.RLock()
	key, ok := k.keyCache[keyID]
	k.cacheMutex.RUnlock()

	// If we don't have the key caches, use KMS to decrypt the cipher key
	if !ok {
		resp, err := k.kmsC.Decrypt(&kms.DecryptInput{
			CiphertextBlob: cipherKey,
		})
		if err != nil {
			return nil, err
		}
		k.cacheMutex.Lock()
		defer k.cacheMutex.Unlock()
		k.keyCache[*resp.KeyId] = resp.Plaintext
		key = resp.Plaintext
	}
	return key, nil
}

// Shameless: http://stackoverflow.com/questions/18817336/golang-encrypting-a-string-with-aes-and-base64
func (k *kmsDecrypter) decrypt(key, d, nonce []byte) ([]byte, error) {
	cipherBlock, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(cipherBlock)
	if err != nil {
		return nil, err
	}
	return aead.Open(nil, nonce, d, nil)
}
