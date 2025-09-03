import sys

NAME = "tp0"
SERVER_IMAGE = "server:latest"
CLIENT_IMAGE = "client:latest"
NETWORK_NAME = "testing_net"
SUBNET = "172.25.125.0/24"
SERVER_ENTRYPOINT = "python3 /main.py"
CLIENT_ENTRYPOINT = "/client"
PYTHON_UNBUFFERED = "1"
DRIVER = "default"

FIRST_CLIENT_ID = 1
MIN_CLIENTS = 0
EXPECTED_ARGS = 3   

NAMES = ["MARTA", "JULIAN", "SOFIA", "CARLOS", "LAURA"]
SURNAMES = ["PEREZ", "GOMEZ", "RODRIGUEZ", "FERNANDEZ", "MARTINEZ"]
DNIS = ["10234567", "20345678", "30456789", "40567890", "50678901"]
BIRTHDATES = ["1985-03-21", "1990-07-15", "1995-11-30", "1980-06-05", "1992-12-12"]
NUMBERS = ["9876", "1234", "5678", "4321", "8765"]


def generar_compose(clients):
    if clients < MIN_CLIENTS:
        raise ValueError("La cantidad de clientes debe ser mayor o igual a 0")
    if clients > len(NAMES):
        raise ValueError("No hay suficientes datos de clientes para generar ese número")

    lines = [
        f"name: {NAME}",
        "services:",
        "  server:",
        f"    container_name: server",
        f"    image: {SERVER_IMAGE}",
        f"    entrypoint: {SERVER_ENTRYPOINT}",
        "    environment:",
        f"      - PYTHONUNBUFFERED={PYTHON_UNBUFFERED}",
        f"      - TOTAL_AGENCIES={clients}",
        "    networks:",
        f"      - {NETWORK_NAME}",
        "    volumes:",
        "      - ./server/config.ini:/config.ini"
    ]

    for i in range(FIRST_CLIENT_ID, clients + 1):
        client_block = [
            f"  client{i}:",
            f"    container_name: client{i}",
            f"    image: {CLIENT_IMAGE}",
            f"    entrypoint: {CLIENT_ENTRYPOINT}",
            "    environment:",
            f"      - CLI_ID={i}",
            f"      - CLI_NOMBRE={NAMES[i-1]}",
            f"      - CLI_APELLIDO={SURNAMES[i-1]}",
            f"      - CLI_DOCUMENTO={DNIS[i-1]}",
            f"      - CLI_NACIMIENTO={BIRTHDATES[i-1]}",
            f"      - CLI_NUMERO={NUMBERS[i-1]}",
            "    networks:",
            f"      - {NETWORK_NAME}",
            "    depends_on:",
            "      - server",
            "    volumes:",
            f"      - ./client/config.yaml:/config.yaml",
            f"      - ./.data:/.data",
        ]
        lines.extend(client_block)

    network_block = [
        "networks:",
        f"  {NETWORK_NAME}:",
        "    ipam:",
        f"      driver: {DRIVER}",
        "      config:",
        f"        - subnet: {SUBNET}"
    ]
    lines.extend(network_block)

    return "\n".join(lines)

def guardar_compose(path, clients):
    content = generar_compose(clients)
    try:
        with open(path, "w", encoding="utf-8") as f:
            f.write(content)
    except Exception as e:
        print(f"Error al guardar el archivo '{path}': {e}")
        sys.exit(1)

def main():
    if len(sys.argv) != EXPECTED_ARGS:
        print(f"La forma correcta de ejecutar el programa es: python3 {sys.argv[0]} <archivo_salida> <cantidad_clientes>")
        sys.exit(1)

    output_file = sys.argv[1]

    try:
        clients_count = int(sys.argv[2])
        if clients_count < MIN_CLIENTS:
            raise ValueError
    except ValueError:
        print("Error: la cantidad de clientes debe ser un entero mayor o igual a 1")
        sys.exit(1)

    guardar_compose(output_file, clients_count)
    print(f"Docker Compose generado correctamente en '{output_file}' con {clients_count} clientes.")

if __name__ == "__main__":
    main()
