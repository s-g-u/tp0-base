package common

import (
	"fmt"
)


const ACK = "ACK"

// Bet represents a bet message
type Bet struct {
	Agency    string
	Name      string
	Surname   string
	DNI       string
	Birthdate string
	Number    string
}

// serializeBet converts a Bet into a semicolon-separated string
func serializeBet(b Bet) string {
	return fmt.Sprintf("%s;%s;%s;%s;%s;%s", b.Agency, b.Name, b.Surname, b.DNI, b.Birthdate, b.Number)
}
