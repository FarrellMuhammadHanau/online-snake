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
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sort"
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

type Location struct {
	X uint8
	Y uint8
}

type Player struct {
	UserID     uint32
	Move       rune
	Snake      []Location
	Point      uint32
	Username   string
	SnakeShape rune
}

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

type DisplayResponse struct {
	Players []Player
	Foods   []Location
}

var (
	userID         uint32
	isPlaying      bool
	isPlayingMutex sync.Mutex
	symmetricKey   []byte
	userName       string
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

	// Get public key
	pubKeyBuffer := make([]byte, BUFFER_SIZE)
	pubKeyBuffLength, _ := tcpSocket.Read(pubKeyBuffer)
	pubKeyTemp, err := x509.ParsePKIXPublicKey(pubKeyBuffer[:pubKeyBuffLength])
	if err != nil {
		log.Fatalln(err)
	}
	pubKey := pubKeyTemp.(*rsa.PublicKey)

	// Send symmetric Key
	symmetricKey = make([]byte, 32)
	if _, err := io.ReadFull(crand.Reader, symmetricKey); err != nil {
		log.Fatalln(err)
	}

	encryptedSKey, err := rsa.EncryptOAEP(sha256.New(), crand.Reader, pubKey, symmetricKey, nil)
	if err != nil {
		log.Fatalln(err)
	}
	tcpSocket.Write(encryptedSKey)

	// Ambil userId
	userIdBuffer := make([]byte, BUFFER_SIZE)
	length, _ := tcpSocket.Read(userIdBuffer)
	userID = binary.BigEndian.Uint32(decryptMessage(userIdBuffer[:length]))

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
	tcpSocket.Write(encryptMessage(udpAddrBuffer.Bytes()))

	defer closeConn(tcpSocket, udpSocket)

	// Handle SIGINT and SIGTERM
	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChannel
		closeConn(tcpSocket, udpSocket)
		os.Exit(0)
	}()

	clearScreen()
	for {
		if isPlaying {
			go readKeyboard(tcpSocket, udpSocket)
			for {
				isPlayingMutex.Lock()
				if !isPlaying {
					isPlayingMutex.Unlock()
					clearScreen()
					break
				}
				draw(udpSocket)
				isPlayingMutex.Unlock()
			}
		} else {
			var roomNumStr string
			fmt.Print("Enter room number (1-255): ")
			fmt.Scanln(&roomNumStr)
			roomNum, _ := strconv.Atoi(roomNumStr)

			if roomNum > 0 && roomNum < 256 {
				fmt.Print("Enter username (5 char max): ")
				fmt.Scanln(&userName)
				if len(userName) == 0 {
					clearScreen()
					fmt.Println("username must not be blank")
					continue
				} else if len(userName) > 5 {
					userName = userName[:5]
				}

				var shapeString string
				fmt.Print("Enter snake shape: ")
				fmt.Scanln(&shapeString)
				if len(shapeString) == 0 {
					clearScreen()
					fmt.Println("snake shape must not be blank")
					continue
				}

				runeUsername := make([]rune, 5)
				tempRuneUsername := []rune(userName)
				copy(runeUsername, tempRuneUsername)
				fmt.Println(tempRuneUsername)
				commandRequest := CommandRequest{userID, true, uint8(roomNum), false, false, [5]rune(runeUsername), rune(shapeString[0])}
				encodedCommandRequest := encodeCommandRequest(commandRequest)
				tcpSocket.Write(encodedCommandRequest)

				receiveBuffer := make([]byte, BUFFER_SIZE)
				receiveLength, _ := tcpSocket.Read(receiveBuffer)
				response := decodeCommandResponse(receiveBuffer[:receiveLength])
				if response.JoinRoom && response.IsSuccess {
					isPlaying = true
				} else {
					clearScreen()
					fmt.Println("Room is full")
				}

			} else {
				clearScreen()
				fmt.Println("Enter in range of 1-255")
			}
		}
	}
}

