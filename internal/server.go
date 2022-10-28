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
)

// Port to open the serve and the protocol to be used
const (
	PORT     string = ":8080"
	PROTOCOL string = "tcp"
)

// Key pair for the server
var (
	serverPrivate *rsa.PrivateKey
	serverPublic  rsa.PublicKey
)

// Each client is a struct that contains information about themselves
type client struct {
	conn        net.Conn
	username    string
	currentRoom string
	adminOf     []room
	modOf       []room
	public      rsa.PublicKey
}

// Each room is a struct that contains information about itself
type room struct {
	roomAdmin        *client
	roomName         string
	connectedClients []*client
	mods             []*client
}

var rooms []room
var clients []*client
var wg sync.WaitGroup

var logFileName string = "../../logging/sessionHistory.txt"

func newClient(conn net.Conn) {
	wg.Add(1)
	defer wg.Done()

	// First message from the user contains the selected username and a generated public key
	name := setUsername(conn)
	key := setPublicKeyClient(conn)

	cli := &client{
		conn:     conn,
		username: name,
		public:   key,
	}

	clients = append(clients, cli)

	// Server informs that a client is connected with username and the remote adress
	fmt.Println(green("\nClient connected!"))
	fmt.Println(blue("Name: "), blue(getUsername(conn)))
	fmt.Println(cyan("Connection: "), cyan(conn.RemoteAddr().String()))

	// Logs the connect action
	logText := "Client connected: " + getUsername(conn) + ", Connection: " + conn.RemoteAddr().String()
	writeLog(logText)

	// First message from the server to the clients contains a generated public key for the server
	_, err := conn.Write([]byte(serverPublic.N.String() + " " + strconv.Itoa(serverPublic.E) + "\n"))
	checkErrorServer(err, "")

	handleUserConnection(conn)
}

