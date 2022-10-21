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

func monitorSocket(conn net.Conn, wg sync.WaitGroup) {
	defer wg.Done()
	for {
		status, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			fmt.Println("Unable to read input from the server ", err.Error())
		}
		status = strings.Trim(status, "\r\n")
		fmt.Println(status)
	}
}

func sendMessage(conn net.Conn, wg sync.WaitGroup) {
	fmt.Println("input text:")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	err := scanner.Err()
	if err != nil {
		log.Fatal(err)
	}

	_, err = conn.Write([]byte(scanner.Text()))
}

func main() {
	wg := sync.WaitGroup{}
	fmt.Println("Client Starting...")
	conn, err := net.Dial("tcp", ":8080")

	if err != nil {
		fmt.Println("Unable to connect to server: ", err.Error())
	}

	wg.Add(1)
	go monitorSocket(conn, wg)
	//go sendMessage(conn, wg)

	wg.Wait()
}
