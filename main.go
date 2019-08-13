package main

import (
	"fmt"
	"net"
	"strconv"
)

// from https://jameshfisher.com/2017/04/18/golang-tcp-server/

// PORT which port to serve on
const PORT = 8080
const buff = 128

var newConns chan net.Conn
var deadConns chan net.Conn
var conns map[net.Conn]bool
var published chan []byte

func setup() {
	// connections to pick up or drop
	newConns = make(chan net.Conn, buff)
	deadConns = make(chan net.Conn, buff)

	// messages to send
	published = make(chan []byte, buff)

	// existing connections
	conns = make(map[net.Conn]bool)
}

func main() {

	setup()

	// listen
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(PORT))
	if err != nil {
		panic(err)
	}
	fmt.Println("addr", listener.Addr())

	defer listener.Close()

	// accept new connections
	// add them to newConns
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				panic(err)
			}
			newConns <- conn
		}
	}()

	// loop forever for
	// - newConnections
	// - deadConnections
	// - publish received messages to other connections
	for {
		select {
		case conn := <-newConns:
			conns[conn] = true
			fmt.Println("new conn", conn)
			go func() {

				// is 1024 a big enough buffer?
				buf := make([]byte, 1024)

				// block to read from the new connection
				for {
					nbyte, err := conn.Read(buf)
					if err != nil {
						deadConns <- conn
						break
					} else {
						fragment := make([]byte, nbyte)
						copy(fragment, buf[:nbyte])
						published <- fragment
						fmt.Println("new msg", fragment)
					}
				}
			}()
		case deadConn := <-deadConns:
			_ = deadConn.Close()
			delete(conns, deadConn)
			fmt.Println("lost conn", deadConn)
		case publish := <-published:
			// for conn, _ := range conns {
			fmt.Println("pub", publish)
			for conn := range conns {
				go func(conn net.Conn) {
					totalWritten := 0
					for totalWritten < len(publish) {
						writtenThisCall, err := conn.Write(publish[totalWritten:])
						if err != nil {
							deadConns <- conn
							break
						}
						totalWritten += writtenThisCall
					}
				}(conn)
			}
		}
	}
}
