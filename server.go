package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
)

const (
	PORT     string = ":8080"
	PROTOCOL string = "tcp"
	USAGE    string = "Usage: /<Command> arguments \n"
	NAME     string = "/name <new_name> (Sets new username) \n"
	MSG      string = "/msg <receiver_username> <message> (Sends a DM) \n"
	QUIT     string = "/quit (Quits the server) \n"
	HELP     string = "/help (Lists all commands) \n"
	LIST     string = "/list (Lists active users) \n"
)

const (
	cmdName       string = "/name" //Done
	cmdMsg        string = "/msg"  //Done
	cmdBroadcast  string = "/all"  //Done
	cmdCreateRoom string = "/create"
	cmdJoinRoom   string = "/join"
	cmdQuitRoom   string = "/quit"
	cmdSpam       string = "/spam"
	cmdShout      string = "/shout"
	cmdKick       string = "/kick"
	cmdExit       string = "/exit"
	cmdHelp       string = "/help" //Fix newline problem
	cmdList       string = "/list"
)

type client struct {
	conn        net.Conn
	username    string
	currentRoom string
}

type room struct {
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
	cli := &client{
		conn:     conn,
		username: name,
	}

	clients = append(clients, cli)
	fmt.Println(cli.username)
	fmt.Println(clients)

	handleUserConnection(conn, wg)
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

func handleUserConnection(conn net.Conn, wg sync.WaitGroup) {

	for {
		userInput, err := bufio.NewReader(conn).ReadString('\n')

		if err != nil {
			fmt.Println(err.Error())
			os.Exit(0)
		}

		userInput = strings.Trim(userInput, "\r\n")
		args := strings.Split(userInput, " ")
		cmd := strings.TrimSpace(args[0])
		//msg := strings.Join(args[2:], " ")

		fmt.Println(cmd)

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
			rooms = append(rooms, newRoom)

			//time.Sleep(1 * time.Second)
			msg := "Room created with name: " + roomName

			var destination string
			for i := 0; i < len(clients); i++ {
				if clients[i].conn == conn {
					destination = clients[i].username
				}
			}

			sendClientMessage(conn, msg, destination, "SERVER")
			//newRoom.connectedClients clients = append(clients, cli)
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
		case cmdKick:
		case cmdSpam:

		case cmdList:
			fmt.Println("cmdList: ", cmd)

		case cmdHelp:
			commandList := [6]string{USAGE, NAME, MSG, QUIT, HELP, LIST}
			for i := 0; i < 6; i++ {
				_, e := conn.Write([]byte(commandList[i]))

				if e != nil {
					log.Fatalln("unable to write over client connection")
				}
			}

		case cmdExit:
			fmt.Println("cmdQuit: ", cmd)

		default:
			fmt.Println("No such command: ", cmd)
		}

	}

}

func sendClientMessage(conn net.Conn, msg string, destination string, sender string) {

	for i := 0; i < len(clients); i++ {
		fmt.Println(clients[i].username)
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
		fmt.Println(clients[i].username)
		if clients[i].username != owner && clients[i].currentRoom == senderRoom {
			_, e := clients[i].conn.Write([]byte(sender + ": " + msg + "\n"))

			if e != nil {
				log.Fatalln("unable to write over client connection")
			}
		}
	}

}

func main() {
	//wg = sync.WaitGroup{}
	//wg1 := &wg
	fmt.Println("Server starting...")
	ln, err := net.Listen(PROTOCOL, PORT)

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
