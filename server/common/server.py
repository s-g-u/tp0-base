import socket
import logging
import signal

from common.utils import Bet, store_bets
from common.connection import send, read_up_to_delimiter

NUM_BET_FIELDS = 6
class Server:
    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)

        self._is_running = True
        self._last_client_socket = None

        # graceful shutdown on SIGTERM
        signal.signal(signal.SIGTERM, self.__shutdown_server)

    def run(self):
        """
        Main server loop: keeps accepting clients and processing bets
        """
        try:
            while self._is_running:
                client_socket = self.__accept_new_connection()
                if client_socket:
                    self._last_client_socket = client_socket
                    self.__handle_client_connection()
        except Exception as e:
            logging.error(f"action: server_run | result: fail | error: {e}")
        finally:
            self.__shutdown_server(None, None)

    def __handle_client_connection(self):
        """
        Handle bets from one client, send response, and close socket
        """
        try:
            bets, invalid_count = self.__read_bets()
            valid_count = len(bets)

            # Store valid bets
            if valid_count:
                store_bets(bets)

            # Log and respond
            if invalid_count > 0:
                logging.error(
                    f"action: apuesta_recibida | result: partial_fail | valid: {valid_count} | invalid: {invalid_count}"
                )
                send(self._last_client_socket, f"ERR;{invalid_count}")
            else:
                logging.info(
                    f"action: apuesta_recibida | result: success | cantidad: {valid_count}"
                )
                send(self._last_client_socket, "ACK")

        except (ConnectionResetError, OSError) as e:
            logging.error(f"action: client_connection | result: fail | error: {e}")
            try:
                send(self._last_client_socket, "ERR")
            except Exception:
                pass
        finally:
            self._last_client_socket.close()
            self._last_client_socket = None

    def __read_bets(self):
        """
        Reads a batch of bets until null delimiter.
        Returns a tuple: (list_of_bets, errors_count)
        """
        data = read_up_to_delimiter(self._last_client_socket, "\0")
        bets = []
        errors = 0

        for line in filter(None, data.split("\n")):
            bet = self.__parse_bet_line(line)
            if bet:
                bets.append(bet)
            else:
                errors += 1

        return bets, errors

    def __parse_bet_line(self, line):
        """
        Parse a single line and return a Bet object.
        Returns None if parsing fails.
        """
        fields = line.strip().split(";")
        if len(fields) != NUM_BET_FIELDS:
            logging.warning(f"action: parse_bet | result: fail | line: {line}")
            return None

        agency, name, surname, dni, birthdate, number = fields
        if not agency.isdigit() or not number.isdigit():
            logging.warning(f"action: parse_bet | result: fail | invalid agency/number | line: {line}")
            return None

        return Bet(agency, name, surname, dni, birthdate, number)

    def __accept_new_connection(self):
        """
        Wait for a new client connection
        """
        logging.info("action: accept_connections | result: in_progress")
        try:
            client, addr = self._server_socket.accept()
            logging.info(f"action: accept_connections | result: success | ip: {addr[0]}")
            return client
        except OSError:
            logging.info("action: accept_connections | result: server_closed")
            return None

    def __shutdown_server(self, signum, frame):
        """
        Gracefully shutdown the server and client socket
        """
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
