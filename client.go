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
			os.Exit(0)
		}
		status = strings.Trim(status, "\r\n")
		status = strings.Trim(status, ">")
		fmt.Println("\n" + status)
		fmt.Print(">")
	}
}

func sendMessage(conn net.Conn, wg sync.WaitGroup) {
	for {
		fmt.Print(">")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		err := scanner.Err()
		if err != nil {
			log.Fatal(err)
			os.Exit(0)
		}

		_, err = conn.Write([]byte(scanner.Text() + "\n"))
	}
}

func setusrname(conn net.Conn) {
	fmt.Println("input username:")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()

	err := scanner.Err()
	if err != nil {
		log.Fatal(err)
		os.Exit(0)
	}

	_, err = conn.Write([]byte(scanner.Text() + "\n"))
}
func main() {
	wg := sync.WaitGroup{}

	fmt.Println("Client Starting...")

	conn, err := net.Dial("tcp", ":8080")

	if err != nil {
		fmt.Println("Unable to connect to server: ", err.Error())
		os.Exit(0)
	}

	setusrname(conn)

	wg.Add(1)
	go monitorSocket(conn, wg)
	go sendMessage(conn, wg)

	wg.Wait()
}
