package common

import (
    "net"
)
const BUFFER_SIZE = 1024
// send writes a string message to the TCP connection,
// ensuring that the full payload is delivered (handling short writes).
func send(conn net.Conn, message string) error {
    data := []byte(message + "\000")
    totalSent := 0

    for totalSent < len(data) {
        n, err := conn.Write(data[totalSent:])
        if err != nil {
            return err
        }
        if n == 0 {
            return io.ErrUnexpectedEOF
        }
        totalSent += n
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