// Set the public key field for the client to be used as a decryption key for the future messages
func setPublicKeyClient(conn net.Conn) rsa.PublicKey {
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

// Set the username for a user that will be used as an identifier
func setUsername(conn net.Conn) string {

	for {
		userInput, err := bufio.NewReader(conn).ReadString('\n')
		checkErrorServer(err, "")

		if userInput != "" {
			userInput = strings.Trim(userInput, "\n")
			return userInput
		}
	}

}

// Returns the username for a user, given a connection string
func getUsername(conn net.Conn) string {
	for i := 0; i < len(clients); i++ {
		if clients[i].conn == conn {
			return clients[i].username
		}
	}

	return "No username available"
}

// Returns a client, given a connection string
func getClient(conn net.Conn) *client {
	for i := 0; i < len(clients); i++ {
		if clients[i].conn == conn {
			return clients[i]
		}
	}
	return nil
}

// Returns a client, given a username
func getClientByUsername(name string) *client {
	for i := 0; i < len(clients); i++ {
		if clients[i].username == name {
			return clients[i]
		}
	}
	return nil
}

// Returns a room, given a room name
func getRoom(roomName string) room {
	for i := 0; i < len(rooms); i++ {
		if rooms[i].roomName == roomName {
			return rooms[i]
		}
	}
	return room{nil, "", nil, nil}
}

// Removes a client from the connected clients list, given a connection string
func remove(conn net.Conn) {
	for i := 0; i < len(clients); i++ {
		if clients[i].conn == conn {
			clients[i] = clients[len(clients)-1]
			clients = clients[:len(clients)-1]
		}
	}
}

// Main function that handles the commands, decrypts and splits the message and selects the action based on the command
// First argument is the command
func handleUserConnection(conn net.Conn) {
	for {
		// Waits for input from the clients
		userInput, err := bufio.NewReader(conn).ReadString('\n')

		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				fmt.Println("Connection closed with one user")
			}
			break
		}

		// decrypt using the server private key, trim the newline and split into a string array
		userInput = decrypt(userInput, *serverPrivate)
		userInput = strings.Trim(userInput, "\r\n")
		args := strings.Split(userInput, " ")
		cmd := strings.TrimSpace(args[0])

		switch cmd {

		// Set the username of the client to a new username
		// Finds the user using the connection string and changes the username
		case cmdName:
			for i := 0; i < len(clients); i++ {
				if clients[i].conn == conn {
					newName := strings.TrimSpace(args[1])
					logText := getUsername(conn) + " changed username to " + newName
					writeLog(logText)
					clients[i].username = newName
					_, e := conn.Write([]byte("You changed your name to: " + newName))

					if e != nil {
						log.Fatalln("unable to write over client connection")
					}
				}
			}

		// Sends a DM to the specified user. Second element in the args array is the destination username
		// Uses that username as an identifier for sending the DM to the specified user
		case cmdMsg:
			destination := args[1]
			if userInput != "" {
				msg := strings.Join(args[2:], " ")
				logText := "'" + getUsername(conn) + "'" + " WHISPER ->" + "'" + destination + "'" + ":" + msg
				writeLog(logText)
				sendClientMessage(msg, destination, getUsername(conn))
			}

		// Sends the message to all of the clients connected to the same room as the sender
		case cmdBroadcast:
			var owner string
			for i := 0; i < len(clients); i++ {
				if clients[i].conn == conn {
					owner = clients[i].username
				}
			}
			if userInput != "" {
				msg := strings.Join(args[1:], " ")
				logText := "'" + getUsername(conn) + "'" + " BROADCAST ->" + getClientByUsername(owner).currentRoom + ":" + msg
				writeLog(logText)
				broadcastMessage(conn, msg, owner, getUsername(conn))
			}

		// Create a new room specified by the name, which is the second element of the args array
		case cmdCreateRoom:
			roomName := strings.TrimSpace(args[1])

			var destination string
			var newRoom room

			newRoom.roomName = roomName
			newRoom.roomAdmin = getClient(conn)
			newRoom.mods = append(newRoom.mods, getClient(conn))

			rooms = append(rooms, newRoom)
			msg := "Room created with name: " + roomName

			logText := "'" + getUsername(conn) + "'" + " CREATED A ROOM ->" + "'" + roomName + "'"
			writeLog(logText)

			for i := 0; i < len(clients); i++ {
				if clients[i].conn == conn {
					destination = clients[i].username
					clients[i].adminOf = append(clients[i].adminOf, newRoom)
					clients[i].modOf = append(clients[i].modOf, newRoom)
				}
			}

			sendClientMessage(msg, destination, "SERVER")

		// Join a room specified by the room name, which is the second element of the args array
		case cmdJoinRoom:
			var cli *client
			roomName := strings.TrimSpace(args[1])
			for i := 0; i < len(clients); i++ {
				if clients[i].conn == conn {
					cli = clients[i]
					clients[i].currentRoom = roomName

					logText := "'" + getUsername(conn) + "'" + " JOINED A ROOM ->" + "'" + roomName + "'"
					writeLog(logText)

					sendClientMessage("You joined a room: '"+yellow(roomName)+"'", getUsername(conn), "SERVER")
				}
			}

			for i := 0; i < len(rooms); i++ {
				if rooms[i].roomName == roomName {
					rooms[i].connectedClients = append(rooms[i].connectedClients, cli)
				}
			}

		// Qui the current room, no second arguments are required
		case cmdQuitRoom:
			for i := 0; i < len(clients); i++ {
				if clients[i].conn == conn {
					logText := "'" + getUsername(conn) + "'" + " QUITTED A ROOM ->" + "'" + getClient(conn).currentRoom + "'"
					writeLog(logText)

					sendClientMessage("You quitted the room: '"+yellow(getClient(conn).currentRoom)+"'", getUsername(conn), "SERVER")
					clients[i].currentRoom = ""

				}
			}

		// Promotes a member of the room, given that the promoter is the admin of the room
		// O(N)^^3 code im so bad...
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
									logText := "'" + toPromote + "'" + " PROMOTED TO A MOD BY->" + "'" + getClient(conn).username + "'" + " FOR ROOM -> " + "'" + rooms[i].roomName + "'"
									writeLog(logText)
								}
							}
							clients[i].modOf = append(clients[i].modOf, getRoom(currentRoom))
							sendClientMessage("You have been promoted to a moderator by: "+getUsername(conn), clients[i].username, "SERVER")
						}
					}

				}
			}

		// Broadcast message all in capitals, to all ussers
		case cmdShout:
			var owner string
			for i := 0; i < len(clients); i++ {
				if clients[i].conn == conn {
					owner = clients[i].username
				}
			}
			if userInput != "" {
				msg := strings.ToUpper(strings.Join(args[1:], " "))
				logText := "'" + getUsername(conn) + "'" + " SHOUT ->" + getClientByUsername(owner).currentRoom
				writeLog(logText)
				broadcastMessage(conn, msg, owner, getUsername(conn))
			}

		// Kicks a user from the room, given that the kicker is either an admin or a mod of the room
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
						logText := "'" + getUsername(conn) + "'" + " KICKED " + "'" + clients[i].username + "'" + " FROM ROOM ->" + "'" + clients[i].currentRoom + "'"
						writeLog(logText)
						clients[i].currentRoom = ""

						sendClientMessage("You have been kicked from '"+currentRoom+"' by: "+getUsername(conn), kicker.username, "SERVER")
					}
				}
			}

			if isMod(getRoom(currentRoom).mods, kicker.username) && !isMod(getRoom(currentRoom).mods, toKick) {
				logText := "'" + getUsername(conn) + "'" + " KICKED " + "'" + toKick + "'" + " FROM ROOM ->" + "'" + currentRoom + "'"
				writeLog(logText)

				getClientByUsername(toKick).currentRoom = ""
				sendClientMessage("You have been kicked from '"+currentRoom+"' by: "+getUsername(conn), toKick, "SERVER")

			}

		// Spams the message to the server 'N' times. Number of times to spam is the second element of the args array
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

				logText := "'" + getUsername(conn) + "'" + " SPAMMED " + msg + " " + strings.TrimSpace(args[1]) + " Times"
				writeLog(logText)
			}

		// Lists the active users or lists the active users in a room. If listing for room, a room name is required
		case cmdList:
			var activeUsers string
			if len(args) == 1 {
				for i := 0; i < len(clients); i++ {
					activeUsers += "'" + clients[i].username + "'" + " "
				}

				sendClientMessage(yellow(activeUsers), getUsername(conn), "SERVER: Active users are")
				checkErrorServer(err, "unable to write over client connection")

			} else if len(args) == 2 {
				for i := 0; i < len(clients); i++ {
					if clients[i].currentRoom == args[1] {
						activeUsers += "'" + clients[i].username + "'" + " "
					}
				}

				sendClientMessage(yellow(activeUsers), getUsername(conn), "SERVER: Active users in '"+getRoom(args[1]).roomName+"' are")
				checkErrorServer(err, "unable to write over client connection")
			}

		// Lists the active rooms for the server
		case cmdListRooms:
			var activeRooms string
			for i := 0; i < len(rooms); i++ {
				activeRooms += "'" + rooms[i].roomName + "'" + " "
			}
			sendClientMessage(yellow(activeRooms), getUsername(conn), "SERVER: Active rooms are")
			checkErrorServer(err, "unable to write over client connection")

		case cmdHelp:

		// Disconnects from the server
		case cmdExit:
			name := getUsername(conn)
			remove(conn)

			err := conn.Close()
			checkErrorServer(err, "")

			logText := "'" + name + "'" + " DISCONNECTED"
			writeLog(logText)
			fmt.Println(name + " disconnected")

		default:
			fmt.Println("No such command: ", cmd)
		}

	}
}

