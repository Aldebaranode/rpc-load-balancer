# RPC Load Balancer

A high-performance, intelligent load balancer for Ethereum-compatible RPC nodes, written in Go. This gateway forwards your JSON-RPC requests to the healthiest and most performant node available, improving the reliability and speed of your dApp or service interactions.

## Features

* **Multiple Upstream Support:** Configure and manage a pool of several RPC endpoints.
* **Intelligent Health Checking:** Periodically checks each node for:
    * **Latency:** Measures the response time.
    * **Block Height:** Determines how up-to-date the node is.
* **Best Node Selection:** Routes traffic based on:
    * The lowest latency.
    * Nodes within an acceptable block height tolerance (`BlockTolerance`) compared to the highest available block.
* **Rate Limit Handling:** Detects `HTTP 429 (Too Many Requests)` responses and temporarily backs off from the affected node to avoid hammering it.
* **YAML Configuration:** Easily configure endpoints and parameters through a simple `config.yaml` file.
* **Efficient Reverse Proxy:** Uses Go's standard library for efficient request forwarding.
* **Graceful Shutdown:** Handles `SIGINT` and `SIGTERM` signals to shut down cleanly, allowing in-flight requests to complete.

## How it Works

1.  **Startup:** The gateway reads its configuration from `config.yaml`, initializes a pool of RPC endpoints, and starts an HTTP server.
2.  **Periodic Checks:** A background goroutine runs at a configured interval (`checkInterval`). It concurrently sends an `eth_blockNumber` request to every non-rate-limited node.
3.  **Selection:** It gathers latency and block number data from the checks. It filters out nodes that are too far behind the highest block (`blockTolerance`) and then selects the node with the lowest latency from the remaining candidates. This node becomes the `CurrentBest`.
4.  **Request Handling:** When the gateway receives an incoming JSON-RPC request:
    * It consults the `CurrentBest` node.
    * It forwards the request to that node using a reverse proxy.
    * It monitors the response. If a `429` is detected, it marks that node as rate-limited for a configurable period (`rateLimitBackoff`).
    * It returns the response to the client.

## Prerequisites

* **Go:** Version 1.21 or higher is recommended. You can download it from [golang.org](https://golang.org/).

## Configuration

Before running, create a `config.yaml` file in the same directory as the executable. You can start with `config-example.yaml` and modify it:

```yaml
# config.yaml - Configuration for the Go Ethereum RPC Gateway

# Port for the gateway to listen on (e.g., ":8545")
gatewayPort: ":8545"

# How often to check node status (e.g., "30s", "1m", "500ms")
checkInterval: "30s"

# Max time to wait for an RPC node response during checks (e.g., "5s")
requestTimeout: "5s"

# How many blocks behind the highest an endpoint can be
blockTolerance: 5

# How long to wait before retrying a rate-limited node (e.g., "1m", "90s")
rateLimitBackoff: "1m"

# List of upstream Ethereum RPC nodes.
# IMPORTANT: Replace placeholders with your actual keys/IDs.
rpcEndpoints:
  - "https://mainnet.infura.io/v3/YOUR_INFURA_PROJECT_ID"
  - "https://rpc.ankr.com/eth"
  - "https://eth-mainnet.alchemyapi.io/v2/YOUR_ALCHEMY_API_KEY"
  - "https://cloudflare-eth.com"
  # Add more endpoints here
```

**Make sure to replace the placeholder API keys/Project IDs with your actual ones.**

## Installation & Setup

1.  **Clone the Repository (or setup files):**
    ```bash
    git clone <repository_url> # Or just place the .go files in a directory
    cd rpc-load-balancer
    ```
2.  **Initialize Go Module:**
    If you cloned, it might already exist. If not, or if you created files manually:
    ```bash
    go mod init rpc-load-balancer # Or your preferred module path
    ```
3.  **Get Dependencies:**
    This command will download the `yaml.v3` library and update your `go.mod` and `go.sum` files.
    ```bash
    go mod tidy
    ```
4.  **Configure:**
    Create or edit your `config.yaml` as described above.

## Usage

### Running Directly

You can run the gateway directly from the source code:

```bash
go run .
```

### Building an Executable

For production use or easier distribution, build a binary:

```bash
go build -o rpc-gateway .
```

Then run the executable:

```bash
./rpc-gateway
```

### Using the Gateway

Once running, the gateway will listen on the port specified in `config.yaml` (default: `:8545`). Point your dApp, wallet, or script's RPC endpoint to `http://localhost:8545` (or `http://<your_server_ip>:8545`). The gateway will handle routing your requests to the best upstream node.

## Contributing

Contributions are welcome! Please feel free to open an issue or submit a pull request.

## License

This project is licensed under the MIT License - see the LICENSE file (if available) for details.