package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
)

func GenerateSalt() ([]byte, error) {
	salt := make([]byte, 16)
	_, err := rand.Read(salt)
	return salt, err
}

// Функция для шифрования данных с AES-256
func encryptAES256(plaintext, key []byte) (string, error) {
	// Создаем блочный шифр
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// Генерируем случайный IV
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	// Шифруем данные
	//TODO use AEAD instead NewCtr. Need to read more about it
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)

	// Кодируем в base64 для хранения в JSON
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func decryptAES256(encryptedData string, key []byte) (string, error) {
	// Декодируем base64
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return "", err
	}

	// Проверяем длину данных
	if len(ciphertext) < aes.BlockSize {
		return "", fmt.Errorf("зашифрованные данные слишком короткие")
	}

	// Создаем блочный шифр
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// Извлекаем IV и данные
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	// Расшифровываем данные
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)

	return string(ciphertext), nil
}

func addEntrie() {}

// Функция для генерации ключа из мастер-пароля
func GenerateKey(masterPassword string, salt []byte) []byte {
	// time=1, memory=64MB, threads=4, keyLen=32
	return argon2.IDKey([]byte(masterPassword), salt, 1, 64*1024, 4, 32)
}
