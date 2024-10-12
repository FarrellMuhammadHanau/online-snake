package main

import (
	"bytes"
	"encoding/binary"
	"log"
	"math/rand"
	"net"
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
	UserID   uint32
	JoinRoom bool
	RoomID   uint8
	Restart  bool
	ExitRoom bool
	Quit     bool
}

type CommandResponse struct {
	IsSuccess bool
	Restart   bool
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

func main() {
	Users = make(map[uint32]*User)
	Rooms = make(map[uint8]*Room)

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
		receiveLength, _, _ := conn.ReadFromUDP(receiveBuffer)
		move := decodeMove(receiveBuffer[:receiveLength])
		room := Rooms[Users[move.UserID].RoomID]
		room.mainChannel <- move
	}
}

func readTCP(conn *net.TCPConn) {
	defer conn.Close()

	user := User{}

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
	conn.Write(bytesID)

	// Get UDP Address
	udpAddrBuffer := make([]byte, BUFFER_SIZE)
	length, _ := conn.Read(udpAddrBuffer)
	user.UdpAddress, _ = net.ResolveUDPAddr(UDP, string(udpAddrBuffer[:length]))

	for {
		receiveBuffer := make([]byte, BUFFER_SIZE)
		receiveLength, _ := conn.Read(receiveBuffer)
		command := decodeCommandRequest(receiveBuffer[:receiveLength])
		response := CommandResponse{false, false, false, false, false}
		if command.JoinRoom {
			room, roomExist := Rooms[command.RoomID]
			if roomExist {
				response.IsSuccess = room.AddPlayer(&user)
				response.JoinRoom = true
			} else {
				room := Room{
					ID:          command.RoomID,
					playerNum:   0,
					mainChannel: make(chan MoveRequest, 1),
					playerMoves: make(map[uint32]chan MoveRequest),
					players:     make(map[uint32]*Player),
					foodCount:   0,
				}
				Rooms[command.RoomID] = &room

				response.IsSuccess = room.AddPlayer(&user)
				response.JoinRoom = true
				go room.Start()
			}
		} else if command.ExitRoom {
			room := Rooms[user.RoomID]
			room.ExitRoom(&user)
			response.IsSuccess = true
			response.ExitRoom = true
		} else if command.Restart {

		} else if command.Quit {
			if user.RoomID != 0 {
				room := Rooms[user.RoomID]
				room.ExitRoom(&user)
			}

			response.IsSuccess = true
			response.Quit = true
			conn.Write(encodeCommandResponse(response))

			break
		} else {

		}

		conn.Write(encodeCommandResponse(response))
	}
}

func decodeCommandRequest(bytesCommand []byte) CommandRequest {
	var command CommandRequest
	bytesReader := bytes.NewReader(bytesCommand)
	binary.Read(bytesReader, binary.BigEndian, &command)
	return command
}

func decodeMove(bytesMoves []byte) MoveRequest {
	var move MoveRequest
	bytesReader := bytes.NewReader(bytesMoves)
	binary.Read(bytesReader, binary.BigEndian, &move)
	return move
}

func encodeCommandResponse(response CommandResponse) []byte {
	buffer := new(bytes.Buffer)
	binary.Write(buffer, binary.BigEndian, response)
	return buffer.Bytes()
}
