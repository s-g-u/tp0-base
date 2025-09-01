package common

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
 	"time"
	"github.com/op/go-logging"
)

const ACK = "ACK"

var log = logging.MustGetLogger("log")

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID            string
	ServerAddress string
    LoopAmount    int           
    LoopPeriod    time.Duration
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
	is_running    bool
}

// NewClient Initializes a new client receiving the configuration as a parameter
func NewClient(config ClientConfig) *Client {
	client := &Client{
		config:        config,
		signalChannel: make(chan os.Signal, 1),
		is_running:    true,
	}

	signal.Notify(client.signalChannel, syscall.SIGTERM)

	return client
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
		log.Criticalf(
			"action: connect | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
		return err
	}
	c.conn = conn
	return nil
}

// StartClientLoop handles connection, sends a single bet, and waits for ACK
func (c *Client) StartClientLoop() {
	go c.shutdownClientHandler()

	if err := c.createClientSocket(); err != nil {
		return
	}
	defer c.conn.Close()

	// Build bet from environment
	bet := c.buildBetFromEnv()

	// Send bet and wait for ACK
	if err := c.sendBetAndWait(bet); err != nil {
		log.Errorf("action: send_bet | result: fail | client_id: %v | error: %v", c.config.ID, err)
	} else {
		log.Infof("action: apuesta_enviada | result: success | dni: %s | numero: %s", bet.DNI, bet.Number)
	}
}

// buildBetFromEnv constructs a Bet from environment variables
func (c *Client) buildBetFromEnv() Bet {
	return Bet{
		Agency:    c.config.ID,
		Name:      os.Getenv("CLI_NOMBRE"),
		Surname:   os.Getenv("CLI_APELLIDO"),
		DNI:       os.Getenv("CLI_DOCUMENTO"),
		Birthdate: os.Getenv("CLI_NACIMIENTO"),
		Number:    os.Getenv("CLI_NUMERO"),
	}
}

// sendBetAndWait sends a single bet and waits for ACK
func (c *Client) sendBetAndWait(bet Bet) error {
	msg := serializeBet(bet)

	if err := send(c.conn, msg); err != nil {
		return fmt.Errorf("send error: %v", err)
	}

	resp, err := readUpToDelimiter(c.conn, "\000")
	if err != nil {
		return fmt.Errorf("receive error: %v", err)
	}

	if resp != ACK {
		return fmt.Errorf("unexpected response: %s", resp)
	}

	return nil
}

// serializeBet converts a Bet into a semicolon-separated string
func serializeBet(b Bet) string {
	return fmt.Sprintf("%s;%s;%s;%s;%s;%s", b.Agency, b.Name, b.Surname, b.DNI, b.Birthdate, b.Number)
}
