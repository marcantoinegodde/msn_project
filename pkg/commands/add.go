package commands

import (
	"errors"
	"fmt"
	"log"
	"msnserver/pkg/clients"
	"msnserver/pkg/database"
	"regexp"
	"strings"

	"gorm.io/gorm"
)

func HandleADD(db *gorm.DB, clients map[string]*clients.Client, c *clients.Client, args string) error {
	args, _, _ = strings.Cut(args, "\r\n")
	transactionID, args, err := parseTransactionID(args)
	if err != nil {
		return err
	}

	if !c.Session.Authenticated {
		SendError(c.SendChan, transactionID, ERR_NOT_LOGGED_IN)
		return errors.New("not logged in")
	}

	splitArguments := strings.Fields(args)
	if len(splitArguments) != 3 {
		return errors.New("invalid transaction")
	}

	listName := splitArguments[0]
	email := splitArguments[1]
	displayName := splitArguments[2]

	if !isValidEmail(email) {
		SendError(c.SendChan, transactionID, ERR_INVALID_PARAMETER)
		log.Printf("Error: invalid email: %s\n", email)
		return nil
	}

	if c.Session.Email == email {
		SendError(c.SendChan, transactionID, ERR_INVALID_USER)
		log.Println("Error: tried to add self to list")
		return nil
	}

	var user database.User
	query := db.Preload("ForwardList").Preload("AllowList").Preload("BlockList").First(&user, "email = ?", c.Session.Email)
	if errors.Is(query.Error, gorm.ErrRecordNotFound) {
		return errors.New("user not found")
	} else if query.Error != nil {
		return query.Error
	}

	var principal database.User
	query = db.Preload("ForwardList").Preload("AllowList").Preload("BlockList").First(&principal, "email = ?", email)
	if errors.Is(query.Error, gorm.ErrRecordNotFound) {
		SendError(c.SendChan, transactionID, ERR_INVALID_USER)
		log.Printf("Error: user not found: %s\n", email)
		return nil
	} else if query.Error != nil {
		return query.Error
	}

	switch listName {
	case "FL":
		if len(user.ForwardList) >= 150 {
			SendError(c.SendChan, transactionID, ERR_LIST_FULL)
			log.Println("Error: forward list full")
			return nil
		}

		if isMember(user.ForwardList, &principal) {
			SendError(c.SendChan, transactionID, ERR_ALREADY_THERE)
			log.Println("Error: user already in forward list")
			return nil
		}

		// Add principal to user's forward list
		user.ForwardList = append(user.ForwardList, &principal)
		user.DataVersion++
		if err := db.Save(&user).Error; err != nil {
			return err
		}

		// Add user to principal's reverse list
		principal.ReverseList = append(principal.ReverseList, &user)
		principal.DataVersion++
		if err := db.Save(&principal).Error; err != nil {
			return err
		}

		// Notify principal if online
		if clients[principal.Email] != nil {
			res := fmt.Sprintf("ADD %s %s %d %s %s\r\n", "0", "RL", principal.DataVersion, user.Email, user.DisplayName)
			clients[principal.Email].SendChan <- res
		}

		// Notify user if online, not blocked and explicitely allowed if BLP is BL
		if !(principal.Status == "FLN" || principal.Status == "HDN") &&
			!isMember(principal.BlockList, &user) &&
			!(principal.Blp == "BL" && !isMember(principal.AllowList, &user)) {
			HandleSendILN(c.SendChan, transactionID, principal.Status, principal.Email, principal.DisplayName)
		}

	case "AL":
		if len(user.AllowList) >= 150 {
			SendError(c.SendChan, transactionID, ERR_LIST_FULL)
			log.Println("Error: allow list full")
			return nil
		}

		if isMember(user.BlockList, &principal) {
			SendError(c.SendChan, transactionID, ERR_ALREADY_IN_OPPOSITE_LIST)
			log.Println("Error: trying to add in AL and BL")
			return nil
		}

		// Add principal to user's allow list
		user.AllowList = append(user.AllowList, &principal)
		user.DataVersion++
		if err := db.Save(&user).Error; err != nil {
			return err
		}

		HandleSendNLN(clients[principal.Email].SendChan, user.Status, user.Email, user.DisplayName)

	case "BL":
		if len(user.BlockList) >= 150 {
			SendError(c.SendChan, transactionID, ERR_LIST_FULL)
			log.Println("Error: block list full")
			return nil
		}

		if isMember(user.AllowList, &principal) {
			SendError(c.SendChan, transactionID, ERR_ALREADY_IN_OPPOSITE_LIST)
			log.Println("Error: trying to add in AL and BL")
			return nil
		}

		// Add principal to user's block list
		user.BlockList = append(user.BlockList, &principal)
		user.DataVersion++
		if err := db.Save(&user).Error; err != nil {
			return err
		}

		HandleSendFLN(clients[principal.Email].SendChan, user.Email)

	case "RL":
		// User cannot modify reverse list
		err := errors.New("tried to add to reverse list")
		return err

	default:
		err := errors.New("unsupported list")
		return err
	}

	res := fmt.Sprintf("ADD %s %s %d %s %s\r\n", transactionID, listName, user.DataVersion, email, displayName)
	c.SendChan <- res

	return nil
}

func isValidEmail(email string) bool {
	const emailRegex = `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	re := regexp.MustCompile(emailRegex)
	return re.MatchString(email)
}