// Writes a message to the client using a connection. Destination connection string determined by destination username
func sendClientMessage(msg string, destination string, sender string) {
	for i := 0; i < len(clients); i++ {
		if clients[i].username == destination {
			cipherText := encrypt(blue(sender)+blue(": ")+msg, clients[i].public)

			_, err := clients[i].conn.Write([]byte(cipherText + "\n"))
			checkErrorServer(err, "unable to write over client connection")
		}
	}

}

// Sends message to all clients that are in the same room. Who to send is filtered by checking the current room of the user and each client's current room
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

// Checks whether the given user is a moderator for the room. The client array passed to this function is the mods array of a room object
func isMod(r []*client, user string) bool {
	for i := 0; i < len(r); i++ {
		if r[i].username == user {

			return true
		}
	}
	return false
}

// Writes to the log file for session logging
func writeLog(logText string) {
	fo, err := os.OpenFile(logFileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	checkErrorServer(err, "error opening log file")

	currentTime := time.Now()
	date := (currentTime.Format("\n[2006-01-02 15:04:0]"))

	_, err = fo.WriteString(date + "          " + logText)

	fo.Close()
	checkErrorServer(err, "")
}

func checkErrorServer(err error, errMsg string) {
	if err != nil {
		fmt.Println(errMsg + err.Error())
		os.Exit(0)
	}
}

// Main function that handles server connections in a loop
func RunServer() {
	fmt.Println("Server starting...")
	ln, err := net.Listen(PROTOCOL, PORT)
	checkErrorServer(err, "Error listening on port")

	// Generates a public and private key pair for the server
	private, err := rsa.GenerateKey(rand.Reader, 2048)
	checkErrorServer(err, " Unable to generate private key")

	serverPrivate = private
	serverPublic = serverPrivate.PublicKey

	defer ln.Close()

	// Creates a log file for session logging
	fo, err := os.Create(logFileName)
	checkErrorServer(err, "")

	fmt.Println(green("Server started on port "), cyan(PORT))

	// Logs the session start time when the server is started
	currentTime := time.Now()
	date := (currentTime.Format("[2006-01-02 15:04:0]"))
	l, err := fo.WriteString(date + "          Server started on port " + PORT + " successfully")

	if l == 0 {
		fmt.Println("Write error")
	}

	if err != nil {
		fmt.Println(err)
		fo.Close()
		return
	}

	// Main loop that accepts connections and sends them to a goroutine that continiously monitors the socket and handles the requests
	for {
		conn, err := ln.Accept()
		checkErrorServer(err, "Unable to connect to client: ")

		go newClient(conn)

	}
}
