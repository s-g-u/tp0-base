package common

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/op/go-logging"
)

const MAX_WAITS = 10
const BET_FIELDS = 5

var log = logging.MustGetLogger("log")

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID            string
	ServerAddress string
	LoopAmount    int
	LoopPeriod    time.Duration
	Batchs        int
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

// createBatch builds a batch from the reader
func (client *Client) createBatch(reader *bufio.Reader) ([]Bet, error) {
	batch := make([]Bet, 0, client.config.Batchs)
	usedMem := 0

	for len(batch) < client.config.Batchs {
		var line string
		var err error

		if client.lastLine != "" {
			line = client.lastLine
			client.lastLine = ""
		} else {
			line, err = reader.ReadString('\n')
		}

		if err == io.EOF {
			return batch, nil
		}
		if err != nil {
			return batch, err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) != BET_FIELDS {
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

// sendBatches reads batches from reader sequentially and sends them
func (client *Client) sendBatches(reader *bufio.Reader) error {
	for client.is_running {
		if err := client.createClientSocket(); err != nil {
			log.Criticalf("action: connect | result: fail | client_id: %v | error: %v",
				client.config.ID, err)
			return err
		}
		defer client.conn.Close()

		batch, err := client.createBatch(reader)
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
	file, err := os.Open(filepath)
	if err != nil {
		log.Errorf("action: open_file | result: fail | client_id: %v | error: %v",
			client.config.ID, err)
		return
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	if err := client.sendBatches(reader); err != nil {
		log.Errorf("action: sending_batches | result: fail | client_id: %v | error: %v",
			client.config.ID, err)
		return
	}

	if err := client.waitForWinners(); err != nil {
		log.Errorf("action: waiting_for_winners | result: fail | client_id: %v | error: %v",
			client.config.ID, err)
	}
}

// waitForWinners waits until WINNERS is received
func (client *Client) waitForWinners() error {
	resultCh := make(chan string, 1)
	errorCh := make(chan error, 1)

	go func() {
		for attempts := 1; attempts <= MAX_WAITS; attempts++ {
			if err := client.createClientSocket(); err != nil {
				errorCh <- err
				return
			}

			raw, err := client.getWinners()
			client.conn.Close()
			if err != nil {
				errorCh <- err
				return
			}

			if strings.HasPrefix(raw, WINNERS_MESSAGE) {
				resultCh <- raw
				return
			}

			time.Sleep(client.config.LoopPeriod * (1 << (attempts - 1)))
		}

		errorCh <- fmt.Errorf("max waits exceeded")
	}()

	select {
	case raw := <-resultCh:
		client.handleServerResponse(raw)
		return nil
	case err := <-errorCh:
		return err
	case <-client.signalChannel:
		client.is_running = false
		return fmt.Errorf("client shutdown")
	}
}

// getWinners sends the request to ask for the winners to the server
func (client *Client) getWinners() (string, error) {
	if err := send(client.conn, fmt.Sprintf("GETWINNERS;%v", client.config.ID)); err != nil {
		return "", err
	}

	return readUpToDelimiter(client.conn, "\000")
}
