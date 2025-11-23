package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"ehang.io/nps/lib/crypt"
)

// Client structure for migration
type Client struct {
	Id              int    `json:"Id"`
	VerifyKey       string `json:"VerifyKey"`
	WebUserName     string `json:"WebUserName"`
	WebPassword     string `json:"WebPassword"`
	Remark          string `json:"Remark"`
	Status          bool   `json:"Status"`
	RateLimit       int    `json:"RateLimit"`
	MaxConn         int    `json:"MaxConn"`
	MaxTunnelNum    int    `json:"MaxTunnelNum"`
	ConfigConnAllow bool   `json:"ConfigConnAllow"`
	NoStore         bool   `json:"NoStore"`
	NoDisplay       bool   `json:"NoDisplay"`
}

func main() {
	fmt.Println("NPS Password Migration Tool")
	fmt.Println("===========================")
	fmt.Println("This tool will migrate plaintext passwords to bcrypt hashes.")
	fmt.Println()

	// Get the client file path
	var clientFilePath string
	if len(os.Args) > 1 {
		clientFilePath = os.Args[1]
	} else {
		clientFilePath = "conf/clients.json"
	}

	if !fileExists(clientFilePath) {
		fmt.Printf("Error: Client file not found at %s\n", clientFilePath)
		fmt.Println("Usage: migrate_passwords [path_to_clients.json]")
		os.Exit(1)
	}

	// Create backup
	backupPath := clientFilePath + ".backup"
	if err := copyFile(clientFilePath, backupPath); err != nil {
		fmt.Printf("Error creating backup: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Backup created at: %s\n", backupPath)

	// Read the file
	data, err := ioutil.ReadFile(clientFilePath)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Split by delimiter
	entries := strings.Split(string(data), "\n\r\n")
	var clients []Client
	var migratedCount int

	for _, entry := range entries {
		if strings.TrimSpace(entry) == "" {
			continue
		}

		var client Client
		if err := json.Unmarshal([]byte(entry), &client); err != nil {
			fmt.Printf("Warning: Failed to parse client entry: %v\n", err)
			continue
		}

		// Check if password needs migration
		if client.WebPassword != "" && !isBcryptHash(client.WebPassword) {
			// Migrate password
			hashedPassword, err := crypt.HashPassword(client.WebPassword)
			if err != nil {
				fmt.Printf("Warning: Failed to hash password for client %d: %v\n", client.Id, err)
				continue
			}

			fmt.Printf("✓ Migrated password for client ID %d (%s)\n", client.Id, client.WebUserName)
			client.WebPassword = hashedPassword
			migratedCount++
		}

		clients = append(clients, client)
	}

	// Write back to file
	file, err := os.Create(clientFilePath + ".tmp")
	if err != nil {
		fmt.Printf("Error creating temp file: %v\n", err)
		os.Exit(1)
	}

	for i, client := range clients {
		jsonData, err := json.Marshal(client)
		if err != nil {
			fmt.Printf("Error marshaling client %d: %v\n", client.Id, err)
			continue
		}

		file.Write(jsonData)
		if i < len(clients)-1 {
			file.Write([]byte("\n\r\n"))
		}
	}

	file.Sync()
	file.Close()

	// Replace original file
	if err := os.Rename(clientFilePath+".tmp", clientFilePath); err != nil {
		fmt.Printf("Error replacing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Printf("Migration completed successfully!\n")
	fmt.Printf("Total clients migrated: %d\n", migratedCount)
	fmt.Printf("Backup file: %s\n", backupPath)
}

func isBcryptHash(password string) bool {
	return len(password) == 60 && strings.HasPrefix(password, "$2a$")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func copyFile(src, dst string) error {
	data, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(dst, data, 0600)
}
