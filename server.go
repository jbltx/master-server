package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"
)

const (
	usEastCoast                            byte   = 0x00
	usWestCoast                            byte   = 0x01
	southAmerica                           byte   = 0x02
	europe                                 byte   = 0x03
	asia                                   byte   = 0x04
	australia                              byte   = 0x05
	middleEast                             byte   = 0x06
	africa                                 byte   = 0x07
	restOfTheWorld                         byte   = 0xFF
	bufferSize                             int    = 1024
	playerClientMagicNumber                byte   = 0x31
	gameServerClientMagicNumber            byte   = 0x71
	gameServerChallengeResponseMagicNumber byte   = 0x30
	gameServerShutdownMagicNumber          byte   = 0x62
	defaultIP                              string = "0.0.0.0:0"
)

var (
	playerClientReplyHeader        = []byte{0xFF, 0xFF, 0xFF, 0xFF, 0x66, 0x0A}
	gameServerChallengeReplyHeader = []byte{0xFF, 0xFF, 0xFF, 0xFF, 0x73, 0x0A}
)

// MasterServer is the base structure for the master server
type MasterServer struct {
	ctx         context.Context
	cancel      context.CancelFunc
	clients     map[net.Addr]query
	gameServers []gameServer
	connection  net.PacketConn
}

type gameServer struct {
	addr      net.Addr
	challenge uint32
}

type query struct {
	region     byte
	ipPort     string
	filters    []*filter
	isNew      bool
	receivedAt time.Time
}

// NewMasterServer creates a new instance of MasterServer
func NewMasterServer() *MasterServer {

	ctx, cancel := context.WithCancel(context.Background())

	return &MasterServer{
		ctx:         ctx,
		cancel:      cancel,
		clients:     map[net.Addr]query{},
		gameServers: []gameServer{},
	}
}

func (s *MasterServer) reply(clientAddr net.Addr, q query) {

	// TODO : Ask SQL server the query
	// Here's a dummy response
	responses := []string{"66.254.119.42:80"}

	b := bytes.NewBuffer(playerClientReplyHeader)
	for _, res := range responses {

		ipPort := strings.Split(res, ":")
		ip := strings.Split(ipPort[0], ".")

		for _, ippart := range ip {
			ipp, _ := strconv.Atoi(ippart)
			b.WriteByte(byte(ipp))
		}

		port, _ := strconv.Atoi(ipPort[1])
		b.WriteByte(byte(port))
	}
	s.connection.WriteTo(b.Bytes(), clientAddr)
}

func (s *MasterServer) handlePlayerClientMessage(clientAddr net.Addr, buffer []byte) {

	n := len(buffer)

	isNewClient := true
	for client := range s.clients {
		if clientAddr == client {
			isNewClient = false
			break
		}
	}

	fmt.Printf("Received %d bytes from %s\n", n, clientAddr)

	ipPort := strings.Builder{}
	ptr := 2
	for {
		if ptr == n || buffer[ptr] == 0x00 {
			break
		}
		ipPort.WriteByte(buffer[ptr])
		ptr++
	}

	if ptr == n {
		// this shouldn't happen, even if the client doesn't
		// want to use filters, he has to put an additional
		// 0x00 byte as "Empty Filter".
		//connectionErrorCh <- errors.New("Invalid message received")
		return
	}

	ptr++

	filters := []*filter{}
	filterStream := strings.Builder{}
	for {
		if ptr == n {
			break
		}

		if buffer[ptr] == 0x00 {
			filters = append(filters, newFilter(filterStream.String()))
			filterStream = strings.Builder{}
			if ptr == n-1 {
				break
			}

		} else {
			filterStream.WriteByte(buffer[ptr])
		}
		ptr++
	}

	q := query{
		region:     buffer[1],
		ipPort:     ipPort.String(),
		filters:    filters,
		isNew:      (isNewClient || ipPort.String() == defaultIP),
		receivedAt: time.Now(),
	}

	s.clients[clientAddr] = q
	s.reply(clientAddr, q)
}

func (s *MasterServer) findGameServer(clientAddr net.Addr) int {
	gameServerIndex := -1
	for i, gs := range s.gameServers {
		if gs.addr == clientAddr {
			gameServerIndex = i
			break
		}
	}
	return gameServerIndex
}

func (s *MasterServer) sendGameServerChallenge(clientAddr net.Addr) {

	i := s.findGameServer(clientAddr)

	if i == -1 {
		s.gameServers = append(s.gameServers, gameServer{
			addr:      clientAddr,
			challenge: rand.Uint32(),
		})
		i = len(s.gameServers) - 1
	} else {
		s.gameServers[i].challenge = rand.Uint32()
	}

	b := bytes.NewBuffer(gameServerChallengeReplyHeader)
	challengeBuffer := make([]byte, 4)
	binary.LittleEndian.PutUint32(challengeBuffer, s.gameServers[i].challenge)
	b.Write(challengeBuffer)
	s.connection.WriteTo(b.Bytes(), clientAddr)
}

func (s *MasterServer) handleGameServerChallengeResponse(clientAddr net.Addr) {
	i := s.findGameServer(clientAddr)

	if i == -1 {
		s.sendGameServerChallenge(clientAddr)
	} else {
		// parse message
		// ex : 0.\protocol\7\challenge\123\players\0\max\2\bots\0\gamedir
		//        \cstrike\map\de_dust\password\0\os\1\lan\0\region\255
		//        \type\d\secure\0\version\1.0.0.0\product\cstrike.

	}
}

func (s *MasterServer) handleGameServerShutdown(clientAddr net.Addr, buffer []byte) {
	i := s.findGameServer(clientAddr)

	if i >= 0 {
		if len(buffer) == 3 && buffer[1] == 0x0A && buffer[2] == 0x00 {
			copy(s.gameServers[i:], s.gameServers[i+1:])
			s.gameServers = s.gameServers[:len(s.gameServers)-1]
		}
	}
}

// Listen to the given address
func (s *MasterServer) Listen(addr string) error {

	var err error = nil
	s.connection, err = net.ListenPacket("udp", addr)

	if err != nil {
		panic(err)
	}

	defer s.connection.Close()

	connectionErrorCh := make(chan error, 1)
	interruptCh := make(chan os.Signal, 1)

	signal.Notify(interruptCh, os.Interrupt)

	buffer := make([]byte, bufferSize)

	go func(s *MasterServer, ctx context.Context) {
		for {
			n, clientAddr, err := s.connection.ReadFrom(buffer)
			if err != nil {
				connectionErrorCh <- err
				return
			}

			if n > 0 {

				switch buffer[0] {
				case playerClientMagicNumber:
					s.handlePlayerClientMessage(clientAddr, buffer[:n])
					break
				case gameServerClientMagicNumber:
					s.sendGameServerChallenge(clientAddr)
					break
				case gameServerChallengeResponseMagicNumber:
					s.handleGameServerChallengeResponse(clientAddr)
				case gameServerShutdownMagicNumber:
					s.handleGameServerShutdown(clientAddr, buffer[:n])
				default:
					break
				}
			}
		}
	}(s, s.ctx)

	err = nil

	select {
	case <-interruptCh:
		signal.Stop(interruptCh)
		fmt.Println("Connection interrupted")
		s.cancel()
	case <-s.ctx.Done():
		fmt.Println("Closing connection")
		err = s.ctx.Err()
	case err = <-connectionErrorCh:
	}

	return err
}
