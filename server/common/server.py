import socket
import logging
import signal
from common.utils import Bet,store_bets, load_bets, has_won
from common.connection import send, read_up_to_delimiter

class Server:
    def __init__(self, port, listen_backlog, clients):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self._is_running = True
        self._last_client_socket = None
        self._completed_agencies = set()
        self._clients = clients

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
        Handles communication with a single client: receives data, processes it, and sends a response.
        The socket is always closed at the end, regardless of the outcome.
        """
        sock = self._last_client_socket
        reply = None
        try:
            incoming = read_up_to_delimiter(sock, "\0")

            if incoming.startswith("GETWINNERS"):
                reply = self.__get_winners(incoming)
            else:
                apuestas, errores = self.__get_bets(incoming)
                store_bets(apuestas)

                if errores:
                    logging.error(
                        f"action: apuesta_recibida | result: fail  | cantidad: {errores}"
                    )
                    reply = f"ERR;{errores}"
                else:
                    logging.info(
                        f"action: apuesta_recibida | result: success | cantidad: {len(apuestas)}"
                    )
                    reply = "ACK"

            if reply is not None:
                send(sock, reply)

        except ConnectionResetError:
            logging.info(
                "action: server_run | result: success | message: the socket has closed"
            )
        except OSError as err:
            logging.error(f"action: receive_message | result: fail | error: {err}")
        finally:
            try:
                sock.close()
            finally:
                self._last_client_socket = None

    
    def __get_winners(self, message): 
            """
            Get the Winners bets from a Message in the format WINNERS;AGENCY_NUMBER
            from the bets stored
            """
            values = message.split(";")
            
            self._completed_agencies.add(values[1])
            if len(self._completed_agencies) != self._clients:
                logging.info("action: no_winner | result: success")
                return "NOWINNER"
            else:
                message = "WINNERS;"
                for bet in load_bets():
                    if has_won(bet):
                        if bet.agency == int(values[1]):
                            message = message + bet.document + ";"
                logging.info(f"action: lottery | result: success")
                return message[:-1]
        
    def __get_bets(self, raw_message):
        """
        Converts a raw bets message into a list of Bet objects,
        while counting any lines that failed validation.
        """
        valid_bets = []
        errors = 0

        lines = raw_message.strip().splitlines()
        for line in lines:
            parts = line.strip().split(";")
            
            if len(parts) != 6:
                errors += 1
                continue

            agency, name, surname, dni, birthdate, number = parts

            if not (agency.isdigit() and number.isdigit()):
                errors += 1
                continue

            valid_bets.append(Bet(agency, name, surname, dni, birthdate, number))

        return valid_bets, errors

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