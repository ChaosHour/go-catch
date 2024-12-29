# Go-Catch - MySQL Process Monitor

A lightweight MySQL process monitoring tool written in Go that helps you track and analyze database queries in real-time.

## Features

- Real-time monitoring of MySQL processes
- Colorized output for different query types
- Query filtering capabilities
- File-based logging
- Support for MySQL configuration file (.my.cnf)
- Debug and verbose modes for detailed analysis

## Installation

```bash
git clone https://github.com/yourusername/go-catch.git
cd go-catch
go build
```

## Configuration

The tool reads MySQL credentials from your `.my.cnf` file in your home directory. Example format:

```ini
user=yourusername
password=yourpassword
host=localhost
```

## Usage

```bash
./go-catch [options]

Options:
  -h string
        MySQL host address (default: from .my.cnf or "localhost")
  -f string
        Output file name (without date)
  -s int
        Sleep duration in nanoseconds (default: 1)
  -q    
        Show only queries (SELECT statements)
  -d    
        Debug mode - show all queries with timing
  -v    
        Verbose debug mode
```

## Examples

1. Basic monitoring:
```bash
./go-catch
```

2. Monitor specific host with debug mode:
```bash
./go-catch -h mydb.example.com -d
```

3. Monitor queries only with custom output file:
```bash
./go-catch -q -f mydb_queries
```

## Output

The tool provides both console output (with colors) and file logging. Each process is displayed with:
- Process ID
- User
- Host
- Database
- Command
- Execution Time
- State
- Query Info

## Color Coding

- SELECT queries: Cyan
- INSERT queries: Green
- UPDATE queries: Yellow
- DELETE queries: Red
- DDL queries (CREATE/ALTER/DROP): Magenta
- Count queries: Magenta (bold)
- Queries with LIMIT: Green (bold)

## Requirements

- Go 1.16 or higher
- MySQL 5.7 or higher
- Appropriate MySQL user privileges to access process list


## Reason for this project
I needed a simple tool to get queries from one instance of MySQL so I can apply them to another instance that I am testing for upgrades on GCP CloudSQL MySQL 8.0.