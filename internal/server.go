package internal

import (
	"bufio"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
)

const (
	PORT     string = ":8080"
	PROTOCOL string = "tcp"
)

var (
	USAGE     [3]string = [3]string{red("\nUsage:"), yellow(" /<Command>"), cyan(" arguments")}
	NAME      [3]string = [3]string{red("/name"), yellow(" <new_name>"), cyan(" (Sets new username)")}
	MSG       [3]string = [3]string{red("/msg"), yellow(" <receiver_username> <message>"), cyan(" (Sends a DM)")}
	BROADCAST [3]string = [3]string{red("/all"), yellow(" <message>"), cyan(" (Sends a message to all users in the current room")}
	SPAM      [3]string = [3]string{red("/spam"), yellow(" <spam_n_times <message>"), cyan(" (Spams the room 'N' times)")}
	SHOUT     [3]string = [3]string{red("/shout"), yellow(" <message>"), cyan(" (Sends a message to room in capitals")}
	CREATE    [3]string = [3]string{red("/create"), yellow(" <room_name>"), cyan(" (creates a new room with the specified name)")}
	JOIN      [3]string = [3]string{red("/join"), yellow(" <room_name>"), cyan(" (Joins a room)")}
	KICK      [3]string = [3]string{red("/kick"), yellow(" <username>"), cyan(" (Kicks the user out of the room, you have to be admin)")}
	QUIT      [3]string = [3]string{red("/quit"), yellow(""), cyan(" (Quits the room)")}
	EXIT      [3]string = [3]string{red("/exit"), yellow(""), cyan(" (Close the client connection)")}
	HELP      [3]string = [3]string{red("/help"), yellow(""), cyan(" (Lists all commands)")}
	LIST      [3]string = [3]string{red("/list"), yellow(" <(optional) room_name>"), cyan(" (Lists active users)\n")}
)

const (
	cmdName       string = "/name"    //Done
	cmdMsg        string = "/msg"     //Done
	cmdBroadcast  string = "/all"     //Done
	cmdCreateRoom string = "/create"  //Done
	cmdJoinRoom   string = "/join"    //Done
	cmdQuitRoom   string = "/quit"    //Done
	cmdPromote    string = "/promote" //Done
	cmdSpam       string = "/spam"    //Done
	cmdShout      string = "/shout"   //Done
	cmdKick       string = "/kick"    //Done
	cmdExit       string = "/exit"    //TODO
	cmdHelp       string = "/help"    //Done but fix error messages
	cmdList       string = "/list"    //TODO
)

var (
	serverPrivate *rsa.PrivateKey
	serverPublic  rsa.PublicKey
)

// Colours!!
var cyan = color.New(color.FgCyan).SprintFunc()
var green = color.New(color.FgGreen).SprintFunc()
var blue = color.New(color.FgBlue).SprintFunc()
var purple = color.New(color.FgMagenta).SprintFunc()
var red = color.New(color.FgRed).SprintFunc()
var yellow = color.New(color.FgYellow).SprintFunc()

type client struct {
	conn        net.Conn
	username    string
	currentRoom string
	adminOf     []room
	modOf       []room
	public      rsa.PublicKey
}

type room struct {
	roomAdmin        *client
	roomName         string
	connectedClients []*client
	mods             []*client
}

var rooms []room
var clients []*client
var wg sync.WaitGroup

func newClient(conn net.Conn, wg sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	name := setUsername(conn, wg)

	key := setPublicKeyClient(conn, wg)

	cli := &client{
		conn:     conn,
		username: name,
		public:   key,
	}

	clients = append(clients, cli)
	fmt.Println(green("\nClient connected!"))
	fmt.Println(blue("Name: "), blue(getUsername(conn)))
	fmt.Println(cyan("Connection: "), cyan(conn.RemoteAddr().String()))

	_, err := conn.Write([]byte(serverPublic.N.String() + " " + strconv.Itoa(serverPublic.E) + "\n"))
	checkErrorServer(err, "")

	handleUserConnection(conn, wg)
}

