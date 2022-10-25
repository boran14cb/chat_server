package main

import (
	"bufio"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
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

func monitorSocket(conn net.Conn, wg sync.WaitGroup) {
	defer wg.Done()
	for {
		status, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			fmt.Println("Unable to read input from the server ", err.Error())
			os.Exit(0)
		}

		//fmt.Println(status)
		status = decryptClient(status, *privateKey)
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

		msg := encryptClient(scanner.Text(), ServerPublicKey)
		//fmt.Println(msg)
		_, err = conn.Write([]byte(msg + "\n"))
	}
}

func setusrname(conn net.Conn) {
	fmt.Print("input username: ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()

	err := scanner.Err()
	if err != nil {
		log.Fatal(err)
		os.Exit(0)
	}

	pKey, err := rsa.GenerateKey(rand.Reader, 2048)

	privateKey = pKey
	publicKey = privateKey.PublicKey

	if err != nil {
		log.Fatalln(err)
	}

	//fmt.Println(scanner.Text())
	_, err = conn.Write([]byte(scanner.Text() + "\n"))
	time.Sleep(100 * time.Millisecond)
	//fmt.Println(publicKey)
	_, err = conn.Write([]byte(publicKey.N.String() + " " + strconv.Itoa(publicKey.E) + "\n"))

	ServerPublicKey = setPublicKeyServer(conn)
	//fmt.Println("PublicKey: ", ServerPublicKey)
}

func setPublicKeyServer(conn net.Conn) rsa.PublicKey {
	for {
		userInput, err := bufio.NewReader(conn).ReadString('\n')

		if err != nil {
			fmt.Println(err.Error())
			os.Exit(0)
		}

		if userInput != "" {
			userInput = strings.Trim(userInput, "\r\n")
			args := strings.Split(userInput, " ")
			N := strings.TrimSpace(args[0])
			E := strings.TrimSpace(args[1])

			var pKey rsa.PublicKey
			i := new(big.Int)
			_, err := fmt.Sscan(N, i)

			if err != nil {
				log.Println("error scanning value:", err)
			}

			pKey.N = i
			pKey.E, err = strconv.Atoi(E)

			if err != nil {
				log.Println("error scanning value:", err)
			}

			return pKey
		}
	}
}

func encryptClient(msg string, key rsa.PublicKey) string {

	label := []byte("OAEP Encrypted")
	rng := rand.Reader

	// * using OAEP algorithm to make it more secure
	// * using sha256
	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rng, &key, []byte(msg), label)
	// check for errors
	if err != nil {
		log.Fatalln("unable to encrypt")
	}

	return base64.StdEncoding.EncodeToString(ciphertext)
}

// function to decrypt message to be received
func decryptClient(cipherText string, key rsa.PrivateKey) string {

	ct, _ := base64.StdEncoding.DecodeString(cipherText)
	label := []byte("OAEP Encrypted")
	rng := rand.Reader

	// decrypting based on same parameters as encryption
	plaintext, err := rsa.DecryptOAEP(sha256.New(), rng, &key, ct, label)
	// check for errors
	if err != nil {
		log.Fatalln(err)
	}
	return string(plaintext)
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
