# RPC Load Balancer

A simple yet effective load balancer for Ethereum-compatible RPC nodes, built with Go. It directs your requests to the best-performing node, making your DApp or service more reliable.

## Features

* **Multi-Node Support:** Use multiple RPC endpoints.
* **Health Checks:** Picks nodes with low latency and recent block numbers.
* **Rate Limit Aware:** Avoids nodes that are temporarily rate-limited (HTTP 429).
* **Configurable:** Uses a simple `config.yaml` file.
* **Metrics:** Provides Prometheus metrics for monitoring.
* **Docker Ready:** Easy to run with Docker and `docker-compose`.

## Quick Start (Docker Compose)

This is the recommended way to run the gateway.

**Prerequisites:**

* Docker & Docker Compose installed.

**Steps:**

1.  **Clone / Download:** Get the project files.
2.  **Create `config.yaml`:** Copy `config-example.yaml` to `config.yaml` and **edit it**:
    * **Add your RPC endpoints** with API keys/project IDs.
    * (Optional) Adjust ports and intervals.
    ```yaml
    # config.yaml
    gatewayPort: ":8545"
    metricsPort: ":9090"
    checkInterval: "30s"
    requestTimeout: "5s"
    blockTolerance: 5
    rateLimitBackoff: "1m"
    rpcEndpoints:
      - "https://YOUR_RPC_ENDPOINT_1"
      - "https://YOUR_RPC_ENDPOINT_2"
    ```
3.  **Create `docker-compose.yaml`:**
    ```yaml
    # docker-compose.yaml
    version: '3.8'
    services:
      rpc-gateway:
        build: . # Assumes Dockerfile exists in the current directory
        container_name: rpc-gateway
        restart: unless-stopped
        volumes:
          # Mount your local config.yaml into the container
          - ./config.yaml:/app/config.yaml
        ports:
          - "8545:8545" # Gateway port
          - "9191:9090" # Metrics port (mapped to avoid host conflict)
    ```
4.  **Run:**
    ```bash
    docker-compose up -d
    ```
5.  **Use:**
    * Send RPC requests to `http://localhost:8545`.
    * View metrics at `http://localhost:9191/metrics`.
6.  **Stop:**
    ```bash
    docker-compose down
    ```

## Local Usage (Without Docker)

**Prerequisites:**

* Go (1.21+) installed.

**Steps:**

1.  **Clone / Download:** Get the project files.
2.  **Navigate:** `cd rpc-load-balancer`
3.  **Setup Go Module:** `go mod init rpc-load-balancer && go mod tidy`
4.  **Configure:** Create and edit `config.yaml` as shown above.
5.  **Run:** `go run .`
6.  **Use:**
    * Gateway: `http://localhost:8545`
    * Metrics: `http://localhost:9090/metrics`