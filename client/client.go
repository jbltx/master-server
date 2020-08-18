package client

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

type ServerEndpoint struct {
	IP   net.IP
	Port uint16
}

var nullEndpoint ServerEndpoint = ServerEndpoint{
	IP:   net.ParseIP("0.0.0.0"),
	Port: 0,
}

func NewServerEndpoint(buffer []byte) *ServerEndpoint {
	return &ServerEndpoint{
		IP:   net.IPv4(buffer[0], buffer[1], buffer[2], buffer[3]),
		Port: binary.BigEndian.Uint16(buffer[4:]),
	}
}

func (c *ServerEndpoint) String() string {
	p := strconv.Itoa(int(c.Port))
	return c.IP.String() + ":" + p
}

func sendListRequest(c *net.UDPConn) {
	const queryHeader byte = 0x31
	var regionCode byte = 0xFF
	ipStart := "0.0.0.0:0"
	filter := ""

	var b bytes.Buffer

	b.WriteByte(queryHeader)
	b.WriteByte(regionCode)
	b.WriteString(ipStart)
	b.WriteString(filter)

	_, err := c.Write(b.Bytes())
	if err != nil {
		log.Fatal(err)
	}

	buffer := make([]byte, 1600)
	n, _, err := c.ReadFromUDP(buffer)
	if err != nil {
		log.Fatal(err)
	}

	const structSize int = 6

	if n%structSize > 0 {
		log.Fatal("Query list response has a length which is not multiple of 6")
	}

	var replyHeader []byte = []byte{0xFF, 0xFF, 0xFF, 0xFF, 0x66, 0x0A}
	if bytes.Equal(replyHeader, buffer[0:structSize]) {
		res := buffer[structSize:n]
		count := len(res)
		i := 0
		for count > 0 {
			endpoint := NewServerEndpoint(res[i*structSize : i*structSize+structSize])
			fmt.Println("- Server " + endpoint.String())
			count -= 6
			i++
		}
	} else {
		log.Fatal("Query list response header is malformed")
	}
}

func sendJoinRequest(c *net.UDPConn) {
	_, err := c.Write([]byte{0x71})
	if err != nil {
		log.Fatal(err)
	}

	buffer := make([]byte, 1600)
	n, _, err := c.ReadFromUDP(buffer)
	if err != nil {
		log.Fatal(err)
	}

	var replyHeader []byte = []byte{0xFF, 0xFF, 0xFF, 0xFF, 0x73, 0x0A}
	if bytes.Equal(replyHeader, buffer[0:6]) {
		res := buffer[6:n]
		if len(res) != 4 {
			log.Fatal("Join response has invalid length to parse int32 value")
		}
		challengeNumber := int32(binary.BigEndian.Uint32(res))
		challengeNumberStr := strconv.Itoa(int(challengeNumber))
		fmt.Println("-> Received challenge number " + challengeNumberStr)

		challengeData := "0\n\\protocol\\7\\challenge\\" + challengeNumberStr
		challengeData += "\\players\\1\\max\\4\\bots\\0\\gamedir\\cstrike"
		challengeData += "\\map\\de_dust\\password\\0\\os\\l\\lan\\0\\region\\255"
		challengeData += "\\type\\d\\secure\\0\\version\\1.0.0.28\\product\\cstrike\n"
		_, err := c.Write([]byte(challengeData))
		if err != nil {
			log.Fatal(err)
		}

	} else {
		log.Fatal("Join response header is malformed")
	}
}

func runClient() {
	arguments := os.Args
	if len(arguments) == 1 {
		fmt.Println("Please provide a host:port string")
		return
	}
	CONNECT := arguments[1]

	s, err := net.ResolveUDPAddr("udp4", CONNECT)
	c, err := net.DialUDP("udp4", nil, s)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("The UDP server is %s\n", c.RemoteAddr().String())
	defer c.Close()

	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print(">> ")
		text, _ := reader.ReadString('\n')
		data := []byte(text + "\n")
		cmd := strings.TrimSpace(string(data))

		switch cmd {
		case "STOP":
			fmt.Println("Exiting UDP client!")
			return
		case "LIST":
			sendListRequest(c)
		case "JOIN":
			sendJoinRequest(c)
		default:
			continue
		}
	}
}
