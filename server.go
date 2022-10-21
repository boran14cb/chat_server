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
func handleUserConnection(conn net.Conn, wg *sync.WaitGroup) {

	for {
		userInput, err := bufio.NewReader(conn).ReadString('\n')

		userInput = strings.Trim(userInput, "\r\n")
		args := strings.Split(userInput, " ")
		destination := strings.TrimSpace(args[0])
		msg := strings.TrimSpace(args[1])
		fmt.Println(args)

		if err != nil {
			fmt.Println(err.Error())
			os.Exit(0)
		}

		if userInput != "" {
			sendClientMessage(conn, msg, destination)
		}
	}

}

func sendClientMessage(conn net.Conn, msg string, destination string) {

	for i := 0; i < len(clients); i++ {
		fmt.Println(clients[i].username)
		if clients[i].username == destination {
			_, e := clients[i].conn.Write([]byte(msg + "\n"))

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
