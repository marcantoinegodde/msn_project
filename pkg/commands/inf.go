package commands

import (
	"fmt"
	"strings"
)

var supportedAuthMethods = []string{"MD5"}

func HandleINF(c chan string, arguments string) error {
	arguments, _, _ = strings.Cut(arguments, "\r\n")
	transactionID, _, err := parseTransactionID(arguments)
	if err != nil {
		return err
	}

	res := fmt.Sprintf("INF %s %s\r\n", transactionID, strings.Join(supportedAuthMethods, " "))
	c <- res
	return nil
}
