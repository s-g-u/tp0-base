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

def generar_compose(clients):
    if clients < MIN_CLIENTS:
        raise ValueError("La cantidad de clientes debe ser mayor o igual a 1")

    lines = [
        f"name: {NAME}",
        "services:",
        "  server:",
        "    container_name: server",
        f"    image: {SERVER_IMAGE}",
        f"    entrypoint: {SERVER_ENTRYPOINT}",
        "    environment:",
        f"      - PYTHONUNBUFFERED={PYTHON_UNBUFFERED}",
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
            "    networks:",
            f"      - {NETWORK_NAME}",
            "    depends_on:",
            "      - server",
            "    volumes:",
            f"      - ./client/config.yaml:/config.yaml"
        ]
        lines.extend(client_block)

    network_block = [
        "networks:",
        f"  {NETWORK_NAME}:",
        "    ipam:",
        f"     driver: {DRIVER}",
        "     config:",
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
        print("La forma correcta de ejecutar el programa es: python3 mi-generador.py <archivo_salida> <cantidad_clientes>")
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