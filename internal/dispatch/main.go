package dispatch

import (
	"log"
	"msnserver/config"
	"msnserver/pkg/commands"
	"net"
	"strings"

	"gorm.io/gorm"
)

type DispatchServer struct {
	db     *gorm.DB
	config *config.MSNServerConfiguration
}

func NewDispatchServer(db *gorm.DB, c *config.MSNServerConfiguration) *DispatchServer {
	return &DispatchServer{
		db:     db,
		config: c,
	}
}

func (ds *DispatchServer) Start() {
	ln, err := net.Listen("tcp", ds.config.DispatchServer.ServerAddr+":"+ds.config.DispatchServer.ServerPort)
	if err != nil {
		log.Fatalln("Error starting server:", err)
	}

	defer ln.Close()

	log.Println("Listening on:", ds.config.DispatchServer.ServerAddr+":"+ds.config.DispatchServer.ServerPort)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}
		log.Println("Client connected:", conn.RemoteAddr())
		go ds.handleConnection(conn)
	}
}

func (ds *DispatchServer) handleConnection(conn net.Conn) {
	c := &Client{
		id:       conn.RemoteAddr().String(),
		conn:     conn,
		sendChan: make(chan string),
		session:  &commands.Session{},
	}

	defer func() {
		conn.Close()
		log.Println("Client disconnected:", conn.RemoteAddr())
	}()

	go c.sendHandler()

	for {
		buffer := make([]byte, 1024)
		_, err := conn.Read(buffer)
		if err != nil {
			return
		}

		go func() {
			data := string(buffer)
			log.Println("<<<", data)

			command, arguments, found := strings.Cut(data, " ")
			if !found {
				command, _, _ = strings.Cut(data, "\r\n")
			}

			switch command {
			case "VER":
				if err := commands.HandleVER(c.sendChan, arguments); err != nil {
					log.Println("Error:", err)
					close(c.sendChan)
				}

			case "INF":
				if err := commands.HandleINF(c.sendChan, arguments); err != nil {
					log.Println("Error:", err)
					close(c.sendChan)
				}

			case "USR":
				tid, err := commands.HandleReceiveUSR(c.session, arguments)
				if err != nil {
					log.Println("Error:", err)
					close(c.sendChan)
				}

				commands.HandleXFR(c.sendChan, ds.config.DispatchServer, tid)
				close(c.sendChan)

			case "OUT":
				commands.HandleOUT(c.sendChan)
				close(c.sendChan)

			default:
				log.Println("Unknown command:", command)
				close(c.sendChan)
			}
		}()
	}
}
