package main

import (
	"bufio"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	_ "github.com/go-sql-driver/mysql"
)

const (
	checkMark = "✓"
)

type MySQLConfig struct {
	User     string
	Password string
	Host     string
}

type Process struct {
	ID      int64
	User    string
	Host    string
	DB      sql.NullString
	Command string
	Time    int
	State   sql.NullString
	Info    sql.NullString
}

func readMySQLConfig() MySQLConfig {
	home, err := os.UserHomeDir()
	if err != nil {
		return MySQLConfig{}
	}

	configPath := filepath.Join(home, ".my.cnf")
	content, err := os.ReadFile(configPath)
	if err != nil {
		return MySQLConfig{}
	}

	config := MySQLConfig{}
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "user=") {
			config.User = strings.TrimPrefix(line, "user=")
		} else if strings.HasPrefix(line, "password=") {
			config.Password = strings.TrimPrefix(line, "password=")
		} else if strings.HasPrefix(line, "host=") {
			config.Host = strings.TrimPrefix(line, "host=")
		}
	}
	return config
}

func testConnection(db *sql.DB, host string) error {
	err := db.Ping()
	if err != nil {
		return err
	}

	green := color.New(color.FgGreen)
	green.Printf("Connected successfully to %s %s\n", host, checkMark)
	return nil
}

func formatProcessOutput(p Process, useColor bool) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	stateColor := color.New(color.FgYellow)
	infoColor := color.New(color.FgCyan)
	if useColor {
		switch {
		case p.State.String == "login":
			stateColor = color.New(color.FgRed)
		case p.State.String == "Receiving from client":
			stateColor = color.New(color.FgBlue)
		case strings.Contains(strings.ToLower(p.Info.String), "select"):
			if strings.Contains(strings.ToLower(p.Info.String), "count(*)") {
				infoColor = color.New(color.FgMagenta, color.Bold)
			} else if strings.Contains(strings.ToLower(p.Info.String), "limit") {
				infoColor = color.New(color.FgGreen, color.Bold)
			} else {
				infoColor = color.New(color.FgCyan, color.Bold)
			}
			stateColor = color.New(color.FgGreen)
		case strings.Contains(strings.ToLower(p.Info.String), "insert"):
			infoColor = color.New(color.FgGreen, color.Bold)
		case strings.Contains(strings.ToLower(p.Info.String), "update"):
			infoColor = color.New(color.FgYellow, color.Bold)
		case strings.Contains(strings.ToLower(p.Info.String), "delete"):
			infoColor = color.New(color.FgRed, color.Bold)
		case strings.Contains(strings.ToLower(p.Info.String), "create") ||
			strings.Contains(strings.ToLower(p.Info.String), "alter") ||
			strings.Contains(strings.ToLower(p.Info.String), "drop"):
			infoColor = color.New(color.FgMagenta, color.Bold)
		}
	}

	header := fmt.Sprintf("*************************** Process Info @ %s ***************************\n", timestamp)
	info := fmt.Sprintf("       ID: %d\n"+
		"     USER: %s\n"+
		"     HOST: %s\n"+
		"       DB: %s\n"+
		"  COMMAND: %s\n"+
		"     TIME: %d\n"+
		"    STATE: %s\n"+
		"     INFO: %s\n\n",
		p.ID, p.User, p.Host, p.DB.String, p.Command, p.Time,
		stateColor.SprintFunc()(p.State.String),
		infoColor.SprintFunc()(p.Info.String))

	return header + info
}

func isMonitoringQuery(info string) bool {
	// Check if this is our own monitoring query
	return strings.Contains(info, "FROM information_schema.processlist") &&
		strings.Contains(info, "WHERE command != 'Sleep'")
}

