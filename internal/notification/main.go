package notification

import (
	"log"
	"msnserver/config"
	"msnserver/pkg/clients"
	"msnserver/pkg/commands"
	"msnserver/pkg/database"
	"net"
	"strings"
	"sync"

	"gorm.io/gorm"
)

type NotificationServer struct {
	db      *gorm.DB
	config  *config.MSNServerConfiguration
	m       *sync.Mutex
	clients map[string]*clients.Client
}

func NewNotificationServer(db *gorm.DB, c *config.MSNServerConfiguration) *NotificationServer {
	return &NotificationServer{
		db:      db,
		config:  c,
		m:       &sync.Mutex{},
		clients: map[string]*clients.Client{},
	}
}

func (ns *NotificationServer) Start() {
	ln, err := net.Listen("tcp", ns.config.NotificationServer.ServerAddr+":"+ns.config.NotificationServer.ServerPort)
	if err != nil {
		log.Fatalln("Error starting server:", err)
	}

	defer ln.Close()

	log.Println("Listening on:", ns.config.NotificationServer.ServerAddr+":"+ns.config.NotificationServer.ServerPort)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}
		log.Println("Client connected:", conn.RemoteAddr())
		go ns.handleConnection(conn)
	}
}

func (ns *NotificationServer) handleConnection(conn net.Conn) {
	c := &clients.Client{
		Id:       conn.RemoteAddr().String(),
		Conn:     conn,
		SendChan: make(chan string),
		Session:  &clients.Session{},
	}

	defer func() {
		if c.Session.Email != "" {
			var user database.User
			query := ns.db.First(&user, "email = ?", c.Session.Email)
			if query.Error == nil {
				user.Status = "FLN"
				ns.db.Save(&user)
			}

			if err := commands.HandleBatchFLN(ns.db, ns.m, ns.clients, c); err != nil {
				log.Println("Error:", err)
			}

			ns.m.Lock()
			delete(ns.clients, c.Session.Email)
			ns.m.Unlock()
		}

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
			log.Println("Error reading from connection:", err)
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
			if err := commands.HandleVER(c, arguments); err != nil {
				log.Println("Error:", err)
				return
			}

		case "INF":
			if err := commands.HandleINF(c, arguments); err != nil {
				log.Println("Error:", err)
				return
			}

		case "USR":
			if err := commands.HandleUSR(ns.db, c, arguments); err != nil {
				log.Println("Error:", err)
				return
			}

			ns.m.Lock()
			ns.clients[c.Session.Email] = c
			ns.m.Unlock()

		case "SYN":
			if err := commands.HandleSYN(ns.db, c, arguments); err != nil {
				log.Println("Error:", err)
				return
			}

		case "CHG":
			err := commands.HandleCHG(ns.db, ns.m, ns.clients, c, arguments)
			if err != nil {
				log.Println("Error:", err)
				return
			}

		case "CVR":
			if err := commands.HandleCVR(c, arguments); err != nil {
				log.Println("Error:", err)
				return
			}

		case "GTC":
			if err := commands.HandleGTC(ns.db, c, arguments); err != nil {
				log.Println("Error:", err)
				return
			}

		case "BLP":
			if err := commands.HandleBLP(ns.db, c, arguments); err != nil {
				log.Println("Error:", err)
				return
			}

		case "ADD":
			if err := commands.HandleADD(ns.db, ns.m, ns.clients, c, arguments); err != nil {
				log.Println("Error:", err)
				return
			}

		case "REM":
			if err := commands.HandleREM(ns.db, ns.m, ns.clients, c, arguments); err != nil {
				log.Println("Error:", err)
				return
			}

		case "REA":
			if err := commands.HandleREA(ns.db, ns.m, ns.clients, c, arguments); err != nil {
				log.Println("Error:", err)
				return
			}

		case "FND":
			if err := commands.HandleFND(ns.db, c, arguments); err != nil {
				log.Println("Error:", err)
				return
			}

		case "SND":
			if err := commands.HandleSND(c, arguments); err != nil {
				log.Println("Error:", err)
				return
			}

		case "URL":
			if err := commands.HandleURL(c, arguments); err != nil {
				log.Println("Error:", err)
				return
			}

		case "OUT":
			commands.HandleOUT(c)
			return

		default:
			log.Println("Unknown command:", command)
			return
		}
	}
}
