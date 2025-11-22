# Logstash SSH Client

A Go application built with [Fyne](https://fyne.io/) to connect to a remote Logstash server via SSH. It allows you to browse log files in a specified directory and view their contents, with support for transparent decompression of `.gz` files.

## Features

- **SSH Connection**: Connect using Host, Port, Username, Password, or Private Key.
- **File Browser**: Navigate through directories on the remote server.
- **Log Viewer**: View the content of `.log` and `.log.gz` files.
- **Gzip Support**: Automatically decompresses `.gz` files for viewing.

## Prerequisites

- Go 1.20 or later
- A remote server running Logstash (or any server with logs you want to view)
- SSH access to the server

## Installation

1.  Clone the repository:
    ```bash
    git clone https://github.com/n-cekic/logsearch.git
    cd logsearch
    ```

2.  Install dependencies:
    ```bash
    go mod tidy
    ```

## Usage

1.  Run the application:
    ```bash
    go run .
    ```

2.  **Login Screen**:
    - Enter the **Host** (e.g., `192.168.1.100`).
    - Enter the **Port** (default `22`).
    - Enter **Username** and **Password** OR **Key Path**.
    - Enter the **Remote Log Path** (default `/var/log/logstash`).
    - Click **Connect**.

3.  **Dashboard**:
    - The left panel shows the files and folders in the current directory.
    - Double-click a **Folder** to navigate into it.
    - Click the **Up** button to go to the parent directory.
    - Click a **File** to view its content in the right panel.

## License

[MIT](LICENSE)
