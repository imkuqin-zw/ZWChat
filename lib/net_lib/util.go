package net_lib

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"errors"
	"github.com/imkuqin-zw/ZWChat/common/logger"
	"go.uber.org/zap"
	"math/big"
)

func deriveAuthKey(shareKey []byte) []byte {
	authKey := make([]byte, 128)
	digest := sha256.Sum256(shareKey)
	copy(authKey[0:32], shareKey)
	copy(authKey[32:64], digest[:])
	copy(authKey[64:96], shareKey[:])
	copy(authKey[96:128], digest[:])
	return authKey
}

func DeriveMsgKey(shareKey []byte, buf []byte) []byte {
	authKey := deriveAuthKey(shareKey)
	MessageKey := make([]byte, len(buf)+32)
	var result []byte
	copy(MessageKey, authKey[88:120])
	copy(MessageKey[32:], buf)
	r := sha256.Sum256(MessageKey)
	result = r[8:24]
	return result
}

func DeriveAESKey(shareKey, msgKey []byte, key, iv []byte) {
	if len(shareKey) != 32 {
		panic("invalid auth key len")
	}
	if len(msgKey) != 16 {
		panic("invalid msg key len")
	}
	authKey := deriveAuthKey(shareKey)

	var src [68]byte
	copy(src[0:16], msgKey)
	copy(src[16:52], authKey[0:36])
	a := sha256.Sum256(src[:52])
	copy(src[0:36], authKey[40:76])
	copy(src[36:52], msgKey[:16])
	b := sha256.Sum256(src[:52])

	//key
	key = make([]byte, 32)
	copy(key, a[:8])
	copy(key[8:24], b[8:24])
	copy(key[24:32], a[24:32])
	//iv
	iv = make([]byte, 32)
	copy(iv, b[:8])
	copy(iv[8:24], a[8:24])
	copy(iv[24:32], b[24:32])
}

//AES CBC 模式加密
func AESCBCPadEncrypt(dst, src, key, iv []byte) ([]byte, error) {
	ciph, err := aes.NewCipher(key)
	if err != nil {
		logger.Error("AESCBCPadEncrypt NewCipher: ", zap.Error(err))
		return nil, err
	}
	bs := ciph.BlockSize()
	if len(iv) < bs {
		err = errors.New("IV length must equal block size")
		logger.Error("AESCBCPadEncrypt: ", zap.Error(err))
		return nil, err
	}

	iv16bytes := iv[:bs]
	src = PKCS5Padding(src, bs)
	if dst == nil {
		dst = make([]byte, len(src))
	}
	blockModel := cipher.NewCBCEncrypter(ciph, iv16bytes)
	blockModel.CryptBlocks(dst, src)
	return dst, nil
}

func PKCS5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

//AES CBC 模式解密
func AESCBCDecrypt(dst, src, key, iv []byte) ([]byte, error) {
	ciph, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	if len(src)%ciph.BlockSize() != 0 {
		err = errors.New("input not multiple of block size")
		logger.Error("AESCBCDecrypt: ", zap.Error(err))
		return nil, err
	}

	bs := ciph.BlockSize()

	if len(iv) < bs {
		err = errors.New("IV length must equal block size")
		logger.Error("AESCBCDecrypt: ", zap.Error(err))
		return nil, err
	}

	if dst == nil {
		dst = make([]byte, len(src))
	}

	iv16bytes := iv[:bs]

	mode := cipher.NewCBCDecrypter(ciph, iv16bytes)
	mode.CryptBlocks(dst, src)

	dst, err = PKCS5UnPadding(dst, bs)
	if err != nil {
		logger.Error("AESCBCDecrypt: ", zap.Error(err))
		return nil, err
	}
	return dst, nil
}

func PKCS5UnPadding(origData []byte, blockSize int) ([]byte, error) {
	length := len(origData)
	paddingLen := int(origData[length-1])
	if paddingLen >= length {
		logger.Debug("origData: ", zap.Int("length", length), zap.Int("blockSize", blockSize))
		return nil, errors.New("padding size error")
	}
	return origData[:length-paddingLen], nil
}

//4字节倍数的数组是否为0
func IsBytesAllZero(x []byte) bool {
	xBigInt := big.NewInt(0).SetBytes(x)
	zeroBigInt := big.NewInt(0)
	return xBigInt.Cmp(zeroBigInt) == 0
}
