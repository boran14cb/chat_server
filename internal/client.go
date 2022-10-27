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
var clientRoom string
var ptr *string

func monitorSocket(conn net.Conn, wg sync.WaitGroup) {
	defer wg.Done()
	for {
		status, err := bufio.NewReader(conn).ReadString('\n')
		checkError(err, "Unable to read input from the server ")

		status = decrypt(status, *privateKey)
		status = strings.Trim(status, "\r\n")
		status = strings.Trim(status, ">")

		fmt.Println("\n" + status)
		fmt.Print(purple(usrname + "> "))
	}
}

func sendMessage(conn net.Conn, wg sync.WaitGroup) {
	for {
		fmt.Println(clientRoom)
		fmt.Print(purple(usrname + "> "))
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()

		err := scanner.Err()
		checkError(err, "")

		userInput := strings.Trim(scanner.Text(), "\r\n")
		args := strings.Split(userInput, " ")
		if args[0] == "/help" {
			commandList := [13][3]string{USAGE, NAME, MSG, BROADCAST, SPAM, SHOUT, CREATE, JOIN, KICK, MSG, QUIT, HELP, LIST}
			for i := range commandList {
				fmt.Printf("%5s%5s%5s\n", commandList[i][0], commandList[i][1], commandList[i][2])

			}
		} else {

			msg := encrypt(scanner.Text(), ServerPublicKey)

			_, err = conn.Write([]byte(msg + "\n"))
			checkError(err, "")
		}
	}
}

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

func getClientRoom(s string) string {
	if s != "" {
		fmt.Println("Works")
		return s
	}
	return "general"
}

func checkError(err error, errMsg string) {
	if err != nil {
		fmt.Println(errMsg + err.Error())
		os.Exit(0)
	}
}

func RunClient() {
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
