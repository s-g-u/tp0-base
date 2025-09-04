package common

import (
    "net"
)
const BUFFER_SIZE = 1024

// send ensures the entire message is written to the connection,
// handling short writes explicitly, and appending a null terminator.
func send(connection net.Conn, messageToSend string) error {
	data := []byte(messageToSend + "\000")

	totalWritten := 0
	for totalWritten < len(data) {
		n, err := connection.Write(data[totalWritten:])
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrUnexpectedEOF
		}
		totalWritten += n
	}

	return nil
}

// readUpToDelimiter reads from connection until the delimiter is found
func readUpToDelimiter(conn net.Conn, delim string) (string, error) {
    buf := make([]byte, 0, BUFFER_SIZE)
    tmp := make([]byte, 1)
    for {
        _, err := conn.Read(tmp)
        if err != nil {
            return "", err
        }
        if string(tmp) == delim {
            break
        }
        buf = append(buf, tmp[0])
    }
    return string(buf), nil
}
