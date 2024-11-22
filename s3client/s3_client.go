package s3client

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/cloudfront/sign"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Client struct {
	Client                  *s3.Client
	PresignClient           *s3.PresignClient
	CloudFrontPresignClient *sign.URLSigner
	Bucket                  string
}

func rsaParser(str string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(str))
	if block == nil || block.Type != "PRIVATE KEY" {
		return nil, fmt.Errorf("failed to decode PEM block containing private key\n")
	}

	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse RSA private key: %s\n", err.Error())
	}

	return (privateKey).(*rsa.PrivateKey), nil
}

func NewS3Client(bucketName, awsProfile, cfKeyID, privKey string) (*S3Client, error) {
	fmt.Println(cfKeyID, cfKeyID)
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithSharedConfigProfile(awsProfile))
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(cfg)

	// block, _ := os.Open(privKeyPath)
	// if block == nil {
	// 	return nil, fmt.Errorf("failed to open block")
	// }

	// privKey, err := sign.LoadPEMPrivKey(block)
	// if err != nil {
	// 	return nil, fmt.Errorf("Failed to load private key, err: %s\n", err.Error())
	// }
	rsaPrivKey, err := rsaParser(privKey)
	if err != nil {
		return nil, err
	}

	return &S3Client{
		Client:                  client,
		PresignClient:           s3.NewPresignClient(client),
		CloudFrontPresignClient: sign.NewURLSigner(cfKeyID, rsaPrivKey),
		Bucket:                  bucketName,
	}, nil
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

func (s *S3Client) GenerateCloudFrontsignedURL(key string, expiry time.Time) (string, error) {
	url := fmt.Sprintf("https://d3teqayz0fq1v6.cloudfront.net/%s", key)
	signedURL, err := s.CloudFrontPresignClient.Sign(url, expiry)
	if err != nil {
		return "", err
	}
	return signedURL, nil
}
