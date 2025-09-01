package common

import (
    "net"
)

// send writes a string message to the TCP connection
func send(conn net.Conn, message string) error {
    _, err := conn.Write([]byte(message + "\000"))
    return err
}

// readUpToDelimiter reads from connection until the delimiter is found
func readUpToDelimiter(conn net.Conn, delim string) (string, error) {
    buf := make([]byte, 0, 1024)
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
