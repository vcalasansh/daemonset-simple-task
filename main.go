package main

import (
	"flag"
	"fmt"
	"net/http"
)

type StatusCode string

const (
	StatusCodeSuccess StatusCode = "OK"
	StatusCodeFailed  StatusCode = "FAILED"
)

type Params struct {
	Message string `json:"message"`
}

type Task struct {
	ID     string `json:"id"`
	Params Params `json:"params"`
}

type Tasks struct {
	Tasks []Task `json:"tasks"`
}

type DaemonSetResponse struct {
	Status StatusCode `json:"status"`
	Error  string     `json:"error,omitempty"`
}

type Server struct {
	handler *Handler
}

func NewServer(handler *Handler) *Server {
	return &Server{handler: handler}
}

func (s *Server) StartServer(port string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/tasks", s.handler.HandleTasks)

	fmt.Printf("Server is running on port %s\n", port)
	if err := http.ListenAndServe(port, mux); err != nil {
		fmt.Printf("Error starting server: %v", err)
	}
}

func main() {
	port := flag.String("port", "9000", "Port to listen on")
	flag.Parse()
	handler := &Handler{port: *port, tasks: make(map[string]chan bool)}
	server := NewServer(handler)
	server.StartServer(":" + *port)
}
