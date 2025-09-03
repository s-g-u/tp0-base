package common

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/op/go-logging"
)

const ACK_MESSAGE = "ACK"
const NOWINNER_MESSAGE = "NOWINNER"
const ERR_MESSAGE = "ERR"
const WINNERS_MESSAGE = "WINNERS"
const WRONG_MESSAGE = "WRONG"

const MAX_BATCH_MEMORY = 8192
const MAX_WAITS = 10

var log = logging.MustGetLogger("log")

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID            string
	ServerAddress string
	LoopAmount    int
	LoopPeriod    time.Duration
	Batchs        int
}

// Bet represents a bet message
type Bet struct {
	Agency    string
	Name      string
	Surname   string
	DNI       string
	Birthdate string
	Number    string
}

// Client Entity that encapsulates client behavior
type Client struct {
	config        ClientConfig
	conn          net.Conn
	signalChannel chan os.Signal
	lastLine      string
	is_running    bool
}

// NewClient Initializes a new client receiving the configuration as a parameter
func NewClient(config ClientConfig) *Client {
	c := &Client{
		config:        config,
		signalChannel: make(chan os.Signal, 1),
		lastLine:      "",
		is_running:    true,
	}
	signal.Notify(c.signalChannel, syscall.SIGTERM)
	return c
}

// createBatch reads lines from a slice of strings and returns a batch as []Bet
func (client *Client) createBatch(lines []string, startIndex *int) ([]Bet, error) {
	batch := make([]Bet, 0, client.config.Batchs)
	usedMem := 0

	for *startIndex < len(lines) && len(batch) < client.config.Batchs {
		line := strings.TrimSpace(lines[*startIndex])
		*startIndex++

		if line == "" {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) != 5 {
			continue
		}

		newBet := Bet{
			Agency:    client.config.ID,
			Name:      parts[0],
			Surname:   parts[1],
			DNI:       parts[2],
			Birthdate: parts[3],
			Number:    parts[4],
		}

		if usedMem+newBet.size() > MAX_BATCH_MEMORY {
			client.lastLine = line
			break
		}

		batch = append(batch, newBet)
		usedMem += newBet.size()
	}

	return batch, nil
}

// sendBatch sends a slice of bets to the server and waits for a response
func (client *Client) sendBatch(batch []Bet) error {
	payload := serializeBatch(batch)

	if err := send(client.conn, payload); err != nil {
		log.Errorf("action: send_message | result: fail | client_id: %v | error: %v",
			client.config.ID, err)
		return err
	}

	reply, err := readUpToDelimiter(client.conn, "\000")
	if err != nil {
		log.Errorf("action: receive_message | result: fail | client_id: %v | error: %v",
			client.config.ID, err)
		return err
	}

	client.handleServerResponse(reply)
	return nil
}

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

// sendBatches reads lines and sends all batches sequentially
func (client *Client) sendBatchesFromFile(lines []string) error {
	startIndex := 0
	for client.is_running {
		if err := client.createClientSocket(); err != nil {
			log.Criticalf("action: connect | result: fail | client_id: %v | error: %v",
				client.config.ID, err)
			return err
		}

		defer client.conn.Close()

		batch, err := client.createBatch(lines, &startIndex)
		if err != nil {
			log.Errorf("action: create_batch | result: fail | client_id: %v | error: %v",
				client.config.ID, err)
			return err
		}

		if len(batch) == 0 {
			break
		}

		if err := client.sendBatch(batch); err != nil {
			log.Errorf("action: send_batch | result: fail | client_id: %v | error: %v",
				client.config.ID, err)
			return err
		}
	}
	return nil
}

// shutdownClientHandler listens SIGTERM and closes gracefully
func (client *Client) shutdownClientHandler() {
	<-client.signalChannel
	if client.conn != nil {
		client.conn.Close()
	}
	client.is_running = false
	log.Infof("action: shutdown_client | result: success | client_id: %v", client.config.ID)
}

// createClientSocket Initializes client socket
func (c *Client) createClientSocket() error {
	conn, err := net.Dial("tcp", c.config.ServerAddress)
	if err != nil {
		log.Criticalf("action: connect | result: fail | client_id: %v | error: %v", c.config.ID, err)
		return err
	}
	c.conn = conn
	return nil
}

// StartClientLoop runs the main loop of the client
func (client *Client) StartClientLoop() {
	go client.shutdownClientHandler()

	filepath := fmt.Sprintf("/.data/agency-%v.csv", client.config.ID)
	content, err := os.ReadFile(filepath)
	if err != nil {
		log.Errorf("action: read_file | result: fail | client_id: %v | error: %v",
			client.config.ID, err)
		return
	}

	lines := strings.Split(string(content), "\n")

	if err := client.sendBatchesFromFile(lines); err != nil {
		log.Errorf("action: sending_batches | result: fail | client_id: %v | error: %v",
			client.config.ID, err)
		return
	}

	if err := client.waitForWinners(); err != nil {
		log.Errorf("action: waiting_for_winners | result: fail | client_id: %v | error: %v",
			client.config.ID, err)
	}
}

// waitForWinners waits until WINNERS is received using a ticker-based loop
func (client *Client) waitForWinners() error {
	interval := client.config.LoopPeriod
	for attempts := 1; attempts <= MAX_WAITS; attempts++ {
		ticker := time.NewTicker(interval)
		<-ticker.C
		ticker.Stop()

		if err := client.createClientSocket(); err != nil {
			return err
		}

		raw, err := client.getWinners()
		client.conn.Close()
		if err != nil {
			return err
		}

		client.handleServerResponse(raw)
		if strings.HasPrefix(raw, WINNERS_MESSAGE) {
			return nil
		}

		// increase wait time exponentially
		interval *= 2
	}

	log.Errorf("action: wait_for_winners | result: fail | client_id: %v", client.config.ID)
	return fmt.Errorf("max waits exceeded")
}


// getWinners sends the request to ask for the winners to the server
func (client *Client) getWinners() (string, error) {
	if err := send(client.conn, fmt.Sprintf("GETWINNERS;%v", client.config.ID)); err != nil {
		return "", err
	}

	return readUpToDelimiter(client.conn, "\000")
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
