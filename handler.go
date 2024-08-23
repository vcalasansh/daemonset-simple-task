package main

import (
	"encoding/base64"
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
	tasks map[string]chan bool
	lock  sync.Mutex
}

var (
	name       string = "daemonset-simple-task"
	outputPath string = "output.txt"
)

func (h *Handler) HandleTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.Assign(w, r)
	case http.MethodDelete:
		h.Remove(w, r)
	case http.MethodGet:
		h.GetTasks(w, r)
	default:
		sendErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (h *Handler) Assign(w http.ResponseWriter, r *http.Request) {
	var tasks Tasks
	if err := parseRequest(r, &tasks); err != nil {
		sendErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	for _, task := range tasks.Tasks {
		var params Params
		if err := parseParams(&task.EncodedParams, &params); err != nil {
			sendErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}
		quit := make(chan bool)
		go h.startTask(params.Message, task.ID, quit)

		h.lock.Lock()
		h.tasks[task.ID] = quit
		h.lock.Unlock()
	}

	sendSuccessResponse(w, DaemonSetResponse{Status: StatusCodeSuccess})
}

func (h *Handler) Remove(w http.ResponseWriter, r *http.Request) {
	taskIds, ok := r.URL.Query()["taskIds"]
	if !ok || len(taskIds) < 1 {
		sendErrorResponse(w, http.StatusBadRequest, "Task IDs are required")
		return
	}

	h.lock.Lock()
	defer h.lock.Unlock()

	for _, taskId := range taskIds {
		if quit, exists := h.tasks[taskId]; exists {
			go func(quit chan bool) { quit <- true }(quit)
			delete(h.tasks, taskId)
		} else {
			sendErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("The task with ID %s does not exist!", taskId))
			return
		}
	}

	sendSuccessResponse(w, DaemonSetResponse{Status: StatusCodeSuccess})
}

func (h *Handler) GetTasks(w http.ResponseWriter, r *http.Request) {
	h.lock.Lock()
	defer h.lock.Unlock()

	var tasks []Task
	for id := range h.tasks {
		tasks = append(tasks, Task{ID: id})
	}

	response := Tasks{Tasks: tasks}
	sendSuccessResponse(w, response)
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	sendSuccessResponse(w, map[string]string{"status": string(StatusCodeSuccess)})
}

func (h *Handler) startTask(message string, taskID string, quit chan bool) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-quit:
			finalMessage := fmt.Sprintf("Stopping to print the message \"%v\" on server %s. Task will be removed [taskId=%s]", message, name, taskID)
			writeToFile(finalMessage)
			return
		case <-ticker.C:
			finalMessage := fmt.Sprintf("%v : printed by server %s, running on %s [taskId=%s]", message, name, h.port, taskID)
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

func parseParams(e *EncodedParams, p *Params) error {
	// decode base64 data
	decoded := make([]byte, base64.StdEncoding.DecodedLen(len(e.Base64Data)))
	n, err := base64.StdEncoding.Decode(decoded, e.Base64Data)
	if err != nil {
		return fmt.Errorf("failed to decode EncodedParams.Base64Data with base64: %w", err)
	}
	decoded = decoded[:n]
	// unmarshall decoded data into `Params` type
	if err := json.Unmarshal(decoded, p); err != nil {
		return fmt.Errorf("decoded value of EncodedParams.Base64Data is not valid Params type: %w", err)
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
