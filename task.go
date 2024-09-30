package main

import (
	"time"

	"github.com/sirupsen/logrus"
)

type Params struct {
	Message string `json:"message"`
}

func startTask(taskId string, params Params, quit chan bool) {
	message := params.Message
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-quit:
			return
		case <-ticker.C:
			logrus.Infof("%s [taskId=%s]", message, taskId)
		}
	}
}
