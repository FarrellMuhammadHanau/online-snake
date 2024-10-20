package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	crand "crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/binary"
	"io"
	"log"
	"math/rand"
	"net"
	"strings"
)

const (
	SERVER_IP   = ""
	UDP_PORT    = "1566"
	TCP_PORT    = "1567"
	UDP         = "udp4"
	TCP         = "tcp4"
	BUFFER_SIZE = 2048
	MAX_ROOMS   = 10
)

type CommandRequest struct {
	UserID     uint32
	JoinRoom   bool
	RoomID     uint8
	ExitRoom   bool
	Quit       bool
	Username   [5]rune
	SnakeShape rune
}

type CommandResponse struct {
	IsSuccess bool
	ExitRoom  bool
	JoinRoom  bool
	Quit      bool
}

type MoveRequest struct {
	UserID uint32
	Move   rune
}

type User struct {
	ID         uint32
	RoomID     uint8
	UdpAddress *net.UDPAddr
}

var Users map[uint32]*User
var Rooms map[uint8]*Room
var socketUDP *net.UDPConn
var symmetricKeys map[string][]byte

func main() {
	Users = make(map[uint32]*User)
	Rooms = make(map[uint8]*Room)
	symmetricKeys = make(map[string][]byte)

	// Create UDP Listener
	udpListenAddress, err := net.ResolveUDPAddr(UDP, net.JoinHostPort(SERVER_IP, UDP_PORT))
	if err != nil {
		log.Fatalln(err)
	}
	socketUDP, err = net.ListenUDP(UDP, udpListenAddress)
	if err != nil {
		log.Fatalln(err)
	}
	defer socketUDP.Close()

	go readUDP(socketUDP)

	// Create TCP LIstener
	tcpListenAddress, err := net.ResolveTCPAddr(TCP, net.JoinHostPort(SERVER_IP, TCP_PORT))
	if err != nil {
		log.Fatalln(err)
	}
	socketTCP, err := net.ListenTCP(TCP, tcpListenAddress)
	if err != nil {
		log.Fatalln(err)
	}
	defer socketTCP.Close()

	for {
		connection, _ := socketTCP.AcceptTCP()
		go readTCP(connection)
	}
}

func readUDP(conn *net.UDPConn) {
	for {
		receiveBuffer := make([]byte, BUFFER_SIZE)
		receiveLength, udpAddr, _ := conn.ReadFromUDP(receiveBuffer)
		go func(recBuffer []byte, addr *net.UDPAddr) {
			key := symmetricKeys[udpAddr.String()]
			move := decodeMove(recBuffer, key)
			room := Rooms[Users[move.UserID].RoomID]
			room.mainChannel <- move
		}(receiveBuffer[:receiveLength], udpAddr)
	}
}

func readTCP(conn *net.TCPConn) {
	defer conn.Close()

	user := User{}

	// Give public key to client
	privateKey, err := rsa.GenerateKey(crand.Reader, 2048)
	if err != nil {
		log.Fatalln(err)
	}
	publicKey := &privateKey.PublicKey
	publicKeyByte, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		log.Fatalln(err)
	}
	conn.Write(publicKeyByte)

	// Get symmetric key from client
	sKeyBuffer := make([]byte, BUFFER_SIZE)
	sKeyBuffLength, _ := conn.Read(sKeyBuffer)
	symmetricKey := sKeyBuffer[:sKeyBuffLength]
	symmetricKey, err = rsa.DecryptOAEP(sha256.New(), crand.Reader, privateKey, symmetricKey, nil)
	if err != nil {
		log.Fatalln(err)
	}

	for {
		user.ID = rand.Uint32()

		_, exist := Users[user.ID]

		if !exist {
			Users[user.ID] = &user
			break
		}
	}

	// Send ID
	bytesID := make([]byte, 4)
	binary.BigEndian.PutUint32(bytesID, user.ID)
	conn.Write(encryptMessage(bytesID, symmetricKey))

	// Get UDP Address
	udpAddrBuffer := make([]byte, BUFFER_SIZE)
	length, _ := conn.Read(udpAddrBuffer)
	user.UdpAddress, _ = net.ResolveUDPAddr(UDP, string(decryptMessage(udpAddrBuffer[:length], symmetricKey)))

	symmetricKeys[user.UdpAddress.String()] = symmetricKey

	for {
		receiveBuffer := make([]byte, BUFFER_SIZE)
		receiveLength, _ := conn.Read(receiveBuffer)
		command := decodeCommandRequest(receiveBuffer[:receiveLength], symmetricKey)
		response := CommandResponse{false, false, false, false}
		if command.JoinRoom {
			room, roomExist := Rooms[command.RoomID]
			if roomExist {
				response.IsSuccess = room.AddPlayer(&user, strings.ReplaceAll(string(command.Username[:]), "\x00", ""), command.SnakeShape)
				response.JoinRoom = true
			} else {
				room := Room{
					ID:          command.RoomID,
					playerNum:   0,
					mainChannel: make(chan MoveRequest, 1),
					playerMoves: make(map[uint32]chan MoveRequest),
					players:     make(map[uint32]*Player),
					foods:       make(map[Location]Location),
				}
				Rooms[command.RoomID] = &room
				room.InitialMap()

				response.IsSuccess = room.AddPlayer(&user, strings.ReplaceAll(string(command.Username[:]), "\x00", ""), command.SnakeShape)
				response.JoinRoom = true
				go room.Start()
			}
		} else if command.ExitRoom {
			room := Rooms[user.RoomID]
			room.ExitRoom(&user)
			response.IsSuccess = true
			response.ExitRoom = true
		} else if command.Quit {
			if user.RoomID != 0 {
				room := Rooms[user.RoomID]
				room.ExitRoom(&user)
			}

			response.IsSuccess = true
			response.Quit = true
			conn.Write(encodeCommandResponse(response, symmetricKey))

			break
		}

		conn.Write(encodeCommandResponse(response, symmetricKey))
	}
}

func decodeCommandRequest(bytesCommand []byte, key []byte) CommandRequest {
	var command CommandRequest
	bytesReader := bytes.NewReader(decryptMessage(bytesCommand, key))
	binary.Read(bytesReader, binary.BigEndian, &command)
	return command
}

func decodeMove(bytesMoves []byte, key []byte) MoveRequest {
	var move MoveRequest
	bytesReader := bytes.NewReader(decryptMessage(bytesMoves, key))
	binary.Read(bytesReader, binary.BigEndian, &move)
	return move
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

func decryptMessage(message []byte, key []byte) []byte {
	block, err := aes.NewCipher(key)
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
