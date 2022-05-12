package encypy

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
)

var AESKey = []byte("KDJDKJJFJ*LKJSD)") //这个key很重要，不能泄露

// PKCS7Padding PKCS7 填充模式
func PKCS7Padding(ciphertext []byte, blockSize int) []byte {
	var (
		padding = blockSize - len(ciphertext)%blockSize

		//Repeat() 函数把切片[]byte{byte(padding)}复制padding个， 然后合并成新的切片返回
		padText = bytes.Repeat([]byte{byte(padding)}, padding)
	)

	return append(ciphertext, padText...)
}

// PKCS7UnPadding PKCS7填充的反向操作， 删除填充的字符串
func PKCS7UnPadding(origData []byte) ([]byte, error) {
	var length = len(origData)

	if length == 0 {
		return nil, errors.New("加密字符串错误！")
	}

	//填充的字符串的长度
	unPadding := int(origData[length-1])

	//截取字符串，删除填写的字符串
	return origData[:(length - unPadding)], nil
}

// AesEcrypt AES加密操作
func AesEcrypt(origData, key []byte) ([]byte, error) {
	//创建加密算法实例
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	//获取块大小
	blockSize := block.BlockSize()
	//对数据进行填充，让数据长度满足要求
	origData = PKCS7Padding(origData, blockSize)
	//采用AES中的CBC加密模式
	blockMode := cipher.NewCBCEncrypter(block, key[:blockSize])
	crypted := make([]byte, len(origData))

	//执行加密
	blockMode.CryptBlocks(crypted, origData)

	return crypted, nil
}

// AesDecrypt AES 解密操作
func AesDecrypt(cypted, key []byte) ([]byte, error) {
	//创建加密算法实例
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	//获取块大小
	blockSize := block.BlockSize()
	//采用AES加密方法中的CBC加密模式　创建加密客户端实例
	blockMode := cipher.NewCBCDecrypter(block, key[:blockSize])
	origData := make([]byte, len(cypted))

	//解密
	blockMode.CryptBlocks(origData, cypted)

	//去除填充的字符串
	origData, err = PKCS7UnPadding(origData)
	if err != nil {
		return nil, err
	}

	return origData, err
}

// Base64EnCode base64加密
func Base64EnCode(data []byte) (string, error) {
	result, err := AesEcrypt(data, AESKey)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(result), err
}

// Base64DeCode base64解密
func Base64DeCode(str string) ([]byte, error) {
	//解密base64字符串
	strByte, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return nil, err
	}

	//AES 解密
	return AesDecrypt(strByte, AESKey)
}
