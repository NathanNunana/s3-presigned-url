package s3client

import (
	"bytes"
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Client struct {
	Client        *s3.Client
	PresignClient *s3.PresignClient
	Bucket        string
}

func NewS3Client(bucketName string) (*S3Client, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithSharedConfigProfile("rwanda"))
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(cfg)
	return &S3Client{Client: client, PresignClient: s3.NewPresignClient(client), Bucket: bucketName}, nil
}

func (s *S3Client) UploadImage(key string, fileData []byte) error {
	_, err := s.Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(fileData),
	})
	return err
}

func (s *S3Client) GeneratePresignedURL(key string, expiry time.Duration) (string, error) {
	req, err := s.PresignClient.PresignGetObject(context.TODO(), &s3.GetObjectInput{
		Bucket:                     aws.String(s.Bucket),
		Key:                        aws.String(key),
		ResponseContentDisposition: aws.String("inline"),
	}, func(po *s3.PresignOptions) {
		po.Expires = time.Duration(expiry)
	})
	if err != nil {
		return "", err
	}
	return req.URL, nil
}