func main() {
	hostFlag := flag.String("h", "", "MySQL host address")
	fileFlag := flag.String("f", "", "Output file name (without date)")
	sleepFlag := flag.Int("s", 1, "Sleep duration in nanoseconds (default: 1)")
	queryFlag := flag.Bool("q", false, "Show only queries (SELECT statements)")
	debugFlag := flag.Bool("d", false, "Debug mode - show all queries with timing")
	verboseFlag := flag.Bool("v", false, "Verbose debug mode")
	flag.Parse()

	// Read MySQL config
	config := readMySQLConfig()

	// Determine host
	host := *hostFlag
	if host == "" {
		if config.Host != "" {
			host = config.Host
		} else {
			host = "localhost"
		}
	}

	// Build DSN
	dsn := fmt.Sprintf("%s:%s@tcp(%s:3306)/",
		config.User,
		config.Password,
		host,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Test connection and show status
	if err := testConnection(db, host); err != nil {
		panic(fmt.Sprintf("Failed to connect to %s: %v", host, err))
	}

	// Add debug counter
	queryCount := 0
	lastCheck := time.Now()

	for {
		// Determine filename
		var filename string
		date := time.Now().Format("2006-01-02")
		if *fileFlag != "" {
			filename = *fileFlag + "-" + date + ".txt"
		} else {
			filename = "load_test-" + date + ".txt"
		}

		file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}

		// Use buffered writer for better performance
		writer := bufio.NewWriter(file)

		// Query and write process list
		processes, err := getProcessList(db)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			file.Close()
			continue
		}

		// Write each process to file
		for _, p := range processes {
			info := p.Info.String
			isQuery := strings.Contains(info, "select") ||
				strings.Contains(info, "insert") ||
				strings.Contains(info, "update") ||
				strings.Contains(info, "delete") ||
				strings.Contains(info, "create") ||
				strings.Contains(info, "alter") ||
				strings.Contains(info, "drop")

			// Skip our own monitoring query unless in debug mode
			if !*debugFlag && isMonitoringQuery(p.Info.String) {
				continue
			}

			// Enhanced query detection
			if *queryFlag && !isQuery {
				continue
			}

			if *verboseFlag {
				queryType := "unknown"
				if strings.Contains(strings.ToLower(info), "select") {
					queryType = "SELECT"
				} else if strings.Contains(strings.ToLower(info), "show") {
					queryType = "SHOW"
				}

				fmt.Printf("Debug: Found %s query - State: %s, Time: %d, Info: %.100s...\n",
					queryType, p.State.String, p.Time, info)
			}

			if *debugFlag && (strings.Contains(info, "select") || strings.Contains(info, "count(") ||
				strings.Contains(info, "limit")) {
				queryCount++
				fmt.Printf("Debug: Query #%d detected: %.100s...\nState: %s, Time: %d\n\n",
					queryCount, p.Info.String, p.State.String, p.Time)
			}

			// Write to file without colors
			fileOutput := formatProcessOutput(p, false)
			_, err := writer.WriteString(fileOutput)
			if err != nil {
				fmt.Printf("Error writing to file: %v\n", err)
				continue
			}

			// Print to terminal with colors
			fmt.Print(formatProcessOutput(p, true))
		}

		// Print stats every 5 seconds in debug mode
		if *debugFlag && time.Since(lastCheck) > 5*time.Second {
			fmt.Printf("Stats: Captured %d queries in last 5 seconds\n", queryCount)
			queryCount = 0
			lastCheck = time.Now()
		}

		// Flush the buffer to ensure all data is written
		writer.Flush()
		file.Close()

		// Use nanosecond sleep duration
		time.Sleep(time.Duration(*sleepFlag) * time.Nanosecond)
	}
}

// Remove currentUser parameter since it's no longer used
func getProcessList(db *sql.DB) ([]Process, error) {
	query := `SELECT ID, USER, HOST, DB, COMMAND, TIME, STATE, INFO 
			 FROM information_schema.processlist 
			 WHERE command != 'Sleep'
			 AND (COMMAND = 'Query' 
				  OR INFO IS NOT NULL
				  OR STATE NOT IN ('', 'init', 'after create', 'CONNECTING')
				  OR TIME > 0)
			 ORDER BY TIME DESC`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var processes []Process
	for rows.Next() {
		var p Process
		err := rows.Scan(&p.ID, &p.User, &p.Host, &p.DB, &p.Command, &p.Time, &p.State, &p.Info)
		if err != nil {
			return nil, err
		}
		processes = append(processes, p)
	}
	return processes, nil
}
