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

const ACK = "ACK"
const MAX_BATCH_MEMORY = 8192

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

	path := fmt.Sprintf("/.data/agency-%v.csv", client.config.ID)
	file, err := os.Open(path)
	if err != nil {
		log.Errorf("action: open_file | result: fail | client_id: %v | error: %v", client.config.ID, err)
		return
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	batchChan := make(chan []Bet)

	go client.batchProducer(reader, batchChan)

	for batch := range batchChan {
		if err := client.createClientSocket(); err != nil {
			return
		}

		payload := serializeBatch(batch)
		if err := send(client.conn, payload); err != nil {
			log.Errorf("action: send_batch | result: fail | client_id: %v | error: %v", client.config.ID, err)
			client.conn.Close()
			return
		}

		resp, err := readUpToDelimiter(client.conn, "\000")
		client.conn.Close()
		if err != nil {
			log.Errorf("action: receive_ack | result: fail | client_id: %v | error: %v", client.config.ID, err)
			return
		}

		if resp == ACK {
			log.Infof("action: batch_sent | result: success | client_id: %v | batch_size: %d", client.config.ID, len(batch))
		} else {
			log.Errorf("action: batch_sent | result: fail | client_id: %v | message: %v", client.config.ID, resp)
			return
		}

		time.Sleep(client.config.LoopPeriod)
	}
}

// batchProducer reads lines and groups them into batches
func (client *Client) batchProducer(reader *bufio.Reader, out chan<- []Bet) {
	defer close(out)

	for client.is_running {
		batch := make([]Bet, 0, client.config.Batchs)
		memUsed := 0

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
				if len(batch) > 0 {
					out <- batch
				}
				return
			}
			if err != nil {
				log.Errorf("action: read_line | result: fail | client_id: %v | error: %v", client.config.ID, err)
				return
			}

			parts := strings.Split(strings.TrimSpace(line), ",")
			if len(parts) != 5 {
				continue
			}

			bet := Bet{
				Agency:    client.config.ID,
				Name:      parts[0],
				Surname:   parts[1],
				DNI:       parts[2],
				Birthdate: parts[3],
				Number:    parts[4],
			}

			if memUsed+bet.size() > MAX_BATCH_MEMORY {
				client.lastLine = line
				break
			}

			batch = append(batch, bet)
			memUsed += bet.size()
		}

		if len(batch) > 0 {
			out <- batch
		}
	}
}
