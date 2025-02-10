package commands

import (
	"context"
	"errors"
	"fmt"
	"msnserver/config"
	"msnserver/pkg/clients"
	"msnserver/pkg/database"
	"msnserver/pkg/utils"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	SB_SECURITY_PACKAGE string        = "CKI"
	CKI_TIMEOUT         time.Duration = 2 * time.Minute
)

func HandleXFRDispatch(cf config.DispatchServer, c *clients.Client, transactionID string) {
	res := fmt.Sprintf("XFR %s NS %s:%s\r\n", transactionID, cf.NotificationServerAddr, cf.NotificationServerPort)
	c.SendChan <- res
}

func HandleXFR(cf config.NotificationServer, db *gorm.DB, rdb *redis.Client, c *clients.Client, arguments string) error {
	arguments, _, _ = strings.Cut(arguments, "\r\n")
	tid, arguments, err := parseTransactionID(arguments)
	if err != nil {
		return err
	}

	if !c.Session.Authenticated {
		SendError(c.SendChan, tid, ERR_NOT_LOGGED_IN)
		return errors.New("not logged in")
	}

	if arguments != "SB" {
		SendError(c.SendChan, tid, ERR_INVALID_PARAMETER)
		return errors.New("invalid parameter")
	}

	var user database.User
	query := db.First(&user, "email = ?", c.Session.Email)
	if query.Error != nil {
		return query.Error
	}

	if user.Status == "HDN" {
		SendError(c.SendChan, tid, ERR_NOT_ALLOWED_WHEN_OFFLINE)
		return nil
	}

	cki := utils.GenerateRandomString(25)
	if err := rdb.Set(context.TODO(), c.Session.Email, cki, CKI_TIMEOUT).Err(); err != nil {
		return err
	}

	res := fmt.Sprintf("XFR %s SB %s:%s %s %s\r\n", tid, cf.SwitchboardServerAddr, cf.SwitchboardServerPort, SB_SECURITY_PACKAGE, cki)
	c.Send(res)

	return nil
}
