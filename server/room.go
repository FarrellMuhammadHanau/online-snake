package main

import (
	"bytes"
	"math/rand"
	"sync"
	"time"
)

type DisplayResponse struct {
	Data string
}

type Location struct {
	x uint8
	y uint8
}

type Room struct {
	ID                     uint8
	playerNum              uint8
	mainChannel            chan MoveRequest
	playerMovesMutRun      sync.Mutex
	playerMovesMutMainChan sync.Mutex
	playerMoves            map[uint32]chan MoveRequest
	players                map[uint32]*Player
	roomMap                [][]uint8
	mapMut                 sync.Mutex
	foods                  map[Location]Location // Set of food
}

type Player struct {
	UserID uint32
	Move   rune
	Snake  []Location
	Point  uint32
}

const (
	maxSleep = 750
)

func (room *Room) InitialMap() {
	room.roomMap = [][]uint8{
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	}
}

func (room *Room) Start() {
	room.InitialMap()
	defer delete(Rooms, room.ID)
	var wg sync.WaitGroup
	wg.Add(1)
	go room.HandleMainChannel(&wg)
	wg.Add(1)
	go room.Run(&wg)
	wg.Wait()
}

func (room *Room) Run(wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		start := time.Now()

		if room.playerNum == 0 {
			break
		}

		// Move all player
		room.playerMovesMutRun.Lock()
		for userId, moveCn := range room.playerMoves {
			room.mapMut.Lock()
			if len(moveCn) != 0 {
				move := <-moveCn
				player := room.players[userId]

				player.Move = move.Move
				room.MovePlayer(player)
			}
			room.mapMut.Unlock()
		}
		room.playerMovesMutRun.Unlock()

		// Spawn food
		for i := uint8(0); i < room.playerNum-uint8(len(room.foods)); i++ {
			foodLoc := room.FindLoc()
			room.foods[foodLoc] = foodLoc
			room.roomMap[foodLoc.y][foodLoc.x] = 2
		}

		deltaTime := time.Since(start)
		if time.Duration(deltaTime.Milliseconds()) < maxSleep*time.Millisecond {
			time.Sleep(maxSleep*time.Millisecond - time.Duration(deltaTime.Milliseconds()))
		}
	}
}

func (room *Room) HandleMainChannel(wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		move := <-room.mainChannel
		if room.playerNum == 0 {
			break
		}

		room.playerMovesMutMainChan.Lock()
		if _, exist := room.playerMoves[move.UserID]; exist {
			if len(room.playerMoves[move.UserID]) != 0 {
				<-room.playerMoves[move.UserID]
			}
			room.playerMoves[move.UserID] <- move
		}
		room.playerMovesMutMainChan.Unlock()
	}
}

func (room *Room) AddPlayer(user *User) bool {
	if room.playerNum <= 4 {
		room.playerNum++
		user.RoomID = room.ID
		room.playerMoves[user.ID] = make(chan MoveRequest, 1)

		// Cari koordinat pertama
		room.mapMut.Lock()
		headLoc := room.FindLoc()
		room.players[user.ID] = &Player{user.ID, '>', []Location{headLoc}, 1}
		room.roomMap[headLoc.y][headLoc.x] = 1
		room.mapMut.Unlock()

		return true
	}
	return false
}

func (room *Room) ExitRoom(user *User) {
	// Lock for run
	room.playerMovesMutRun.Lock()
	// Lock for handle main channel
	room.playerMovesMutMainChan.Lock()

	delete(room.playerMoves, user.ID)
	// Make sure HandleMainChannel for loop break
	room.mainChannel <- MoveRequest{user.ID, 'e'}

	user.RoomID = 0
	room.playerNum--

	room.playerMovesMutMainChan.Unlock()
	room.playerMovesMutRun.Unlock()
}

func (room *Room) FindLoc() Location {
	var x, y uint8
	for {
		x = uint8(rand.Intn(30))
		y = uint8(rand.Intn(30))
		if room.roomMap[y][x] == 0 {
			break
		}

	}
	return Location{x, y}
}

