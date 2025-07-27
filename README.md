# DevDocsMCP

[![Build Status](https://github.com/kelvinzer0/DevDocsMCP/actions/workflows/release.yml/badge.svg)](https://github.com/kelvinzer0/DevDocsMCP/actions/workflows/release.yml)
[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/kelvinzer0/DevDocsMCP?style=flat-square)](https://github.com/kelvinzer0/DevDocsMCP/releases/latest)
[![Go Version](https://img.shields.io/badge/Go-1.22-00ADD8?style=flat-square&logo=go)](https://golang.org)


DevDocsMCP is a command-line tool that allows you to search and read documentation directly from [DevDocs.io](https://devdocs.io/). It operates purely online, fetching data as needed.

## Setup

1.  **Go Installation:** Ensure you have Go installed on your system. You can download it from [golang.org](https://golang.org/dl/).

2.  **Clone the Repository:**
    ```bash
    git clone https://github.com/kelvinzer0/DevDocsMCP.git
    cd DevDocsMCP
    ```

3.  **Build the Application:**
    ```bash
    go build -o devdocsmcp cmd/devdocsmcp/main.go
    ```
    This will create an executable named `devdocsmcp` in the current directory.

## Usage

Navigate to the `DevDocsMCP` directory in your terminal.

### Search Documentation

To search for a term within a specific documentation set:

```bash
./devdocsmcp search -lang <language_slug> -query <search_query>
```

*   `<language_slug>`: The slug for the documentation (e.g., `html`, `css`, `angularjs~1.8`, `vite`, `tailwindcss`, `go`, `mysql`, `sqlite`). You can find a list of available documentation on [DevDocs.io](https://devdocs.io/).
*   `<search_query>`: The term you want to search for.

**Examples:**

*   Search for "display" in HTML documentation:
    ```bash
    ./devdocsmcp search -lang html -query display
    ```

*   Search for "foreach" in AngularJS 1.8 documentation:
    ```bash
    ./devdocsmcp search -lang angularjs~1.8 -query foreach
    ```

### Read Documentation Content

To read the content of a specific documentation entry:

```bash
./devdocsmcp read -lang <language_slug> -path <entry_path>
```

*   `<language_slug>`: The slug for the documentation (e.g., `html`, `css`, `angularjs~1.8`).
*   `<entry_path>`: The path to the specific documentation entry, as found in search results (e.g., `reference/elements/a`, `api/ng/function/angular.foreach`).

**Examples:**

*   Read the documentation for the HTML `<a>` element:
    ```bash
    ./devdocsmcp read -lang html -path reference/elements/a
    ```

*   Read the documentation for `angular.forEach`:
    ```bash
    ./devdocsmcp read -lang angularjs~1.8 -path api/ng/function/angular.foreach
    ```

### Run as an MCP Server

`DevDocsMCP` can also run as an HTTP server, exposing its search and read functionalities via API endpoints. This is useful for integrating with other tools or services.

To start the server:

```bash
./devdocsmcp server [-port <port_number>] -lang <comma_separated_languages>
```

*   `-port`: Optional. The port number for the server to listen on. Defaults to `8080`.
*   `-lang`: Optional. A comma-separated list of language slugs that this server instance should serve (e.g., `html,css`). If omitted, all languages will be allowed.

**Example MCP Server Configuration:**

To configure `DevDocsMCP` as an MCP server, you can add a section like this to your MCP configuration file:

```json
{
  "mcpServers": {
    "devdocs-html": {
        "command": "/path/to/your/DevDocsMCP/cmd/devdocsmcp",
        "args":["server", "--lang", "html"]
    },
    "devdocs-css": {
        "command": "/path/to/your/DevDocsMCP/cmd/devdocsmcp",
        "args":["server", "--lang", "css"]
    },
    "devdocs-vite": {
        "command": "/path/to/your/DevDocsMCP/cmd/devdocsmcp",
        "args":["server", "--lang", "vite"]
    },
    "devdocs-tailwindcss": {
        "command": "/path/to/your/DevDocsMCP/cmd/devdocsmcp",
        "args":["server", "--lang", "tailwindcss"]
    },
    "devdocs-go": {
        "command": "/path/to/your/DevDocsMCP/cmd/devdocsmcp",
        "args":["server", "--lang", "go"]
    },
    "devdocs-mysql": {
        "command": "/path/to/your/DevDocsMCP/cmd/devdocsmcp",
        "args":["server", "--lang", "mysql"]
    },
    "devdocs-sqlite": {
        "command": "/path/to/your/DevDocsMCP/cmd/devdocsmcp",
        "args":["server", "--lang", "sqlite"]
    }
  }
}
```

**Note:** Replace `/path/to/your/DevDocsMCP/cmd/devdocsmcp` with the actual absolute path to your `devdocsmcp` executable. The key `"devdocs-html-css"` can be any unique identifier for this server.

### Display Allowed Languages

To display the languages that the `devdocsmcp` server is configured to allow:

```bash
./devdocsmcp allowed-langs
```

This command will show the languages that were specified with the `-lang` flag when the server was started. If no languages were specified, it will indicate that all languages are allowed.

## Download Pre-built Binaries
You can download pre-built binaries for various operating systems and architectures directly from the GitHub Releases page.

Replace `[VERSION]` with the desired release version (e.g., `v1.0.0`).

### Linux / macOS
```bash
# Download the binary (replace [OS] and [ARCH] with your system, e.g., linux_amd64, darwin_arm64)
wget https://github.com/kelvinzer0/DevDocsMCP/releases/download/[VERSION]/devdocsmcp_[OS]_[ARCH] -O devdocsmcp

# Make it executable
chmod +x devdocsmcp

# Move it to a directory in your PATH (e.g., /usr/local/bin)
sudo mv devdocsmcp /usr/local/bin/
```

### Windows
Download the appropriate `.exe` file from the GitHub Releases page (e.g., `devdocsmcp_windows_amd64.exe`).
Rename the downloaded file to `devdocsmcp.exe`.
Move `devdocsmcp.exe` to a directory that is included in your system's `PATH` environment variable. A common practice is to create a `bin` folder in your user directory (e.g., `C:\Users\YourUser\bin`) and add it to `PATH`.