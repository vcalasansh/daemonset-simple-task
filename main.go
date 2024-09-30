package main

import (
	"fmt"
	"net/http"
	"os"

	"daemon-set-example.com/logger"
	"github.com/sirupsen/logrus"
)

type Server struct {
	handler *Handler
}

func NewServer(handler *Handler) *Server {
	return &Server{handler: handler}
}

func (s *Server) StartServer(port string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/tasks", s.handler.HandleTasks)

	logrus.Infof("daemon server is running on port %s", port)
	if err := http.ListenAndServe(port, mux); err != nil {
		fmt.Printf("error starting server: %v", err)
	}
}

func main() {
	logger.SetLogrus()
	port := os.Getenv("DAEMON_SERVER_PORT")
	if port == "" {
		fmt.Printf("environment variable DAEMON_SERVER_PORT is not set. Cannot start server")
		return
	}
	handler := &Handler{port: port, tasks: make(map[string]chan bool)}
	server := NewServer(handler)
	server.StartServer(":" + port)
}
