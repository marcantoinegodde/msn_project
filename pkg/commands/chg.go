package commands

import (
	"errors"
	"fmt"
	"msnserver/pkg/database"
	"slices"
	"strings"

	"gorm.io/gorm"
)

var statusCodes = []string{"NLN", "FLN", "HDN", "IDL", "AWY", "BSY", "BRB", "PHN", "LUN"}

func HandleCHG(c chan string, db *gorm.DB, s *Session, args string) error {
	args, _, _ = strings.Cut(args, "\r\n")
	transactionID, args, err := parseTransactionID(args)
	if err != nil {
		return err
	}
	args, _, _ = strings.Cut(args, " ") // Remove the trailing space sent for this command

	if !slices.Contains(statusCodes, args) {
		return fmt.Errorf("invalid status code: %s", args)
	}

	if !s.connected {
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

	user.Status = args
	query = db.Save(&user)
	if query.Error != nil {
		return query.Error
	}

	res := fmt.Sprintf("CHG %s %s\r\n", transactionID, user.Status)
	c <- res
	return nil
}
