package common

import (
    "net"
)
const BUFFER_SIZE = 1024

// Send writes the entire message to the TCP connection,
// appending a null terminator, handling short writes explicitly.
func send(conn net.Conn, message string) error {
	message += "\000"
	data := []byte(message)

	totalWritten := 0
	for totalWritten < len(data) {
		n, err := conn.Write(data[totalWritten:])
		if err != nil {
			return err
		}
		if n == 0 {
			return &net.OpError{Op: "write", Net: "tcp", Err: err}
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
