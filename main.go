package main

import (
	"bytes"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"s3/models"
)

type Storage struct {
	db  *gorm.DB
	cfg Config
}

func NewStorage(cfg Config) *Storage {
	return &Storage{cfg: cfg}
}

func (s *Storage) Connect() error {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		s.cfg.Host, s.cfg.Port, s.cfg.User, s.cfg.Password, s.cfg.Database)

	database, err := gorm.Open(postgres.Open(dsn))
	if err != nil {
		return err
	}

	s.db = database
	return nil
}

func (s *Storage) Close() error {
	if s.db != nil {
		sqlDB, err := s.db.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

func (s *Storage) Init() *gorm.DB {
	return s.db
}

func (s *Storage) MakeMigrations() error {
	if err := s.db.AutoMigrate(&models.File{}); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	return nil
}

func uploadToS3(filePath, bucket, key, accessKey, secretKey string) (string, error) {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(os.Getenv("S3_REGION")), // You might want to change this
		Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
		Endpoint:    aws.String(os.Getenv("S3_ENDPOINT")), // Yandex Object Storage endpoint
	})

	if err != nil {
		return "", err
	}

	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	fileInfo, _ := file.Stat()
	var size = fileInfo.Size()
	buffer := make([]byte, size)
	file.Read(buffer)

	_, err = s3.New(sess).PutObject(&s3.PutObjectInput{
		Bucket:        aws.String(bucket),
		Key:           aws.String(key),
		ACL:           aws.String("public-read"), // or "private" based on your requirement
		Body:          bytes.NewReader(buffer),
		ContentLength: aws.Int64(size),
		ContentType:   aws.String(http.DetectContentType(buffer)),
	})
	if err != nil {
		return "", err
	}

	return os.Getenv("S3_ENDPOINT") + "/" + bucket + "/" + key, nil
}

func InitializeDB() Storage {
	var cfg Config
	cfg.Database = os.Getenv("DATABASE_NAME")
	cfg.User = os.Getenv("DATABASE_USER")
	cfg.Password = os.Getenv("DATABASE_PASSWORD")
	cfg.Host = os.Getenv("DATABASE_HOST")
	cfg.Port = 5432

	storage := NewStorage(cfg)
	err := storage.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to database: %s", err)
	}

	err = storage.MakeMigrations()
	if err != nil {
		log.Fatalf("Failed to make migrations: %s", err)
	}

	return *storage
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	dir := os.Getenv("S3_DIR_PATH")
	bucket := os.Getenv("S3_BUCKET")
	accessKey := os.Getenv("S3_ACCESS_KEY")
	secretKey := os.Getenv("S3_SECRET_KEY")

	db := InitializeDB()

	defer db.Close()

	fileNameAndID := make(map[string]string)

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			relativePath := dir + path[len(dir):]
			url, err := uploadToS3(path, bucket, relativePath, accessKey, secretKey)
			if err != nil {
				log.Printf("Failed to upload %s: %s", path, err)
			} else {
				log.Printf("Uploaded %s to %s", path, url)
			}

			file := models.File{Url: url, IsConfirmed: true}
			db.Init().Create(&file)
			fileNameAndID[path[len(dir):]] = file.ID
		}
		return nil
	})

	if err != nil {
		log.Fatalf("Error walking through directory: %s", err)
	}

	fmt.Println("File name and ID:")
	for key, value := range fileNameAndID {
		fmt.Println(key, value)
	}
}
