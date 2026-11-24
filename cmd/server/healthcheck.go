package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

// healthcheck asks the server at url whether it is ready and returns the exit
// code: 0 for 200, 1 for anything else.
func healthcheck(url string) int {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Fprintln(os.Stderr, "healthcheck:", err)
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintln(os.Stderr, "healthcheck:", resp.Status)
		return 1
	}
	return 0
}
