package main

import (
	"bufio"
	"os"
)

func readLines(filePath string) ([]string, error) { // nolint:all
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close() //nolint:errcheck

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines, scanner.Err()
}
