package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"example/s3-demo/s3client"
	"example/s3-demo/store"

	"github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

var (
	s3Bucket    = "rwanda-meeting-room-images"
	db, _       = store.Connect()
	s3Client, _ = s3client.NewS3Client(s3Bucket)
)

type ImageMetadata struct {
	ID   *models.RecordID `json:"id"`
	Key  string           `json:"key"`
	Name string           `json:"name"`
	URL  string           `json:"url,omitempty"`
}

func uploadImage(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	fileData, _ := io.ReadAll(file)
	key := "golang"

	if err := s3Client.UploadImage(key, fileData); err != nil {
		http.Error(w, fmt.Sprintf("Failed to upload to S3, %v", err), http.StatusInternalServerError)
		return
	}

	image := ImageMetadata{Key: key, Name: "Golang Asset"}

	if _, err := surrealdb.Create[ImageMetadata](db, models.Table("files"), image); err != nil {
		http.Error(w, fmt.Sprintf("Failed to store image metadata %v", err), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Image uploaded and metadata stored.")
}

func getPresignedURL(w http.ResponseWriter, r *http.Request) {
	ids, ok := r.URL.Query()["id"]
	if !ok || len(ids[0]) < 1 {
		http.Error(w, "Missing 'id' parameter", http.StatusBadRequest)
		return
	}

	id := ids[0]

	imageId := models.ParseRecordID(fmt.Sprintf("files:%s", id))

	image, err := surrealdb.Select[ImageMetadata](db, *imageId)
	if err != nil {
		http.Error(w, "Image not found", http.StatusNotFound)
		return
	}

	url, err := s3Client.GeneratePresignedURL(image.Key, 15*time.Minute)
	if err != nil {
		http.Error(w, "Failed to generate presigned URL", http.StatusInternalServerError)
		return
	}

	image.URL = url

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(image)
}

func main() {
	http.Handle("/", http.FileServer(http.Dir(".")))
	http.HandleFunc("/upload", uploadImage)
	http.HandleFunc("/get-url", getPresignedURL)
	log.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
