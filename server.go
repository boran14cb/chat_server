package main

import (
	"fmt"
	"log"
	"net"
	"sync"
)

const (
	PORT string = ":8080"
)

type client struct {
	conn     net.Conn
	username string
}

var clients []*client

func newClient(conn net.Conn) {
	fmt.Println("Client connected: ", conn.RemoteAddr().String())

	cli := &client{
		conn:     conn,
		username: "Anonymous",
	}

	clients = append(clients, cli)
	fmt.Println(cli.username)
}

func handleUserConnection(conn net.Conn, wg sync.WaitGroup) {

	// userInput, err := bufio.NewReader(conn).ReadString('\n')
	// if err != nil {
	// 	fmt.Println(err.Error())
	// }
	// fmt.Println(userInput)
	_, e := clients[0].conn.Write([]byte("Hello!" + "\n"))

	if e != nil {
		log.Fatalln("unable to write over client connection")
	}
}

func main() {
	wg := sync.WaitGroup{}
	fmt.Println("Server starting...")
	ln, err := net.Listen("tcp", PORT)

	if err != nil {
		fmt.Println("Unable to start server: ", err.Error())
	}

	defer ln.Close()
	fmt.Println("Server started on port ", PORT)
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Unable to connect to client: ", err.Error())
		}
		newClient(conn)
		handleUserConnection(conn, wg)

		wg.Wait()
	}
}