func setPublicKeyClient(conn net.Conn, wg sync.WaitGroup) rsa.PublicKey {
	for {
		userInput, err := bufio.NewReader(conn).ReadString('\n')
		checkErrorServer(err, "")

		if userInput != "" {
			userInput = strings.Trim(userInput, "\r\n")
			args := strings.Split(userInput, " ")
			N := strings.TrimSpace(args[0])
			E := strings.TrimSpace(args[1])

			var pKey rsa.PublicKey
			i := new(big.Int)

			_, err := fmt.Sscan(N, i)
			checkErrorServer(err, "")

			pKey.N = i
			pKey.E, err = strconv.Atoi(E)

			if err != nil {
				log.Println("error scanning value:", err)
			}

			return pKey
		}
	}
}
func setUsername(conn net.Conn, wg sync.WaitGroup) string {

	for {
		userInput, err := bufio.NewReader(conn).ReadString('\n')
		checkErrorServer(err, "")

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
func getClientByUsername(name string) *client {
	for i := 0; i < len(clients); i++ {
		if clients[i].username == name {
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
	return room{nil, "", nil, nil}
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

		userInput = decrypt(userInput, *serverPrivate)
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

			var destination string
			var newRoom room

			newRoom.roomName = roomName
			newRoom.roomAdmin = getClient(conn)
			newRoom.mods = append(newRoom.mods, getClient(conn))

			rooms = append(rooms, newRoom)
			msg := "Room created with name: " + roomName

			for i := 0; i < len(clients); i++ {
				if clients[i].conn == conn {
					destination = clients[i].username
					clients[i].adminOf = append(clients[i].adminOf, newRoom)
					clients[i].modOf = append(clients[i].modOf, newRoom)
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

		case cmdPromote:
			currentRoom := getClient(conn).currentRoom
			toPromote := strings.TrimSpace(args[1])
			for i := 0; i < len(clients); i++ {
				if clients[i].username == toPromote {
					for _, v := range getClient(conn).adminOf {
						if v.roomName == currentRoom {
							for i := 0; i < len(rooms); i++ {
								if rooms[i].roomName == currentRoom {
									rooms[i].mods = append(rooms[i].mods, getClientByUsername(toPromote))
								}
							}
							clients[i].modOf = append(clients[i].modOf, getRoom(currentRoom))
							sendClientMessage(clients[i].conn, "You have been promoted to a moderator by: "+getUsername(conn), clients[i].username, "SERVER")
						}
					}

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
			kicker := getClient(conn)
			currentRoom := kicker.currentRoom
			toKick := strings.TrimSpace(args[1])

			for i := 0; i < len(clients); i++ {
				if clients[i].username != toKick {
					continue
				}

				for _, v := range getClient(conn).adminOf {
					if v.roomName == currentRoom {
						clients[i].currentRoom = ""
						sendClientMessage(clients[i].conn, "You have been kicked from '"+currentRoom+"' by: "+getUsername(conn), kicker.username, "SERVER")
					}
				}
			}

			if isMod(conn, getRoom(currentRoom).mods, kicker.username) && !isMod(getClientByUsername(toKick).conn, getRoom(currentRoom).mods, toKick) {
				getClientByUsername(toKick).currentRoom = ""
				sendClientMessage(getClientByUsername(toKick).conn, "You have been kicked from '"+currentRoom+"' by: "+getUsername(conn), toKick, "SERVER")

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
			if len(args) == 1 {
				for i := 0; i < len(clients); i++ {
					activeUsers += "'" + clients[i].username + "'" + " "

					sendClientMessage(conn, yellow(activeUsers), getUsername(conn), "SERVER: Active users are")
					checkErrorServer(err, "unable to write over client connection")

				}
			} else if len(args) == 2 {
				for i := 0; i < len(clients); i++ {
					if clients[i].currentRoom == args[1] {
						activeUsers += clients[i].username + " "

						sendClientMessage(conn, yellow(activeUsers), getUsername(conn), "SERVER: Active users in '"+getRoom(args[1]).roomName+"' are")
						checkErrorServer(err, "unable to write over client connection")
					}
				}
			}

		case cmdHelp:

		case cmdExit:
			name := getUsername(conn)
			remove(conn)

			err := conn.Close()
			checkErrorServer(err, "")

			fmt.Println(name + " disconnected")

		default:
			fmt.Println("No such command: ", cmd)
		}

		fmt.Println(clients)
	}
}

func sendClientMessage(conn net.Conn, msg string, destination string, sender string) {
	for i := 0; i < len(clients); i++ {
		if clients[i].username == destination {
			cipherText := encrypt(blue(sender)+blue(": ")+msg, clients[i].public)

			_, err := clients[i].conn.Write([]byte(cipherText + "\n"))
			checkErrorServer(err, "unable to write over client connection")
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
		if clients[i].username != owner && clients[i].currentRoom == senderRoom {
			cipherText := encrypt(blue(sender)+blue(": ")+msg, clients[i].public)

			_, err := clients[i].conn.Write([]byte(cipherText + "\n"))
			checkErrorServer(err, "unable to write over client connection")

		}
	}

}

func isMod(conn net.Conn, r []*client, user string) bool {
	fmt.Println(user)
	fmt.Println(len(r))
	for i := 0; i < len(r); i++ {
		fmt.Println(r[i].username)
		if r[i].username == user {

			return true
		}
	}
	return false
}

func contains(s []*client, str string) bool {
	for i := 0; i < len(s); i++ {
		if s[i].username == str {
			return true
		}
	}

	return false
}

func checkErrorServer(err error, errMsg string) {
	if err != nil {
		fmt.Println(errMsg + err.Error())
		os.Exit(0)
	}
}

func RunServer() {
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

	// fo, err := os.Create("sessionHistory.txt")
	// if err != nil {
	// 	panic(err)
	// }

	fmt.Println(green("Server started on port "), cyan(PORT))

	// l, err := fo.WriteString("Server started on port " + cyan(PORT) + " successfully")

	// fmt.Println(l, "bytes written successfully")
	// if err != nil {
	// 	fmt.Println(err)
	// 	fo.Close()
	// 	return
	// }
	for {
		conn, err := ln.Accept()
		checkErrorServer(err, "Unable to connect to client: ")

		go newClient(conn, wg)

		wg.Wait()
		// if err := fo.Close(); err != nil {
		// 	panic(err)
		// }
	}
}
