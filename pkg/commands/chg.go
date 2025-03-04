package commands

import (
	"errors"
	"fmt"
	"log"
	"msnserver/pkg/clients"
	"msnserver/pkg/database"
	"strings"
	"sync"

	"gorm.io/gorm"
)

func HandleCHG(db *gorm.DB, m *sync.Mutex, clients map[string]*clients.Client, c *clients.Client, args string) error {
	args, _, _ = strings.Cut(args, "\r\n")
	tid, args, err := parseTransactionID(args)
	if err != nil {
		return err
	}
	args, _, _ = strings.Cut(args, " ") // Remove the trailing space sent for this command

	if !c.Session.Authenticated {
		SendError(c, tid, ERR_NOT_LOGGED_IN)
		return errors.New("not logged in")
	}

	splitArguments := strings.Fields(args)
	if len(splitArguments) != 1 {
		return errors.New("invalid transaction")
	}

	// User isn't allowed to change their status to FLN
	status := database.Status(splitArguments[0])
	switch status {
	case database.NLN, database.HDN, database.IDL, database.AWY, database.BSY, database.BRB, database.PHN, database.LUN:
		break
	default:
		SendError(c, tid, ERR_INVALID_PARAMETER)
		return nil
	}

	// Perform nested preloading to load users lists of contacts on user's forward list
	var user database.User
	query := db.Preload("ForwardList.AllowList").Preload("ForwardList.BlockList").First(&user, "email = ?", c.Session.Email)
	if errors.Is(query.Error, gorm.ErrRecordNotFound) {
		return errors.New("user not found")
	} else if query.Error != nil {
		return query.Error
	}

	user.Status = status
	query = db.Save(&user)
	if query.Error != nil {
		return query.Error
	}

	res := fmt.Sprintf("CHG %d %s\r\n", tid, user.Status)
	c.Send(res)

	// Receive ILN on first CHG
	if !c.Session.InitialPresenceNotification {
		c.Session.InitialPresenceNotification = true

		for _, contact := range user.ForwardList {
			// Skip contacts that are offline or hidden
			if contact.Status == database.FLN || contact.Status == database.HDN {
				continue
			}

			// Skip contacts that have the user on their block list
			if isMember(contact.BlockList, &user) {
				continue
			}

			// Skip contacts in BL mode that don't have the user on their allow list
			if contact.Blp == database.BL && !isMember(contact.AllowList, &user) {
				continue
			}

			// Send initial presence notification
			HandleSendILN(c, tid, contact.Status, contact.Email, contact.DisplayName)
		}
	}

	// Inform followers (RL) of the status change
	if user.Status == database.HDN {
		if err := HandleBatchFLN(db, m, clients, c); err != nil {
			log.Println("Error:", err)
		}
	} else {
		if err := HandleBatchNLN(db, m, clients, c); err != nil {
			log.Println("Error:", err)
		}
	}

	return nil
}
