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
	cmdName string = "/name"
	cmdMsg  string = "/msg"
	cmdQuit string = "/quit"
	cmdHelp string = "/help"
	cmdList string = "/list"
)

type client struct {
	conn     net.Conn
	username string
}

var clients []*client

func newClient(conn net.Conn, wg *sync.WaitGroup) {
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

func setUsername(conn net.Conn, wg *sync.WaitGroup) string {

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

func handleUserConnection(conn net.Conn, wg *sync.WaitGroup) {

	for {
		userInput, err := bufio.NewReader(conn).ReadString('\n')

		if err != nil {
			fmt.Println(err.Error())
			os.Exit(0)
		}

		userInput = strings.Trim(userInput, "\r\n")
		args := strings.Split(userInput, " ")
		cmd := strings.TrimSpace(args[0])
		msg := strings.Join(args[1:], " ")

		switch cmd {

		case cmdName:
			for i := 0; i < len(clients); i++ {
				if clients[i].conn == conn {
					clients[i].username = msg
				}
			}

		case cmdMsg:
			destination := args[1]
			if userInput != "" {
				sendClientMessage(conn, msg, destination, getUsername(conn))
			}
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

		case cmdQuit:
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

func main() {
	wg := sync.WaitGroup{}
	wg1 := &wg
	fmt.Println("Server starting...")
	ln, err := net.Listen(PROTOCOL, PORT)

	if err != nil {
		fmt.Println("Unable to start server: ", err.Error())
		os.Exit(0)
	}

	defer ln.Close()
	fmt.Println("Server started on port ", PORT)
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Unable to connect to client: ", err.Error())
			os.Exit(0)
		}

		go newClient(conn, wg1)

		wg.Wait()
	}
}
