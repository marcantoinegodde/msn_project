package commands

import (
	"errors"
	"fmt"
	"log"
	"msnserver/pkg/clients"
	"msnserver/pkg/database"
	"slices"
	"strings"

	"gorm.io/gorm"
)

var statusCodes = []string{"NLN", "FLN", "HDN", "IDL", "AWY", "BSY", "BRB", "PHN", "LUN"}

func HandleCHG(db *gorm.DB, clients map[string]*clients.Client, c *clients.Client, args string) error {
	args, _, _ = strings.Cut(args, "\r\n")
	transactionID, args, err := parseTransactionID(args)
	if err != nil {
		return err
	}
	args, _, _ = strings.Cut(args, " ") // Remove the trailing space sent for this command

	if !c.Session.Authenticated {
		SendError(c.SendChan, transactionID, ERR_NOT_LOGGED_IN)
		return errors.New("not logged in")
	}

	if !slices.Contains(statusCodes, args) {
		return fmt.Errorf("invalid status code: %s", args)
	}

	// Perform nested preloading to load users lists of contacts on user's forward list
	var user database.User
	query := db.Preload("ForwardList.ForwardList").Preload("ForwardList.AllowList").Preload("ForwardList.BlockList").Preload("ReverseList").First(&user, "email = ?", c.Session.Email)
	if errors.Is(query.Error, gorm.ErrRecordNotFound) {
		return errors.New("user not found")
	} else if query.Error != nil {
		return query.Error
	}

	user.Status = args
	query = db.Save(&user)
	if query.Error != nil {
		return query.Error
	}

	res := fmt.Sprintf("CHG %s %s\r\n", transactionID, user.Status)
	c.SendChan <- res

	// Receive ILN on first CHG
	if !c.Session.InitialPresenceNotification {
		c.Session.InitialPresenceNotification = true

		for _, contact := range user.ForwardList {
			// Skip contacts that are offline or hidden
			if contact.Status == "FLN" || contact.Status == "HDN" {
				continue
			}

			// Skip contacts that have the user on their block list
			if isMember(contact.BlockList, &user) {
				continue
			}

			// Skip contacts in BL mode that don't have the user on their allow list
			if contact.Blp == "BL" && !isMember(contact.AllowList, &user) {
				continue
			}

			// Send initial presence notification
			HandleSendILN(c.SendChan, transactionID, contact.Status, contact.Email, contact.DisplayName)
		}
	}

	// Inform followers (RL) of the status change
	if user.Status == "HDN" {
		if err := HandleBatchFLN(db, clients, c); err != nil {
			log.Println("Error:", err)
		}
	} else {
		if err := HandleBatchNLN(db, clients, c); err != nil {
			log.Println("Error:", err)
		}
	}

	return nil
}
