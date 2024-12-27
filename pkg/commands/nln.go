package commands

import (
	"fmt"
	"msnserver/pkg/clients"
	"msnserver/pkg/database"

	"gorm.io/gorm"
)

func HandleSendNLN(db *gorm.DB, clients map[string]*clients.Client, s *clients.Session) error {
	var user database.User
	query := db.Preload("ReverseList").First(&user, "email = ?", s.Email)
	if query.Error != nil {
		return query.Error
	}

	for _, contact := range user.ReverseList {
		if contact.Status == "FLN" {
			continue
		}

		if clients[contact.Email] == nil {
			continue
		}

		res := fmt.Sprintf("NLN %s %s %s\r\n", user.Status, user.Email, user.Name)
		clients[contact.Email].SendChan <- res
	}

	return nil
}
