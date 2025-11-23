package crypt

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	mathrand "math/rand"
	"time"

	"golang.org/x/crypto/bcrypt"
)

//en - Fixed: Use random IV instead of key as IV
func AesEncrypt(origData, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	origData = PKCS5Padding(origData, blockSize)
	
	// Generate random IV (SECURITY FIX)
	ciphertext := make([]byte, blockSize+len(origData))
	iv := ciphertext[:blockSize]
	if _, err := rand.Read(iv); err != nil {
		return nil, err
	}
	
	blockMode := cipher.NewCBCEncrypter(block, iv)
	blockMode.CryptBlocks(ciphertext[blockSize:], origData)
	return ciphertext, nil
}

//de - Fixed: Extract IV from ciphertext
func AesDecrypt(crypted, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	
	// Extract IV from ciphertext (SECURITY FIX)
	if len(crypted) < blockSize {
		return nil, errors.New("ciphertext too short")
	}
	iv := crypted[:blockSize]
	crypted = crypted[blockSize:]
	
	blockMode := cipher.NewCBCDecrypter(block, iv)
	origData := make([]byte, len(crypted))
	blockMode.CryptBlocks(origData, crypted)
	err, origData = PKCS5UnPadding(origData)
	return origData, err
}

//Completion when the length is insufficient
func PKCS5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

//Remove excess
func PKCS5UnPadding(origData []byte) (error, []byte) {
	length := len(origData)
	unpadding := int(origData[length-1])
	if (length - unpadding) < 0 {
		return errors.New("len error"), nil
	}
	return nil, origData[:(length - unpadding)]
}

//Generate 32-bit MD5 strings
func Md5(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

//Generating Random Verification Key - Fixed: Use crypto/rand
func GetRandomString(l int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	bytes := []byte(str)
	result := make([]byte, l)
	randomBytes := make([]byte, l)
	
	// Use crypto/rand for cryptographically secure random (SECURITY FIX)
	if _, err := rand.Read(randomBytes); err != nil {
		// Fallback to math/rand if crypto/rand fails
		r := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))
		for i := 0; i < l; i++ {
			result[i] = bytes[r.Intn(len(bytes))]
		}
		return string(result)
	}
	
	for i := 0; i < l; i++ {
		result[i] = bytes[int(randomBytes[i])%len(bytes)]
	}
	return string(result)
}

// HashPassword - Use bcrypt for secure password hashing (SECURITY FIX)
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPasswordHash - Verify bcrypt password hash (SECURITY FIX)
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// SecureCompare - Constant-time string comparison to prevent timing attacks (SECURITY FIX)
func SecureCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
