
package common
import (
	"fmt"
	"strings"
)

const ACK = "ACK"
const MAX_BATCH_MEMORY = 8192

// Bet represents a bet message
type Bet struct {
	Agency    string
	Name      string
	Surname   string
	DNI       string
	Birthdate string
	Number    string
}

// serializeBatch converts bets into a string payload
func serializeBatch(bets []Bet) string {
	records := make([]string, len(bets))
	for i, b := range bets {
		records[i] = fmt.Sprintf("%s;%s;%s;%s;%s;%s", b.Agency, b.Name, b.Surname, b.DNI, b.Birthdate, b.Number)
	}
	return strings.Join(records, "\n")
}

// size returns the size in bytes of the serialized bet
func (b *Bet) size() int {
	return len(fmt.Sprintf("%s;%s;%s;%s;%s;%s", b.Agency, b.Name, b.Surname, b.DNI, b.Birthdate, b.Number))
}
