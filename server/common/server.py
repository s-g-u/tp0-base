import socket
import logging
import signal
from common.connection import send, read_up_to_delimiter
NUM_BET_FIELDS = 6

class Bet:
    """Internal class to represent a bet"""
    def __init__(self, agency, name, surname, dni, birthdate, number):
        self.agency = agency
        self.name = name
        self.surname = surname
        self.dni = dni
        self.birthdate = birthdate
        self.number = number

    @staticmethod
    def from_message(msg: str):
        """Creates a Bet from a semicolon-separated string"""
        parts = msg.split(";")
        if len(parts) != NUM_BET_FIELDS:
            raise ValueError("Bad message, 6 fields were expected")
        agency, name, surname, dni, birthdate, number = parts
        if not agency.isdigit() or not number.isdigit():
            raise ValueError("Fields agency y number should be numbers")
        return Bet(agency, name, surname, dni, birthdate, number)

#dummy 
def store_bets(bets):
    for bet in bets:
        logging.info(f"Stored bet: dni={bet.dni}, number={bet.number}")

class Server:
    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self._is_running = True          
        self._last_client_socket = None   
        signal.signal(signal.SIGTERM, self.__shutdown_server)

    def run(self):
        """
        Dummy Server loop

        Server that accept a new connections and establishes a
        communication with a client. After client with communucation
        finishes, servers starts to accept new connections again
        """
        try:
            while self._is_running:
                self._last_client_socket = self.__accept_new_connection()
                if self._last_client_socket:
                    self.__handle_client_connection()
        except Exception as e:
            logging.error(f"action: server_run | result: fail | error: {e}")
        finally:
            self.__shutdown_server(None, None)

    def __handle_client_connection(self):
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
        try:
            raw_message = read_up_to_delimiter(self._last_client_socket, "\0")
            logging.info(f"action: reading_bet | result: success | message: {raw_message}")
            bet = Bet.from_message(raw_message)
            store_bets([bet])
            logging.info(f"action: apuesta_almacenada | result: success | dni: {bet.dni} | numero: {bet.number}")
            send(self._last_client_socket, "ACK")
        except OSError as e:
            logging.error(f"action: receive_message | result: fail | error: {e}")
            try:
                send(self._last_client_socket, "ERR")
            except Exception:
                pass
        finally:
            self._last_client_socket.close()
            self._last_client_socket = None

    def __accept_new_connection(self):
        """
        Accept new connections

        Function blocks until a connection to a client is made.
        Then connection created is printed and returned
        """
        # Connection arrived
        logging.info('action: accept_connections | result: in_progress')
        c, addr = self._server_socket.accept()
        logging.info(f'action: accept_connections | result: success | ip: {addr[0]}')
        return c

    def __shutdown_server(self, signum, frame):
        """Gracefully shutdown the server and the client connections"""
        self._is_running = False

        if self._server_socket:
            self._server_socket.close()
            self._server_socket = None
            logging.info("action: shutdown_server | result: success")  

        if self._last_client_socket:
            self._last_client_socket.close()
            self._last_client_socket = None
            logging.info("action: shutdown_client | result: success") 

        logging.info("action: shutdown | result: success")
