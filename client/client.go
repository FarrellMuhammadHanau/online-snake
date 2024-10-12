package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	"github.com/eiannone/keyboard"
)

const (
	SERVER_IP   = "127.0.0.1"
	UDP_PORT    = "1566"
	TCP_PORT    = "1567"
	UDP         = "udp4"
	TCP         = "tcp4"
	BUFFER_SIZE = 2048
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

type DisplayResponse struct {
	Data string
}

var (
	userID         uint32
	isPlaying      bool
	isPlayingMutex sync.Mutex
)

func main() {
	isPlaying = false

	remoteTCPAddr, err := net.ResolveTCPAddr(TCP, net.JoinHostPort(SERVER_IP, TCP_PORT))
	if err != nil {
		log.Fatalln(err)
	}
	tcpSocket, err := net.DialTCP(TCP, nil, remoteTCPAddr)
	if err != nil {
		log.Fatalln(err)
	}

	// Ambil userId
	userIdBuffer := make([]byte, BUFFER_SIZE)
	length, _ := tcpSocket.Read(userIdBuffer)
	userID = binary.BigEndian.Uint32(userIdBuffer[:length])

	remoteUdpAddr, err := net.ResolveUDPAddr(UDP, net.JoinHostPort(SERVER_IP, UDP_PORT))
	if err != nil {
		log.Fatalln(err)
	}
	udpSocket, err := net.DialUDP(UDP, nil, remoteUdpAddr)
	if err != nil {
		log.Fatalln(err)
	}

	// Send udp address
	udpAddrBuffer := new(bytes.Buffer)
	udpAddrBuffer.WriteString(udpSocket.LocalAddr().String())
	tcpSocket.Write(udpAddrBuffer.Bytes())

	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChannel
		udpSocket.Close()
		for {
			commandRequest := CommandRequest{userID, false, 0, false, false, true}
			encodedCommandRequest := encodeCommandRequest(commandRequest)
			tcpSocket.Write(encodedCommandRequest)

			receiveBuffer := make([]byte, BUFFER_SIZE)
			receiveLength, _ := tcpSocket.Read(receiveBuffer)
			commandResponse := decodeCommandResponse(receiveBuffer[:receiveLength])

			if commandResponse.IsSuccess && commandResponse.Quit {
				break
			}
		}
		tcpSocket.Close()

		os.Exit(0)
	}()

	for {
		clearScreen()
		if isPlaying {
			go readKeyboard(tcpSocket, udpSocket)
			for {
				isPlayingMutex.Lock()
				if !isPlaying {
					isPlayingMutex.Unlock()
					break
				}
				draw(udpSocket)
				isPlayingMutex.Unlock()
			}
		} else {
			var roomNumStr string
			fmt.Print("Enter room number: ")
			fmt.Scanln(&roomNumStr)
			roomNum, _ := strconv.Atoi(roomNumStr)
			if roomNum > 0 && roomNum < 256 {
				commandRequest := CommandRequest{userID, true, uint8(roomNum), false, false, false}
				encodedCommandRequest := encodeCommandRequest(commandRequest)
				tcpSocket.Write(encodedCommandRequest)

				receiveBuffer := make([]byte, BUFFER_SIZE)
				receiveLength, _ := tcpSocket.Read(receiveBuffer)
				response := decodeCommandResponse(receiveBuffer[:receiveLength])
				if response.JoinRoom && response.IsSuccess {
					isPlaying = true
				} else {
					fmt.Println("Room is full")
				}

			} else {
				fmt.Println("Enter in range of 1-255")
			}
		}
	}
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func draw(udpSocket *net.UDPConn) {
	receiveBuffer := make([]byte, BUFFER_SIZE)
	receiveLength, _, _ := udpSocket.ReadFromUDP(receiveBuffer)
	clearScreen()
	fmt.Println(decodeDisplayResponse(receiveBuffer[:receiveLength]).Data)
}

func readKeyboard(tcpSocket *net.TCPConn, udpSocket *net.UDPConn) {
	if err := keyboard.Open(); err != nil {
		log.Fatalln(err)
	}
	defer keyboard.Close()

	for {
		char, key, err := keyboard.GetKey()
		if err != nil {
			log.Fatalln(err)
		}

		if key == keyboard.KeyEsc {
			isPlayingMutex.Lock()

			encodedCommand := encodeCommandRequest(CommandRequest{userID, false, 0, false, true, false})
			tcpSocket.Write(encodedCommand)

			receiveBuffer := make([]byte, BUFFER_SIZE)
			receiveLength, _ := tcpSocket.Read(receiveBuffer)
			response := decodeCommandResponse(receiveBuffer[:receiveLength])

			if response.ExitRoom && response.IsSuccess {
				isPlaying = false
			}

			isPlayingMutex.Unlock()
			break
		} else if char == 'r' {

		} else if char == 'w' {
			moveRequest := MoveRequest{userID, 'w'}
			encodedMoveRequest := encodeMoveRequest(moveRequest)
			udpSocket.Write(encodedMoveRequest)
		} else if char == 's' {
			moveRequest := MoveRequest{userID, 's'}
			encodedMoveRequest := encodeMoveRequest(moveRequest)
			udpSocket.Write(encodedMoveRequest)
		} else if char == 'd' {
			moveRequest := MoveRequest{userID, 'd'}
			encodedMoveRequest := encodeMoveRequest(moveRequest)
			udpSocket.Write(encodedMoveRequest)
		} else if char == 'a' {
			moveRequest := MoveRequest{userID, 'a'}
			encodedMoveRequest := encodeMoveRequest(moveRequest)
			udpSocket.Write(encodedMoveRequest)
		}
	}
}

func encodeCommandRequest(request CommandRequest) []byte {
	bytesBuffer := new(bytes.Buffer)
	err := binary.Write(bytesBuffer, binary.BigEndian, request)
	if err != nil {
		log.Fatalln(err)
	}
	return bytesBuffer.Bytes()
}

func encodeMoveRequest(request MoveRequest) []byte {
	bytesBuffer := new(bytes.Buffer)
	err := binary.Write(bytesBuffer, binary.BigEndian, request)
	if err != nil {
		log.Fatalln(err)
	}
	return bytesBuffer.Bytes()
}

func decodeCommandResponse(bytesResponse []byte) CommandResponse {
	var response CommandResponse
	bytesReader := bytes.NewReader(bytesResponse)
	err := binary.Read(bytesReader, binary.BigEndian, &response)
	if err != nil {
		log.Fatalln(err)
	}
	return response
}

func decodeDisplayResponse(bytesResponse []byte) DisplayResponse {
	response := DisplayResponse{string(bytesResponse)}
	return response
}
