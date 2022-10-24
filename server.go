package main

import (
	"bufio"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	PORT     string = ":8080"
	PROTOCOL string = "tcp"
	USAGE    string = "Usage: /<Command> arguments \n"
	NAME     string = "/name <new_name> (Sets new username) \n"
	MSG      string = "/msg <receiver_username> <message> (Sends a DM) \n"
	QUIT     string = "/quit (Quits the server) \n"
	HELP     string = "/help (Lists all commands) \n"
	LIST     string = "/list <(optional) room_name> (Lists active users) \n"
)

const (
	cmdName       string = "/name"   //Done
	cmdMsg        string = "/msg"    //Done
	cmdBroadcast  string = "/all"    //Done
	cmdCreateRoom string = "/create" //Done
	cmdJoinRoom   string = "/join"   //Done
	cmdQuitRoom   string = "/quit"   //Done
	cmdSpam       string = "/spam"   //Done
	cmdShout      string = "/shout"  //Done
	cmdKick       string = "/kick"   //Done
	cmdExit       string = "/exit"   //TODO
	cmdHelp       string = "/help"   //Done but fix error messages
	cmdList       string = "/list"   //TODO
)

var (
	serverPrivate *rsa.PrivateKey
	serverPublic  rsa.PublicKey
)

type client struct {
	conn        net.Conn
	username    string
	currentRoom string
	adminOf     []room
	public      rsa.PublicKey
}

type room struct {
	roomAdmin        *client
	roomName         string
	connectedClients []*client
}

var rooms []room
var clients []*client
var wg sync.WaitGroup

func newClient(conn net.Conn, wg sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()
	fmt.Println("Client connected: ", conn.RemoteAddr().String())

	name := setUsername(conn, wg)
	key := setPublicKeyClient(conn, wg)

	cli := &client{
		conn:     conn,
		username: name,
		public:   key,
	}

	clients = append(clients, cli)
	fmt.Println(cli.username)
	fmt.Println(clients)

	handleUserConnection(conn, wg)
}

func setPublicKeyClient(conn net.Conn, wg sync.WaitGroup) rsa.PublicKey {
	for {
		userInput, err := bufio.NewReader(conn).ReadString('\n')

		if err != nil {
			fmt.Println(err.Error())
			os.Exit(0)
		}

		if userInput != "" {
			userInput = strings.Trim(userInput, "\r\n")
			args := strings.Split(userInput, " ")
			N := strings.TrimSpace(args[0])
			E := strings.TrimSpace(args[1])

			var pKey rsa.PublicKey
			i := new(big.Int)
			_, err := fmt.Sscan(N, i)

			if err != nil {
				log.Println("error scanning value:", err)
			}

			pKey.N = i
			pKey.E, err = strconv.Atoi(E)

			if err != nil {
				log.Println("error scanning value:", err)
			}

			//fmt.Println(N)
			//fmt.Println(E)

			//userInput = strings.Trim(userInput, "\n")
			//return userInput
		}
	}
}
func setUsername(conn net.Conn, wg sync.WaitGroup) string {

	for {
		userInput, err := bufio.NewReader(conn).ReadString('\n')

		if err != nil {
			fmt.Println(err.Error())
			os.Exit(0)
		}

		if userInput != "" {
			userInput = strings.Trim(userInput, "\n")
			return userInput
		}
	}

	return "anonymous"

}

func getUsername(conn net.Conn) string {
	for i := 0; i < len(clients); i++ {
		if clients[i].conn == conn {
			return clients[i].username
		}
	}
	return "No username available"
}

func getClient(conn net.Conn) *client {
	for i := 0; i < len(clients); i++ {
		if clients[i].conn == conn {
			return clients[i]
		}
	}
	return nil
}

func getRoom(roomName string) room {
	for i := 0; i < len(rooms); i++ {
		if rooms[i].roomName == roomName {
			return rooms[i]
		}
	}
	return room{nil, "", nil}
}

