package notification

import (
	"log"
	"msnserver/config"
	"msnserver/internal/commands"
	"net"
	"strings"
)

func StartNotificationServer() {
	log.Println("Starting MSN notification server...")

	config.LoadConfig()

	ln, err := net.Listen("tcp", config.Config.NotificationServer.ServerAddr+":"+config.Config.NotificationServer.ServerPort)
	if err != nil {
		log.Fatalln("Error starting server:", err)
	}

	defer ln.Close()

	log.Println("Listening on:", config.Config.NotificationServer.ServerAddr+":"+config.Config.NotificationServer.ServerPort)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Error accepting connection:", err)
			return
		}
		log.Println("Client connected:", conn.RemoteAddr())
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer func() {
		if err := conn.Close(); err != nil {
			log.Println("Error closing connection:", err)
		} else {
			log.Println("Client disconnected:", conn.RemoteAddr())
		}
	}()

	for {
		buffer := make([]byte, 1024)
		_, err := conn.Read(buffer)
		if err != nil {
			log.Println("Error:", err)
			return
		}

		data := string(buffer)
		log.Println("<<<", data)

		command, arguments, _ := strings.Cut(data, " ")

		switch command {
		case "VER":
			err := commands.HandleVER(conn, arguments)
			if err != nil {
				log.Println("Error:", err)
				return
			}
		case "INF":
			err := commands.HandleINF(conn, arguments)
			if err != nil {
				log.Println("Error:", err)
				return
			}
		case "USR":
			transactionID, _, err := commands.HandleReceiveUSR(conn, arguments)
			if err != nil {
				log.Println("Error:", err)
				return
			}
		default:
			log.Println("Unknown command:", command)
			return
		}
	}
}
