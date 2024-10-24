package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	crand "crypto/rand"
	"encoding/binary"
	"encoding/json"
	"io"
	"log"
)

func encodeCommandRequest(request CommandRequest) []byte {
	bytesBuffer := new(bytes.Buffer)
	err := binary.Write(bytesBuffer, binary.BigEndian, request)
	if err != nil {
		log.Fatalln(err)
	}
	return encryptMessage(bytesBuffer.Bytes())
}

func encodeMoveRequest(request MoveRequest) []byte {
	bytesBuffer := new(bytes.Buffer)
	err := binary.Write(bytesBuffer, binary.BigEndian, request)
	if err != nil {
		log.Fatalln(err)
	}
	return encryptMessage(bytesBuffer.Bytes())
}

func decodeCommandResponse(bytesResponse []byte) CommandResponse {
	var response CommandResponse
	bytesReader := bytes.NewReader(decryptMessage(bytesResponse))
	err := binary.Read(bytesReader, binary.BigEndian, &response)
	if err != nil {
		log.Fatalln(err)
	}
	return response
}

func decodeDisplayResponse(bytesResponse []byte) DisplayResponse {
	var response DisplayResponse
	json.Unmarshal(bytesResponse, &response)
	return response
}

func encryptMessage(message []byte) []byte {
	block, err := aes.NewCipher(symmetricKey)
	if err != nil {
		log.Fatalln(err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		log.Fatalln(err)
	}

	nonce := make([]byte, 12)
	if _, err := io.ReadFull(crand.Reader, nonce); err != nil {
		log.Fatalln(err)
	}
	encrypted := gcm.Seal(nonce, nonce, message, nil)
	return encrypted
}

func decryptMessage(message []byte) []byte {
	block, err := aes.NewCipher(symmetricKey)
	if err != nil {
		log.Fatalln(err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		log.Fatalln(err)
	}

	nonceSize := gcm.NonceSize()
	nonce, message := message[:nonceSize], message[nonceSize:]
	decryptedMessage, err := gcm.Open(nil, nonce, message, nil)
	if err != nil {
		log.Fatalln(err)
	}
	return decryptedMessage
}
