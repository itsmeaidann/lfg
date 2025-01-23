package s3client

import (
	"bytes"
	"io"

	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// Init initializes the S3 client using AWS credentials from environment variables
func Init(awsAccessKey, awsSecretKey string) *s3.S3 {
	if awsAccessKey == "" || awsSecretKey == "" {
		log.Fatal("AWS_ACCESS_KEY and AWS_SECRET_KEY must be set")
	}

	sess, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(awsAccessKey, awsSecretKey, ""),
		Region:      aws.String("ap-southeast-1"),
	})

	if err != nil {
		log.Fatalf("failed to create session: %v", err)
	}

	S3Client := s3.New(sess)
	return S3Client
}

// GetObject retrieves an object from S3
func GetObject(s3Client *s3.S3, bucket, key string) ([]byte, error) {

	result, err := s3Client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	defer result.Body.Close()

	body, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func UploadObject(s3Client *s3.S3, bucket, key string, body []byte) error {
	_, err := s3Client.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(body),
	})
	return err
}
