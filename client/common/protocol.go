package common
import (
	"fmt"
	"strings"
)

const ACK_MESSAGE = "ACK"
const NOWINNER_MESSAGE = "NOWINNER"
const ERR_MESSAGE = "ERR"
const WINNERS_MESSAGE = "WINNERS"
const WRONG_MESSAGE = "WRONG"


// handleServerResponse processes raw server response without using structs
func (client *Client) handleServerResponse(reply string) {
	switch {
	case strings.HasPrefix(reply, ACK_MESSAGE):
		log.Infof("action: batch_send | result: success")

	case strings.HasPrefix(reply, NOWINNER_MESSAGE):
		log.Infof("action: sleeping | result: success")

	case strings.HasPrefix(reply, ERR_MESSAGE):
		parts := strings.Split(reply, ";")
		if len(parts) == 2 {
			log.Errorf("action: receive_message | result: fail | number_of_errors: %s", parts[1])
		} else {
			log.Errorf("action: receive_message | result: fail | message: malformed_error")
		}

	case strings.HasPrefix(reply, WINNERS_MESSAGE):
		winners := strings.Split(reply, ";")
		log.Infof("action: consulta_ganadores | result: success | cant_ganadores: %d", len(winners)-1)

	default:
		log.Errorf("action: receive_message | result: fail | message: wrong_message_received")
	}
}


// serializeBatch converts bets into a string payload
func serializeBatch(bets []Bet) string {
	records := make([]string, len(bets))
	for i, b := range bets {
		records[i] = fmt.Sprintf("%s;%s;%s;%s;%s;%s",
			b.Agency, b.Name, b.Surname, b.DNI, b.Birthdate, b.Number)
	}
	return strings.Join(records, "\n")
}

// size returns the size in bytes of the serialized bet
func (b *Bet) size() int {
	return len(fmt.Sprintf("%s;%s;%s;%s;%s;%s",
		b.Agency, b.Name, b.Surname, b.DNI, b.Birthdate, b.Number))
}