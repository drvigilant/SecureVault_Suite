package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

const chunkSize = 64 * 1024

func encryptFile(srcPath, destPath string, key []byte) error {
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	in, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	buf := make([]byte, chunkSize)
	for {
		n, err := in.Read(buf)
		if n > 0 {
			nonce := make([]byte, aesGCM.NonceSize())
			if _, err := rand.Read(nonce); err != nil {
				return err
			}
			ct := aesGCM.Seal(nil, nonce, buf[:n], nil)
			lenBuf := make([]byte, 4)
			binary.BigEndian.PutUint32(lenBuf, uint32(len(ct)))
			out.Write(nonce)
			out.Write(lenBuf)
			out.Write(ct)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func decryptFile(srcPath, destPath string, key []byte) error {
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	in, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	nonceSize := aesGCM.NonceSize()
	for {
		nonce := make([]byte, nonceSize)
		if _, err := io.ReadFull(in, nonce); err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		lenBuf := make([]byte, 4)
		if _, err := io.ReadFull(in, lenBuf); err != nil {
			return err
		}
		ct := make([]byte, binary.BigEndian.Uint32(lenBuf))
		if _, err := io.ReadFull(in, ct); err != nil {
			return err
		}
		plain, err := aesGCM.Open(nil, nonce, ct, nil)
		if err != nil {
			return fmt.Errorf("decryption failed: invalid key or tampered data")
		}
		out.Write(plain)
	}
	return nil
}
