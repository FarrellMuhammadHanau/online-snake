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

func decodeCommandRequest(bytesCommand []byte, key []byte) (*CommandRequest, error) {
	var command CommandRequest
	message, err := decryptMessage(bytesCommand, key)
	if err != nil {
		return nil, err
	}
	bytesReader := bytes.NewReader(message)
	binary.Read(bytesReader, binary.BigEndian, &command)
	return &command, nil
}

func decodeMove(bytesMoves []byte, key []byte) (*MoveRequest, error) {
	var move MoveRequest
	message, err := decryptMessage(bytesMoves, key)
	if err != nil {
		return nil, err
	}
	bytesReader := bytes.NewReader(message)
	binary.Read(bytesReader, binary.BigEndian, &move)
	return &move, nil
}

func encodeCommandResponse(response CommandResponse, key []byte) []byte {
	buffer := new(bytes.Buffer)
	binary.Write(buffer, binary.BigEndian, response)
	return encryptMessage(buffer.Bytes(), key)
}

func encryptMessage(message []byte, key []byte) []byte {
	block, err := aes.NewCipher(key)
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

func decryptMessage(message []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	nonce, message := message[:nonceSize], message[nonceSize:]
	decryptedMessage, err := gcm.Open(nil, nonce, message, nil)
	if err != nil {
		return nil, err
	}
	return decryptedMessage, nil
}

func encodeDisplayResponse(response DisplayResponse) []byte {
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		log.Fatalln(err)
	}
	return jsonResponse
}
