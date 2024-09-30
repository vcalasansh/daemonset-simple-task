package main

type (
	EncodedParams struct {
		Base64Data []byte `json:"binary_data"`
	}

	Task struct {
		ID            string            `json:"id"`
		EncodedParams EncodedParams     `json:"params"`
		Secrets       map[string]string `json:"secrets"`
	}

	Tasks struct {
		Tasks []Task `json:"tasks"`
	}

	TaskMetadata struct {
		ID string `json:"id"`
	}

	TasksMetadata []TaskMetadata

	Response struct {
		TasksMetadata TasksMetadata `json:"tasks_metadata"`
		Error         string        `json:"error,omitempty"`
	}
)
