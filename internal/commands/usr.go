package commands

import (
	"errors"
	"net"
	"strings"
)

type authParams struct {
	authMethod string
	authState  string
	username   string
}

func HandleReceiveUSR(conn net.Conn, arguments string) (string, *authParams, error) {
	arguments, _, _ = strings.Cut(arguments, "\r\n")
	transactionID, arguments, err := parseTransactionID(arguments)
	if err != nil {
		return "", nil, err
	}

	splitArguments := strings.Split(arguments, " ")
	if len(splitArguments) != 3 {
		err := errors.New("Invalid transaction")
		return "", nil, err
	}

	authParams := authParams{authMethod: splitArguments[0], authState: splitArguments[1], username: splitArguments[2]}

	return transactionID, &authParams, nil
}