func (room *Room) MovePlayer(player *Player) {
	x := player.Snake[0].x
	y := player.Snake[0].y
	switch player.Move {
	case '>':
		if x == 29 || room.roomMap[y][x+1] == 1 {
			room.Restart(player)
		} else if room.roomMap[y][x+1] == 2 {
			newSnake := append([]Location{{x + 1, y}}, player.Snake...)
			player.Snake = newSnake
			room.roomMap[y][x+1] = 1
			player.Point++
		} else {
			tailX := player.Snake[player.Point-1].x
			tailY := player.Snake[player.Point-1].y
			room.roomMap[tailY][tailX] = 0
			room.roomMap[y][x+1] = 1
			player.Snake[0].x++
			for i := (len(player.Snake) - 2); i >= 0; i++ {
				player.Snake[i].x = player.Snake[i-1].x
				player.Snake[i].y = player.Snake[i-1].y
			}
		}

	case '<':
		if x == 0 || room.roomMap[y][x-1] == 1 {
			room.Restart(player)
		} else if room.roomMap[y][x-1] == 2 {
			newSnake := append([]Location{{x - 1, y}}, player.Snake...)
			player.Snake = newSnake
			room.roomMap[y][x-1] = 1
			player.Point++
		} else {
			tailX := player.Snake[player.Point-1].x
			tailY := player.Snake[player.Point-1].y
			room.roomMap[tailY][tailX] = 0
			room.roomMap[y][x-1] = 1
			player.Snake[0].x--
			for i := (len(player.Snake) - 2); i >= 0; i++ {
				player.Snake[i].x = player.Snake[i-1].x
				player.Snake[i].y = player.Snake[i-1].y
			}
		}

	case '^':
		if y == 29 || room.roomMap[y+1][x] == 1 {
			room.Restart(player)
		} else if room.roomMap[y+1][x] == 2 {
			newSnake := append([]Location{{x, y + 1}}, player.Snake...)
			player.Snake = newSnake
			room.roomMap[y+1][x] = 1
			player.Point++
		} else {
			tailX := player.Snake[player.Point-1].x
			tailY := player.Snake[player.Point-1].y
			room.roomMap[tailY][tailX] = 0
			room.roomMap[y+1][x] = 1
			player.Snake[0].y++
			for i := (len(player.Snake) - 2); i >= 0; i++ {
				player.Snake[i].x = player.Snake[i-1].x
				player.Snake[i].y = player.Snake[i-1].y
			}
		}

	case 'v':
		if y == 0 || room.roomMap[y-1][x] == 1 {
			room.Restart(player)
		} else if room.roomMap[y-1][x] == 2 {
			newSnake := append([]Location{{x, y - 1}}, player.Snake...)
			player.Snake = newSnake
			room.roomMap[y-1][x] = 1
			player.Point++
		} else {
			tailX := player.Snake[player.Point-1].x
			tailY := player.Snake[player.Point-1].y
			room.roomMap[tailY][tailX] = 0
			room.roomMap[y-1][x] = 1
			player.Snake[0].y--
			for i := (len(player.Snake) - 2); i >= 0; i++ {
				player.Snake[i].x = player.Snake[i-1].x
				player.Snake[i].y = player.Snake[i-1].y
			}
		}

	}
}

func (room *Room) Restart(player *Player) {
	player.Point = 1
	for _, snakeLoc := range player.Snake {
		room.roomMap[snakeLoc.y][snakeLoc.x] = 0
	}
	player.Snake = []Location{room.FindLoc()}
}

func (room *Room) SendResponse(userID uint32, data string) {
	response := room.EncodeDisplayResponse(DisplayResponse{data})
	socketUDP.WriteToUDP(response, Users[userID].UdpAddress)
}

func (room *Room) EncodeDisplayResponse(response DisplayResponse) []byte {
	bytesBuffer := new(bytes.Buffer)
	bytesBuffer.WriteString(response.Data)
	return bytesBuffer.Bytes()
}
