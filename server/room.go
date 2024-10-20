package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/gammazero/deque"
)

type DisplayResponse struct {
	Players []SendPlayer
	Foods   []Location
}

type Room struct {
	ID                     uint8
	playerNum              uint8
	mainChannel            chan MoveRequest
	playerMovesMutRun      sync.Mutex //Mutex for playerMoves on Run
	playerMovesMutMainChan sync.Mutex //Mutex for playerMoves on HandleMainChannel
	playerMoves            map[uint32]chan MoveRequest
	players                map[uint32]*Player
	roomMap                [][]uint8
	playersMut             sync.Mutex            // Mutex for players and playerNum
	foods                  map[Location]Location // Set of food
}

type Location struct {
	X uint8
	Y uint8
}

type Player struct {
	UserID     uint32
	Move       rune
	Snake      deque.Deque[Location]
	Point      uint32
	Username   string
	SnakeShape rune
}

type SendPlayer struct {
	UserID     uint32
	Move       rune
	Snake      []Location
	Point      uint32
	Username   string
	SnakeShape rune
}

const maxSleep = 750

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
			room.playersMut.Lock()
			player := room.players[userId]
			if len(moveCn) != 0 {
				move := <-moveCn
				room.MovePlayer(player, move.Move)
			} else {
				room.MovePlayer(player, player.Move)
			}
			room.playersMut.Unlock()
		}
		room.playerMovesMutRun.Unlock()

		// Spawn food
		room.playersMut.Lock()
		for i := 0; i < int(room.playerNum)-len(room.foods); i++ {
			foodLoc := room.FindLoc()
			room.foods[foodLoc] = foodLoc
			room.roomMap[foodLoc.Y][foodLoc.X] = 2
			// fmt.Printf("X: %d Y: %d\n", foodLoc.X, foodLoc.Y)
		}
		room.playersMut.Unlock()

		// Send data to client
		room.playersMut.Lock()
		var wgResponse sync.WaitGroup
		for _, player := range room.players {
			wgResponse.Add(1)
			go room.SendResponse(player, &wgResponse)
		}
		wgResponse.Wait()
		room.playersMut.Unlock()

		// Tickrate
		deltaTime := time.Since(start)
		fmt.Println(deltaTime)
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

func (room *Room) AddPlayer(user *User, username string, snakeShape rune) bool {
	if room.playerNum <= 4 {
		room.playerNum++
		user.RoomID = room.ID
		room.playerMoves[user.ID] = make(chan MoveRequest, 1)

		// Cari koordinat pertama
		room.playersMut.Lock()
		headLoc := room.FindLoc()
		var snake deque.Deque[Location]
		snake.PushFront(room.FindLoc())
		room.players[user.ID] = &Player{user.ID, '>', snake, 1, username, snakeShape}
		room.roomMap[headLoc.Y][headLoc.X] = 1
		room.playersMut.Unlock()

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
	delete(room.players, user.ID)
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

func (room *Room) MovePlayer(player *Player, move rune) {
	x := player.Snake.Front().X
	y := player.Snake.Front().Y

	switch move {
	case '>':
		if player.Move != '<' {
			player.Move = move
		}
	case '<':
		if player.Move != '>' {
			player.Move = move
		}
	case '^':
		if player.Move != 'v' {
			player.Move = move
		}
	case 'v':
		if player.Move != '^' {
			player.Move = move
		}
	}

	switch player.Move {
	case '>':
		if x == 29 || room.roomMap[y][x+1] == 1 {
			room.Restart(player)
		} else if room.roomMap[y][x+1] == 2 {
			player.Snake.PushFront(Location{x + 1, y})
			room.roomMap[y][x+1] = 1
			player.Point++
			delete(room.foods, Location{x + 1, y})
		} else {
			player.Snake.PushFront(Location{x + 1, y})
			tail := player.Snake.PopBack()
			room.roomMap[tail.Y][tail.X] = 0
			room.roomMap[y][x+1] = 1
		}

	case '<':
		if x == 0 || room.roomMap[y][x-1] == 1 {
			room.Restart(player)
		} else if room.roomMap[y][x-1] == 2 {
			player.Snake.PushFront(Location{x - 1, y})
			room.roomMap[y][x-1] = 1
			player.Point++
			delete(room.foods, Location{x - 1, y})
		} else {
			player.Snake.PushFront(Location{x - 1, y})
			tail := player.Snake.PopBack()
			room.roomMap[tail.Y][tail.X] = 0
			room.roomMap[y][x-1] = 1
		}

	case 'v':
		if y == 29 || room.roomMap[y+1][x] == 1 {
			room.Restart(player)
		} else if room.roomMap[y+1][x] == 2 {
			player.Snake.PushFront(Location{x, y + 1})
			room.roomMap[y+1][x] = 1
			player.Point++
			delete(room.foods, Location{x, y + 1})
		} else {
			player.Snake.PushFront(Location{x, y + 1})
			tail := player.Snake.PopBack()
			room.roomMap[tail.Y][tail.X] = 0
			room.roomMap[y+1][x] = 1
		}

	case '^':
		if y == 0 || room.roomMap[y-1][x] == 1 {
			room.Restart(player)
		} else if room.roomMap[y-1][x] == 2 {
			player.Snake.PushFront(Location{x, y - 1})
			room.roomMap[y-1][x] = 1
			player.Point++
			delete(room.foods, Location{x, y - 1})
		} else {
			player.Snake.PushFront(Location{x, y - 1})
			tail := player.Snake.PopBack()
			room.roomMap[tail.Y][tail.X] = 0
			room.roomMap[y-1][x] = 1
		}

	}
}

func (room *Room) Restart(player *Player) {
	player.Point = 1
	for {
		if player.Snake.Len() == 0 {
			break
		}
		loc := player.Snake.PopFront()
		room.roomMap[loc.Y][loc.X] = 0
	}
	head := room.FindLoc()
	player.Snake.PushBack(head)
	room.roomMap[head.Y][head.X] = 1
}

func (room *Room) SendResponse(player *Player, wg *sync.WaitGroup) {
	defer wg.Done()
	foods := []Location{}
	for foodLoc := range room.foods {
		foods = append(foods, foodLoc)
	}

	players := make([]SendPlayer, len(room.players))
	index := 0
	for _, player := range room.players {
		snake := make([]Location, player.Snake.Len())
		for i := 0; i < len(snake); i++ {
			snake[i] = player.Snake.At(i)
		}

		players[index] = SendPlayer{player.UserID, player.Move, snake, player.Point, player.Username, player.SnakeShape}
		index++
	}

	response := room.EncodeDisplayResponse(DisplayResponse{players, foods})
	socketUDP.WriteToUDP(response, Users[player.UserID].UdpAddress)
}

func (room *Room) EncodeDisplayResponse(response DisplayResponse) []byte {
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		log.Fatalln(err)
	}
	return jsonResponse
}
