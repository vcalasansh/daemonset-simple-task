package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

type StatusCode string

const (
	StatusCodeSuccess StatusCode = "OK"
	StatusCodeFailed  StatusCode = "FAILED"
)

type Input struct {
	Message string `json:"message"`
}

type AssignRequest struct {
	ID   string `json:"id"`
	Data Input  `json:"data"`
}

type AssignResponse struct {
	Status StatusCode `json:"status"`
	Error  string     `json:"error"`
}

type RemoveRequest struct {
	ID string `json:"id"`
}

type RemoveResponse struct {
	Status StatusCode `json:"status"`
	Error  string     `json:"error"`
}

var (
	name  string = "daemonset-simple-task"
	port  string
	tasks map[string](chan bool)
	lock  sync.Mutex
)

func main() {
	// Define the port the server will listen on
	tasks = make(map[string](chan bool))
	port = *flag.String("port", "9000", "Port to listen on")
	flag.Parse()

	// Create a new ServeMux and register the /assign handler
	sm := http.NewServeMux()
	sm.HandleFunc("/assign", assign)
	sm.HandleFunc("/remove", remove)

	// Start the HTTP server
	fmt.Printf("Starting HTTP server on port %s\n", port)
	err := http.ListenAndServe(":"+port, sm)
	if err != nil {
		fmt.Println("Error starting HTTP server:", err)
	}
}

func assign(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(&AssignResponse{
			Status: StatusCodeFailed,
			Error:  "Unable to read request body",
		})
		return
	}
	defer r.Body.Close()
	fmt.Printf("Received payload: %s\n", string(body))
	input := new(AssignRequest)
	if err := json.Unmarshal(body, input); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(&AssignResponse{
			Status: StatusCodeFailed,
			Error:  "Invalid payload",
		})
		return
	}
	quit := make(chan bool)
	go func(msg string, taskId string) {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			select {
			case <-quit:
				finalMessage := fmt.Sprintf("Stopping to print the message \"%s\" on server %s. Task will be removed [taskId=%s]", msg, name, taskId)
				writeToFile(finalMessage)
				return
			default:
				finalMessage := fmt.Sprintf("%s : printed by server %s, running on %s [taskId=%s]", msg, name, port, taskId)
				writeToFile(finalMessage)
			}
		}
	}(input.Data.Message, input.ID)
	lock.Lock()
	tasks[input.ID] = quit
	lock.Unlock()
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&AssignResponse{Status: StatusCodeSuccess})
}

func remove(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(&RemoveResponse{
			Status: StatusCodeFailed,
			Error:  "Unable to read request body",
		})
		return
	}
	defer r.Body.Close()
	fmt.Printf("Received payload: %s\n", string(body))
	input := new(RemoveRequest)
	if err := json.Unmarshal(body, input); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(&RemoveResponse{
			Status: StatusCodeFailed,
			Error:  "Invalid payload",
		})
		return
	}
	quit, ok := tasks[input.ID]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(&RemoveResponse{
			Status: StatusCodeFailed,
			Error:  fmt.Sprintf("The task with ID %s does not exist!", input.ID),
		})
		return
	}
	go func() {
		quit <- true
	}()
	lock.Lock()
	delete(tasks, input.ID)
	lock.Unlock()
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&RemoveResponse{Status: StatusCodeSuccess})
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
