package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

type Handler struct {
	port  string
	tasks map[string](chan bool)
	lock  sync.Mutex
}

var (
	name       string = "daemonset-simple-task"
	outputPath string = "output.txt"
)

func (h *Handler) Assign(w http.ResponseWriter, r *http.Request) {
	var input AssignRequest
	if err := parseRequest(r, &input); err != nil {
		sendErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	quit := make(chan bool)
	go h.startTask(input.Data.Message, input.ID, quit)

	h.lock.Lock()
	h.tasks[input.ID] = quit
	h.lock.Unlock()

	sendSuccessResponse(w, AssignResponse{Status: StatusCodeSuccess})
}

func (h *Handler) Remove(w http.ResponseWriter, r *http.Request) {
	var input RemoveRequest
	if err := parseRequest(r, &input); err != nil {
		sendErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	h.lock.Lock()
	quit, ok := h.tasks[input.ID]
	h.lock.Unlock()
	if !ok {
		sendErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("The task with ID %s does not exist!", input.ID))
		return
	}

	go func() { quit <- true }()
	h.lock.Lock()
	delete(h.tasks, input.ID)
	h.lock.Unlock()

	sendSuccessResponse(w, RemoveResponse{Status: StatusCodeSuccess})
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	sendSuccessResponse(w, map[string]string{"status": string(StatusCodeSuccess)})
}

func (h *Handler) startTask(message, taskID string, quit chan bool) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-quit:
			finalMessage := fmt.Sprintf("Stopping to print the message \"%s\" on server %s. Task will be removed [taskId=%s]", message, name, taskID)
			writeToFile(finalMessage)
			return
		case <-ticker.C:
			finalMessage := fmt.Sprintf("%s : printed by server %s, running on %s [taskId=%s]", message, name, h.port, taskID)
			writeToFile(finalMessage)
		}
	}
}

func parseRequest(r *http.Request, v interface{}) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("unable to read request body")
	}
	defer r.Body.Close()

	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("invalid payload")
	}
	return nil
}

func sendErrorResponse(w http.ResponseWriter, status int, errMsg string) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"status": string(StatusCodeFailed),
		"error":  errMsg,
	})
}

func sendSuccessResponse(w http.ResponseWriter, response interface{}) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func writeToFile(message string) {
	file, err := os.OpenFile(outputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error opening or creating the file: %v\n", err)
		return
	}
	defer file.Close()

	if _, err := file.WriteString(message + "\n"); err != nil {
		fmt.Printf("Error writing to the file: %v\n", err)
	}
}
