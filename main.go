package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/http/fcgi"
	"os"
	"time"
)

type input struct {
	message string `json:"message"`
}

var (
	name string = "simple_task_handler"
	port string
)

func main() {
	port := flag.String("port", "9000", "Port to listen on")
	flag.Parse()
	sm := http.NewServeMux()
	sm.HandleFunc("/assign", assign)
	listener, _ := net.Listen("tcp", "127.0.0.1:"+*port)
	err := fcgi.Serve(listener, sm)
	if err != nil {
		fmt.Println("Error starting FastCGI server:", err)
	}
}

func assign(w http.ResponseWriter, r *http.Request) {
	// unmarshal the input
	in := new(input)
	json.NewDecoder(r.Body).Decode(in)
	message := in.message

	go func(msg string) {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			finalMessage := fmt.Sprintf("%s : printed by server %s, running on %s", msg, name, port)
			writeToFile(finalMessage)
		}
	}(message)

	fmt.Fprintln(w, "Job started to print message every 10 seconds and call web server.")
}

func writeToFile(message string) {
	// The path to the file
	filePath := "output.txt"

	// Open the file in append mode, create it if it doesn't exist
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error opening or creating the file: %v\n", err)
		return
	}
	defer file.Close()

	// Write the message to the file
	_, err = file.WriteString(message + "\n")
	if err != nil {
		fmt.Printf("Error writing to the file: %v\n", err)
		return
	}

	fmt.Printf("Successfully wrote to the file: %s\n", filePath)
}
