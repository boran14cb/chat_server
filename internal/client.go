package internal

import (
	"bufio"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"math/big"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var publicKey rsa.PublicKey
var privateKey *rsa.PrivateKey
var ServerPublicKey rsa.PublicKey
var usrname string

// Monitors the socket continiosly for new messages
func monitorSocket(conn net.Conn) {
	defer wg.Done()
	for {
		status, err := bufio.NewReader(conn).ReadString('\n')
		checkError(err, "Unable to read input from the server ")

		status = decrypt(status, *privateKey)
		status = strings.Trim(status, "\r\n")
		status = strings.Trim(status, ">")

		printAboveLine(status)

	}
}

// Gets input from the user and sends it to the server after encrypting
func sendMessage(conn net.Conn) {
	for {
		fmt.Printf("\033[2K\r%s", purple(usrname+"> "))

		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()

		err := scanner.Err()
		checkError(err, "")

		userInput := strings.Trim(scanner.Text(), "\r\n")
		args := strings.Split(userInput, " ")

		switch args[0] {

		case "/help":
			commandList := [13][3]string{USAGE, NAME, MSG, BROADCAST, SPAM, SHOUT, CREATE, JOIN, KICK, PROMOTE, QUIT, HELP, LIST}
			for i := range commandList {
				fmt.Printf("%5s%5s%5s\n", commandList[i][0], commandList[i][1], commandList[i][2])
			}

		case "/name":
			usrname = args[1]

			msg := encrypt(scanner.Text(), ServerPublicKey)

			_, err = conn.Write([]byte(msg + "\n"))
			checkError(err, "")

		default:
			msg := encrypt(scanner.Text(), ServerPublicKey)

			_, err = conn.Write([]byte(msg + "\n"))
			checkError(err, "")
		}
	}
}

// Sets the username for the user for this session
func setusrname(conn net.Conn) {
	fmt.Print(blue("input username: "))
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()

	err := scanner.Err()
	checkError(err, "")

	pKey, err := rsa.GenerateKey(rand.Reader, 2048)
	checkError(err, "")

	privateKey = pKey
	publicKey = privateKey.PublicKey

	usrname = strings.Trim(scanner.Text(), "\r\n")
	_, err = conn.Write([]byte(scanner.Text() + "\n"))

	checkError(err, "")
	time.Sleep(100 * time.Millisecond)

	_, err = conn.Write([]byte(publicKey.N.String() + " " + strconv.Itoa(publicKey.E) + "\n"))
	checkError(err, "")

	ServerPublicKey = setPublicKeyServer(conn)
}

// The first message received from the server is the public key of the server for encrypting the messages
// So only server can decrypt it by using the server private key
func setPublicKeyServer(conn net.Conn) rsa.PublicKey {
	for {
		userInput, err := bufio.NewReader(conn).ReadString('\n')

		checkError(err, "")

		if userInput != "" {
			userInput = strings.Trim(userInput, "\r\n")
			args := strings.Split(userInput, " ")
			N := strings.TrimSpace(args[0])
			E := strings.TrimSpace(args[1])

			var pKey rsa.PublicKey
			i := new(big.Int)
			_, err := fmt.Sscan(N, i)

			checkError(err, "error scanning value: ")

			pKey.N = i
			pKey.E, err = strconv.Atoi(E)

			checkError(err, "error scanning value: ")

			return pKey
		}
	}
}

// Prints the received message 1 line above the current line
func printAboveLine(s string) {
	fmt.Print("\0337")
	fmt.Print("\033[A")
	fmt.Print("\033[999D")
	fmt.Print("\033[S")
	fmt.Print("\033[L")
	fmt.Println(s)
	fmt.Print("\0338")
	fmt.Printf("\033[2K\r%s", purple(usrname+"> "))
}

// Checks and prints the errors
func checkError(err error, errMsg string) {
	if err != nil {
		fmt.Println(errMsg + err.Error())
		os.Exit(0)
	}
}

// Main function that starts the goroutines and connects to the port by dialing in
func RunClient() {
	wg := sync.WaitGroup{}

	fmt.Println("Client Starting...")

	// Dial in using a protocol and a port
	conn, err := net.Dial(PROTOCOL, PORT)

	if err != nil {
		fmt.Println("Unable to connect to server: ", err.Error())
		os.Exit(0)
	}

	setusrname(conn)

	wg.Add(1)
	go monitorSocket(conn)
	go sendMessage(conn)

	wg.Wait()
}