func remove(conn net.Conn) {
	for i := 0; i < len(clients); i++ {
		if clients[i].conn == conn {
			clients[i] = clients[len(clients)-1]
			clients = clients[:len(clients)-1]
		}
	}
}

func handleUserConnection(conn net.Conn, wg sync.WaitGroup) {

	for {
		userInput, err := bufio.NewReader(conn).ReadString('\n')

		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				fmt.Println("Connection closed with one user")
			}
			break
		}

		userInput = strings.Trim(userInput, "\r\n")
		args := strings.Split(userInput, " ")
		cmd := strings.TrimSpace(args[0])

		switch cmd {

		case cmdName:
			for i := 0; i < len(clients); i++ {
				if clients[i].conn == conn {
					newName := strings.TrimSpace(args[1])
					clients[i].username = newName
					_, e := conn.Write([]byte("You changed your name to: " + newName))

					if e != nil {
						log.Fatalln("unable to write over client connection")
					}
				}
			}

		case cmdMsg:
			destination := args[1]
			if userInput != "" {
				msg := strings.Join(args[2:], " ")
				sendClientMessage(conn, msg, destination, getUsername(conn))
			}

		case cmdBroadcast:
			var owner string
			for i := 0; i < len(clients); i++ {
				if clients[i].conn == conn {
					owner = clients[i].username
				}
			}
			if userInput != "" {
				msg := strings.Join(args[1:], " ")
				broadcastMessage(conn, msg, owner, getUsername(conn))
			}

		case cmdCreateRoom:
			roomName := strings.TrimSpace(args[1])
			var newRoom room
			newRoom.roomName = roomName
			newRoom.roomAdmin = getClient(conn)
			rooms = append(rooms, newRoom)
			msg := "Room created with name: " + roomName

			var destination string
			for i := 0; i < len(clients); i++ {
				if clients[i].conn == conn {
					destination = clients[i].username
					clients[i].adminOf = append(clients[i].adminOf, newRoom)
				}
			}

			sendClientMessage(conn, msg, destination, "SERVER")

		case cmdJoinRoom:
			var cli *client
			roomName := strings.TrimSpace(args[1])
			for i := 0; i < len(clients); i++ {
				if clients[i].conn == conn {
					cli = clients[i]
					clients[i].currentRoom = roomName
				}
			}

			for i := 0; i < len(rooms); i++ {
				if rooms[i].roomName == roomName {
					rooms[i].connectedClients = append(rooms[i].connectedClients, cli)
				}
			}

		case cmdQuitRoom:
			for i := 0; i < len(clients); i++ {
				if clients[i].conn == conn {
					clients[i].currentRoom = ""
				}
			}

		case cmdShout:
			var owner string
			for i := 0; i < len(clients); i++ {
				if clients[i].conn == conn {
					owner = clients[i].username
				}
			}
			if userInput != "" {
				msg := strings.ToUpper(strings.Join(args[1:], " "))
				broadcastMessage(conn, msg, owner, getUsername(conn))
			}

		case cmdKick:
			currentRoom := getClient(conn).currentRoom
			toKick := strings.TrimSpace(args[1])
			for i := 0; i < len(clients); i++ {
				if clients[i].username == toKick {
					for _, v := range getClient(conn).adminOf {
						if v.roomName == currentRoom {
							clients[i].currentRoom = ""
							sendClientMessage(clients[i].conn, "You have been kicked from '"+currentRoom+"' by: "+getUsername(conn), clients[i].username, "SERVER")
						}
					}

				}
			}

		case cmdSpam:
			var owner string
			for i := 0; i < len(clients); i++ {
				if clients[i].conn == conn {
					owner = clients[i].username
				}
			}
			if userInput != "" {
				spamCount, _ := strconv.Atoi(strings.TrimSpace(args[1]))
				msg := strings.Join(args[2:], " ")
				for i := 0; i < spamCount; i++ {
					broadcastMessage(conn, msg, owner, getUsername(conn))
					time.Sleep(250 * time.Millisecond)
				}
			}

		case cmdList:
			var activeUsers string
			if len(args) == 0 {
				for i := 0; i < len(clients); i++ {
					activeUsers += clients[i].username + " "

					_, e := conn.Write([]byte(activeUsers))

					if e != nil {
						log.Fatalln("unable to write over client connection")
					}
				}
			} else if len(args) == 1 {
				for i := 0; i < len(clients); i++ {
					activeUsers += clients[i].username + " "

					_, e := conn.Write([]byte(activeUsers))

					if e != nil {
						log.Fatalln("unable to write over client connection")
					}
				}
			}

			fmt.Println("cmdList: ", cmd)

		case cmdHelp:
			commandList := [6]string{USAGE, NAME, MSG, QUIT, HELP, LIST}
			for i := 0; i < 6; i++ {
				_, e := conn.Write([]byte(commandList[i]))

				if e != nil {
					log.Fatalln("unable to write over client connection")
				}
				time.Sleep(10 * time.Millisecond)
			}

		case cmdExit:
			name := getUsername(conn)
			remove(conn)
			err := conn.Close()
			fmt.Println(name + " disconnected")

			if err != nil {
				fmt.Println(err.Error())
			}

		default:
			fmt.Println("No such command: ", cmd)
		}

	}

}

