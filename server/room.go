package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"sync"
	"time"
)

type DisplayResponse struct {
	Players []Player
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
		room.players[user.ID] = &Player{user.ID, '>', []Location{headLoc}, 1, username, snakeShape}
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
	x := player.Snake[0].X
	y := player.Snake[0].Y

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
			newSnake := append([]Location{{x + 1, y}}, player.Snake...)
			player.Snake = newSnake
			room.roomMap[y][x+1] = 1
			player.Point++
			delete(room.foods, Location{x + 1, y})
		} else {
			tailX := player.Snake[player.Point-1].X
			tailY := player.Snake[player.Point-1].Y
			room.roomMap[tailY][tailX] = 0
			for i := (len(player.Snake) - 1); i > 0; i-- {
				player.Snake[i].X = player.Snake[i-1].X
				player.Snake[i].Y = player.Snake[i-1].Y
			}
			room.roomMap[y][x+1] = 1
			player.Snake[0].X++
		}

	case '<':
		if x == 0 || room.roomMap[y][x-1] == 1 {
			room.Restart(player)
		} else if room.roomMap[y][x-1] == 2 {
			newSnake := append([]Location{{x - 1, y}}, player.Snake...)
			player.Snake = newSnake
			room.roomMap[y][x-1] = 1
			player.Point++
			delete(room.foods, Location{x - 1, y})
		} else {
			tailX := player.Snake[player.Point-1].X
			tailY := player.Snake[player.Point-1].Y
			room.roomMap[tailY][tailX] = 0
			for i := (len(player.Snake) - 1); i > 0; i-- {
				player.Snake[i].X = player.Snake[i-1].X
				player.Snake[i].Y = player.Snake[i-1].Y
			}
			room.roomMap[y][x-1] = 1
			player.Snake[0].X--
		}

	case 'v':
		if y == 29 || room.roomMap[y+1][x] == 1 {
			room.Restart(player)
		} else if room.roomMap[y+1][x] == 2 {
			newSnake := append([]Location{{x, y + 1}}, player.Snake...)
			player.Snake = newSnake
			room.roomMap[y+1][x] = 1
			player.Point++
			delete(room.foods, Location{x, y + 1})
		} else {
			tailX := player.Snake[player.Point-1].X
			tailY := player.Snake[player.Point-1].Y
			room.roomMap[tailY][tailX] = 0
			for i := (len(player.Snake) - 1); i > 0; i-- {
				player.Snake[i].X = player.Snake[i-1].X
				player.Snake[i].Y = player.Snake[i-1].Y
			}
			room.roomMap[y+1][x] = 1
			player.Snake[0].Y++
		}

	case '^':
		if y == 0 || room.roomMap[y-1][x] == 1 {
			room.Restart(player)
		} else if room.roomMap[y-1][x] == 2 {
			newSnake := append([]Location{{x, y - 1}}, player.Snake...)
			player.Snake = newSnake
			room.roomMap[y-1][x] = 1
			player.Point++
			delete(room.foods, Location{x, y - 1})
		} else {
			tailX := player.Snake[player.Point-1].X
			tailY := player.Snake[player.Point-1].Y
			room.roomMap[tailY][tailX] = 0
			for i := (len(player.Snake) - 1); i > 0; i-- {
				player.Snake[i].X = player.Snake[i-1].X
				player.Snake[i].Y = player.Snake[i-1].Y
			}
			room.roomMap[y-1][x] = 1
			player.Snake[0].Y--
		}

	}
}

func (room *Room) Restart(player *Player) {
	player.Point = 1
	for _, snakeLoc := range player.Snake {
		room.roomMap[snakeLoc.Y][snakeLoc.X] = 0
	}
	player.Snake = []Location{room.FindLoc()}
}

func (room *Room) SendResponse(player *Player, wg *sync.WaitGroup) {
	defer wg.Done()
	foods := []Location{}
	for foodLoc := range room.foods {
		foods = append(foods, foodLoc)
	}
	players := []Player{}
	for _, player := range room.players {
		players = append(players, *player)
	}

	response := room.EncodeDisplayResponse(DisplayResponse{players, foods})
	// fmt.Println(len(room.foods))
	socketUDP.WriteToUDP(response, Users[player.UserID].UdpAddress)
}

func (room *Room) EncodeDisplayResponse(response DisplayResponse) []byte {
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		log.Fatalln(err)
	}
	return jsonResponse
}
