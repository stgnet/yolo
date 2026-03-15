package main

import (
	"io"
	"log"
	"os"
)

// ensureCurDir creates the current directory if it doesn't exist and returns its path
func ensureCurDir() (string, error) {
	err := os.MkdirAll("cur", 0755)
	if err != nil {
		return "", err
	}
	return "cur", nil
}

// processEmails scans the inbox for unread emails and processes them
func processEmails(reader io.Reader) (int, error) {
	curDir, err := ensureCurDir()
	if err != nil {
		log.Println("Failed to create cur directory:", err)
		return 0, err
	}

	totalProcessed := 0
	count := 0
	for count < 5 {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return totalProcessed, err
		}
		count++
		totalProcessed++

		newPath := curDir + "/email_" + string(rune(count))
		err = os.WriteFile(newPath, []byte(line), 0644)
		if err != nil {
			log.Println("Failed to write email file:", err)
			return totalProcessed, err
		}
	}

	return totalProcessed, nil
}