func sendClientMessage(conn net.Conn, msg string, destination string, sender string) {

	for i := 0; i < len(clients); i++ {
		//fmt.Println(clients[i].username)
		if clients[i].username == destination {
			_, e := clients[i].conn.Write([]byte(sender + ": " + msg + "\n"))

			if e != nil {
				log.Fatalln("unable to write over client connection")
			}
		}
	}

}

func broadcastMessage(conn net.Conn, msg string, owner string, sender string) {

	var senderRoom string = ""
	for i := 0; i < len(clients); i++ {
		if clients[i].conn == conn {
			senderRoom = clients[i].currentRoom
		}
	}

	for i := 0; i < len(clients); i++ {
		//fmt.Println(clients[i].username)
		if clients[i].username != owner && clients[i].currentRoom == senderRoom {
			_, e := clients[i].conn.Write([]byte(sender + ": " + msg + "\n"))

			if e != nil {
				log.Fatalln("unable to write over client connection")
			}
		}
	}

}

// function to encrypt message to be sent
func encrypt(msg string, key rsa.PublicKey) string {

	label := []byte("OAEP Encrypted")
	rng := rand.Reader

	// * using OAEP algorithm to make it more secure
	// * using sha256
	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rng, &key, []byte(msg), label)
	// check for errors
	if err != nil {
		log.Fatalln("unable to encrypt")
	}

	return base64.StdEncoding.EncodeToString(ciphertext)
}

// function to decrypt message to be received
func decrypt(cipherText string, key rsa.PrivateKey) string {

	ct, _ := base64.StdEncoding.DecodeString(cipherText)
	label := []byte("OAEP Encrypted")
	rng := rand.Reader

	// decrypting based on same parameters as encryption
	plaintext, err := rsa.DecryptOAEP(sha256.New(), rng, &key, ct, label)
	// check for errors
	if err != nil {
		log.Fatalln(err)
	}
	return string(plaintext)
}

func main() {
	//wg = sync.WaitGroup{}
	//wg1 := &wg
	fmt.Println("Server starting...")
	ln, err := net.Listen(PROTOCOL, PORT)

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)

	if err != nil {
		log.Fatalln(err)
	}

	serverPrivate = privateKey
	serverPublic = privateKey.PublicKey

	if err != nil {
		fmt.Println("Unable to start server: ", err.Error())
		os.Exit(0)
	}

	defer ln.Close()
	fmt.Println("Server started on port ", PORT)
	//var oldConn net.Conn
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Unable to connect to client: ", err.Error())
			os.Exit(0)
		}

		//if conn != oldConn {
		//wg.Add(1)
		go newClient(conn, wg)

		//}
		//oldConn = conn
		fmt.Println("Server polling")
		wg.Wait()
		fmt.Println("Server closed")
	}
}
