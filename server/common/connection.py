BUFFER_SIZE = 1024

def send(connection, message):
    """
    Send a message ensuring full delivery, appending a null character at the end.
    """
    full_message = message + "\0"
    encoded = full_message.encode("utf-8")
    total_sent = 0

    while total_sent < len(encoded):
        sent = connection.send(encoded[total_sent:])
        if sent == 0:
            raise RuntimeError("Socket connection broken while sending")
        total_sent += sent


def read_up_to_delimiter(connection, delimiter):
    """
    Read from the connection until the delimiter is found, handling partial reads correctly.
    """
    buffer = bytearray()
    delimiter_bytes = delimiter.encode("utf-8")

    while True:
        data = connection.recv(BUFFER_SIZE)
        if not data:
            raise RuntimeError("Socket closed unexpectedly while reading")
        buffer.extend(data)

        pos = buffer.find(delimiter_bytes)
        if pos >= 0:
            return buffer[:pos].decode("utf-8")
