package dispatch

import (
	"log"
	"msnserver/config"
	"msnserver/pkg/clients"
	"msnserver/pkg/commands"
	"net"
	"strings"
)

type DispatchServer struct {
	config *config.MSNServerConfiguration
}

func NewDispatchServer(c *config.MSNServerConfiguration) *DispatchServer {
	return &DispatchServer{
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
	c := &clients.Client{
		Id:       conn.RemoteAddr().String(),
		Conn:     conn,
		SendChan: make(chan string),
		Session:  &clients.Session{},
	}

	defer func() {
		close(c.SendChan)
		c.Wg.Wait()
		conn.Close()
		log.Println("Client disconnected:", conn.RemoteAddr())
	}()

	c.Wg.Add(1)
	go c.SendHandler()

	for {
		buffer := make([]byte, 1024)
		_, err := conn.Read(buffer)
		if err != nil {
			return
		}

		data := string(buffer)
		log.Printf("[%s] <<< %s\n", c.Id, data)

		command, arguments, found := strings.Cut(data, " ")
		if !found {
			command, _, _ = strings.Cut(data, "\r\n")
		}

		switch command {
		case "VER":
			if err := commands.HandleVER(c.SendChan, arguments); err != nil {
				log.Println("Error:", err)
				return
			}

		case "INF":
			if err := commands.HandleINF(c.SendChan, arguments); err != nil {
				log.Println("Error:", err)
				return
			}

		case "USR":
			tid, err := commands.HandleUSRDispatch(arguments)
			if err != nil {
				log.Println("Error:", err)
				return
			}

			commands.HandleXFR(c.SendChan, ds.config.DispatchServer, tid)
			return

		case "OUT":
			commands.HandleOUT(c.SendChan)
			return

		default:
			log.Println("Unknown command:", command)
			return
		}
	}
}
