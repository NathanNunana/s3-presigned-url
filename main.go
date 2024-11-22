package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"example/s3-demo/s3client"
	"example/s3-demo/store"

	"github.com/joho/godotenv"
	"github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

var (
	s3Bucket   = "cloudfront-demo-asset"
	awsProfile = "training-account"
	db         *surrealdb.DB
	s3Client   *s3client.S3Client
)

func Initializer() {
	log.Println("Loading environment variables...")
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	keyID := os.Getenv("KEY_ID")
	cfPrivKey := os.Getenv("CF_PRIV_KEY")
	cfPrivKeyPath := os.Getenv("PRIV_KEY_PATH")

	if keyID == "" || cfPrivKey == "" || cfPrivKeyPath == "" {
		log.Fatal("Missing required environment variables")
	}

	log.Printf("Environment Variables Loaded: KEY_ID=%s, CF_PRIV_KEY=%s", keyID, "****")

	var errS3 error
	s3Client, errS3 = s3client.NewS3Client(
		s3Bucket,
		awsProfile,
		os.Getenv("KEY_ID"),
		os.Getenv("CF_PRIV_KEY"),
	)
	if errS3 != nil {
		log.Fatalf("Error creating S3 Client: %v", errS3)
	}

	var errDB error
	db, errDB = store.Connect()
	if errDB != nil {
		log.Fatalf("Error connecting to DB: %v", errDB)
	}
}

type ImageMetadata struct {
	ID   *models.RecordID `json:"id,omitempty"`
	Key  string           `json:"key"`
	Name string           `json:"name"`
	URL  string           `json:"url,omitempty"`
}

func uploadImage(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read file: %v", err), http.StatusBadRequest)
		return
	}
	defer file.Close()

	fileData, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read file data: %v", err), http.StatusInternalServerError)
		return
	}

	key := "golang"

	if err := s3Client.UploadImage(key, fileData); err != nil {
		http.Error(w, fmt.Sprintf("Failed to upload to S3: %v", err), http.StatusInternalServerError)
		return
	}

	image := ImageMetadata{Key: key, Name: "Golang Asset"}

	if _, err := surrealdb.Create[ImageMetadata](db, models.Table("files"), image); err != nil {
		http.Error(w, fmt.Sprintf("Failed to store image metadata: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Image uploaded and metadata stored.")
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

func getCloudFrontSignedURL(w http.ResponseWriter, r *http.Request) {
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

	url, err := s3Client.GenerateCloudFrontsignedURL(image.Key, time.Now().Add(time.Minute*15))
	if err != nil {
		http.Error(w, "Failed to generate presigned URL", http.StatusInternalServerError)
		return
	}

	image.URL = url

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(image)
}

func main() {
	Initializer()
	http.Handle("/", http.FileServer(http.Dir(".")))
	http.HandleFunc("/upload", uploadImage)
	http.HandleFunc("/get-url", getPresignedURL)
	http.HandleFunc("/get-url-cf", getCloudFrontSignedURL)
	log.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
