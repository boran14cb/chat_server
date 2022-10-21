package main

import (
	"fmt"
	"net"
)

const (
	PORT string = ":8080"
)

func newClient(conn net.Conn) {
	fmt.Println("Client connected: ", conn.RemoteAddr().String())

	var cli *client
	cli.conn = conn
	cli.username = "anonymous"

	fmt.Println(cli.username)
}

func main() {
	fmt.Println("Server starting...")
	ln, err := net.Listen("tcp", PORT)

	if err != nil {
		fmt.Println("Unable to start server: ", err.Error())
	}

	defer ln.Close()
	fmt.Println("Server sterted on port ", PORT)
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Unable to connect to client: ", err.Error())
		}
		go newClient(conn)
	}
}
