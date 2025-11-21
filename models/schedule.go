package models

import "time"

type University struct {
	Name string `json:"name"`
}

type Course struct {
	Name       string `json:"name"`
	University string `json:"university"`
}

type ScheduleType struct {
	Name       string `json:"name"`
	University string `json:"university"`
	Course     string `json:"course"`
}

type ScheduleFile struct {
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"lastModified"`
	ETag         string    `json:"etag"`
	Version      string    `json:"version,omitempty"`
}

type PresignedURLResponse struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expiresAt"`
	FileName  string    `json:"fileName"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}