func closeConn(tcpSocket *net.TCPConn, udpSocket *net.UDPConn) {
	udpSocket.Close()
	for {
		commandRequest := CommandRequest{userID, false, 0, false, true, [5]rune(make([]rune, 5)), 0}
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
}

func clearScreen() {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "cls")
	default:
		cmd = exec.Command("clear")
	}
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func draw(udpSocket *net.UDPConn) {
	receiveBuffer := make([]byte, BUFFER_SIZE)
	receiveLength, _, _ := udpSocket.ReadFromUDP(receiveBuffer)
	clearScreen()
	response := decodeDisplayResponse(receiveBuffer[:receiveLength])
	// fmt.Println(string(receiveBuffer[:receiveLength]))
	roomMap := [][]rune{
		{'#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#'},
		{'#', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', '#'},
		{'#', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', '#'},
		{'#', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', '#'},
		{'#', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', '#'},
		{'#', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', '#'},
		{'#', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', '#'},
		{'#', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', '#'},
		{'#', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', '#'},
		{'#', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', '#'},
		{'#', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', '#'},
		{'#', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', '#'},
		{'#', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', '#'},
		{'#', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', '#'},
		{'#', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', '#'},
		{'#', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', '#'},
		{'#', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', '#'},
		{'#', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', '#'},
		{'#', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', '#'},
		{'#', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', '#'},
		{'#', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', '#'},
		{'#', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', '#'},
		{'#', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', '#'},
		{'#', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', '#'},
		{'#', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', '#'},
		{'#', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', '#'},
		{'#', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', '#'},
		{'#', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', '#'},
		{'#', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', '#'},
		{'#', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', '#'},
		{'#', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', '#'},
		{'#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#', ' ', '#'},
	}
	for _, player := range response.Players {
		head := player.Snake[0]
		roomMap[head.Y+1][(head.X+1)*2] = player.Move
		userName := []rune(player.Username)
		// Add player to map
		for i := 1; i < len(player.Snake); i++ {
			loc := player.Snake[i]
			if i-2 < len(userName) && i > 1 {
				roomMap[loc.Y+1][(loc.X+1)*2] = userName[i-2]
				continue
			}
			roomMap[loc.Y+1][(loc.X+1)*2] = player.SnakeShape
		}
	}

	for _, food := range response.Foods {
		roomMap[food.Y+1][(food.X+1)*2] = '$'
	}

	fmt.Println(string(roomMap[0]))
	thirdLine := string(roomMap[1]) + "\tLeaderboard"
	fmt.Println(thirdLine)

	sort.SliceStable(response.Players, func(i int, j int) bool {
		if response.Players[i].Point != response.Players[j].Point {
			return response.Players[i].Point > response.Players[j].Point
		}

		return response.Players[i].Username < response.Players[j].Username
	})

	for i := 2; i < len(response.Players)+2; i++ {
		player := response.Players[i-2]
		lines := fmt.Sprintf("%s\t%s - %d - '%c'", string(roomMap[i]), string(player.Username), player.Point, player.SnakeShape)
		fmt.Println(lines)
	}

	for i := len(response.Players) + 2; i < len(roomMap); i++ {
		fmt.Println(string(roomMap[i]))
	}
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

			encodedCommand := encodeCommandRequest(CommandRequest{userID, false, 0, true, false, [5]rune(make([]rune, 5)), 0})
			tcpSocket.Write(encodedCommand)

			receiveBuffer := make([]byte, BUFFER_SIZE)
			receiveLength, _ := tcpSocket.Read(receiveBuffer)
			response := decodeCommandResponse(receiveBuffer[:receiveLength])

			if response.ExitRoom && response.IsSuccess {
				isPlaying = false
			}

			isPlayingMutex.Unlock()
			break
		} else if char == 'w' {
			moveRequest := MoveRequest{userID, '^'}
			encodedMoveRequest := encodeMoveRequest(moveRequest)
			udpSocket.Write(encodedMoveRequest)
		} else if char == 's' {
			moveRequest := MoveRequest{userID, 'v'}
			encodedMoveRequest := encodeMoveRequest(moveRequest)
			udpSocket.Write(encodedMoveRequest)
		} else if char == 'd' {
			moveRequest := MoveRequest{userID, '>'}
			encodedMoveRequest := encodeMoveRequest(moveRequest)
			udpSocket.Write(encodedMoveRequest)
		} else if char == 'a' {
			moveRequest := MoveRequest{userID, '<'}
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
