# Free Clash Sub

Free Clash Sub is a Go-based application that generates and serves Clash configuration files. It fetches proxy data from specified URLs, filters the proxies based on various criteria, and generates a Clash configuration file using a template.

## Features

- Fetches proxy data from multiple URLs
- Filters proxies based on type, name, and other criteria
- Generates Clash configuration files using a template
- Serves the generated configuration file via HTTP
- Periodically updates the configuration file

## Prerequisites

- Go 1.20 or higher
- Docker (optional, for containerized deployment)

## Installation

1. Clone the repository:
   ```
   git clone https://github.com/yourusername/free-clash-sub.git
   cd free-clash-sub
   ```

2. Install dependencies:
   ```
   go mod download
   ```

## Configuration

The application is configured using environment variables:

- `PROXY_SOURCE_URLS`: Comma-separated list of URLs to fetch proxy data from
- `PROXY_SOURCE_NUM`: Maximum number of proxies to include in the configuration
- `PROXY_APPLY_GROUPS`: Comma-separated list of proxy groups to apply the fetched proxies to
- `PROXY_TYPE_FILTER`: Comma-separated list of proxy types to include
- `PROXY_NAME_EXCLUDE_FILTER`: Regex pattern for proxy names to exclude
- `PROXY_NAME_FILTER`: Regex pattern for proxy names to include

## Running the Application

### Running Locally

1. Set the environment variables (replace with your values):
   ```
   export PROXY_SOURCE_URLS="https://example.com/proxies1,https://example.com/proxies2"
   export PROXY_SOURCE_NUM=200
   export PROXY_APPLY_GROUPS="自动选择,节点选择"
   export PROXY_TYPE_FILTER="vmess,vless,trojan"
   export PROXY_NAME_EXCLUDE_FILTER=".*AD"
   export PROXY_NAME_FILTER=".*"
   ```

2. Run the application:
   ```
   go run main.go
   ```

The application will start and listen on port 8000.

### Running with Docker

1. Build the Docker image:
   ```
   docker build -t free-clash-sub .
   ```

2. Run the Docker container:
   ```
   docker run -d -p 8000:8000 \
     -e PROXY_SOURCE_URLS="https://github.com/aiboboxx/clashfree/raw/main/clash.yml" \
     -e PROXY_SOURCE_NUM=20 \
     -e PROXY_APPLY_GROUPS="自动选择,节点选择" \
     -e PROXY_TYPE_FILTER="vmess,vless,trojan" \
     -e PROXY_NAME_EXCLUDE_FILTER=".*AD" \
     -e PROXY_NAME_FILTER=".*" \
     --name free-clash-sub free-clash-sub
   ```

## Usage

Once the application is running, you can access the generated Clash configuration file at:

```
http://localhost:8000/config.yaml
```

The configuration file is automatically updated hourly.

## Free Clash Proxy
- https://github.com/OpenRunner/clash-freenode
- https://github.com/VPN-Subcription-Links/ClashX-V2Ray-TopFreeProxy
- https://github.com/aiboboxx/clashfree

## Development

To modify the template used for generating the Clash configuration, edit the `template.yaml` file in the project root.