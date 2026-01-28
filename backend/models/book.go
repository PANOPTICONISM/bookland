package models

import "time"

type Book struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Author      string    `json:"author"`
	CoverPath   string    `json:"coverPath"`
	FilePath    string    `json:"filePath"`
	FileSize    int64     `json:"fileSize"`
	FileType    string    `json:"fileType"`
	AddedAt     time.Time `json:"addedAt"`
}
