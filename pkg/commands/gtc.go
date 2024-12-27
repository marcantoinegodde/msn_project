package commands

import (
	"errors"
	"fmt"
	"msnserver/pkg/clients"
	"msnserver/pkg/database"
	"slices"
	"strings"

	"gorm.io/gorm"
)

var gtcMode = []string{"A", "N"}

func HandleGTC(c chan string, db *gorm.DB, s *clients.Session, args string) error {
	args, _, _ = strings.Cut(args, "\r\n")
	transactionID, args, err := parseTransactionID(args)
	if err != nil {
		return err
	}

	if !slices.Contains(gtcMode, args) {
		return errors.New("invalid mode")
	}

	if !s.Connected {
		SendError(c, transactionID, ERR_NOT_LOGGED_IN)
		return errors.New("not logged in")
	}

	var user database.User
	query := db.First(&user, "email = ?", s.Email)
	if errors.Is(query.Error, gorm.ErrRecordNotFound) {
		return errors.New("user not found")
	} else if query.Error != nil {
		return query.Error
	}

	if user.Gtc == args {
		SendError(c, transactionID, ERR_ALREADY_IN_THE_MODE)
		return errors.New("user already in requested mode")
	}

	user.Gtc = args
	user.DataVersion++
	query = db.Save(&user)
	if query.Error != nil {
		return query.Error
	}

	HandleSendGTC(c, transactionID, user.DataVersion, user.Gtc)
	return nil
}

func HandleSendGTC(c chan string, tid string, version uint32, gtc string) {
	res := fmt.Sprintf("GTC %s %d %s\r\n", tid, version, gtc)
	c <- res
}
