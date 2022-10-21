package main

import (
	"bufio"
	"fmt"
	"net"
)

type client struct {
	conn     net.Conn
	username string
}

func main() {
	fmt.Println("Client Starting...")
	conn, err := net.Dial("tcp", ":8080")
	if err != nil {
		fmt.Println("Unable to connect to server: ", err.Error())
	}

	status, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		fmt.Println("Unable to read input from the server ", err.Error())
	}
	fmt.Println(status)
}
