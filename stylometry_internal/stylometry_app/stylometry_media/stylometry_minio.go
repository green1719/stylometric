package media

import (
	"context"
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func stylometryEnv(stylometry_key string, stylometry_fallback string) string {
	stylometry_value := strings.TrimSpace(os.Getenv(stylometry_key))
	if stylometry_value == "" {
		return stylometry_fallback
	}
	return stylometry_value
}

func stylometryMinIOEndpoint() (string, bool) {
	stylometry_raw_url := stylometryEnv("MINIO_INTERNAL_URL", "http://127.0.0.1:9000")
	stylometry_use_ssl := strings.HasPrefix(stylometry_raw_url, "https://")
	stylometry_raw_url = strings.TrimPrefix(stylometry_raw_url, "http://")
	stylometry_raw_url = strings.TrimPrefix(stylometry_raw_url, "https://")
	stylometry_raw_url = strings.TrimRight(stylometry_raw_url, "/")
	return stylometry_raw_url, stylometry_use_ssl
}

func stylometryPublicMediaURL(stylometry_object_name string) string {
	stylometry_public_url := strings.TrimRight(stylometryEnv("MINIO_PUBLIC_URL", "http://127.0.0.1:9000"), "/")
	stylometry_bucket := stylometryEnv("MINIO_BUCKET", "stylometry")
	return fmt.Sprintf("%s/%s/%s", stylometry_public_url, stylometry_bucket, stylometry_object_name)
}

func stylometryNewMinIOClient() (*minio.Client, error) {
	stylometry_endpoint, stylometry_use_ssl := stylometryMinIOEndpoint()
	return minio.New(stylometry_endpoint, &minio.Options{
		Creds: credentials.NewStaticV4(
			stylometryEnv("MINIO_ROOT_USER", "root"),
			stylometryEnv("MINIO_ROOT_PASSWORD", "rootpassword"),
			"",
		),
		Secure: stylometry_use_ssl,
	})
}

func stylometryLatinFileName(stylometry_folder string, stylometry_original_name string) string {
	stylometry_ext := strings.ToLower(filepath.Ext(stylometry_original_name))
	if stylometry_ext == "" {
		stylometry_ext = ".bin"
	}
	stylometry_timestamp := time.Now().Format("20060102_150405_000000000")
	return fmt.Sprintf("%s/stylometry_upload_%s%s", stylometry_folder, stylometry_timestamp, stylometry_ext)
}

func UploadFile(stylometry_ctx context.Context, stylometry_file_header *multipart.FileHeader, stylometry_folder string) (string, error) {
	stylometry_client, stylometry_err := stylometryNewMinIOClient()
	if stylometry_err != nil {
		return "", stylometry_err
	}

	stylometry_file, stylometry_err := stylometry_file_header.Open()
	if stylometry_err != nil {
		return "", stylometry_err
	}
	defer stylometry_file.Close()

	stylometry_bucket := stylometryEnv("MINIO_BUCKET", "stylometry")
	stylometry_object_name := stylometryLatinFileName(stylometry_folder, stylometry_file_header.Filename)
	stylometry_content_type := stylometry_file_header.Header.Get("Content-Type")
	if stylometry_content_type == "" {
		stylometry_content_type = "application/octet-stream"
	}

	_, stylometry_err = stylometry_client.PutObject(
		stylometry_ctx,
		stylometry_bucket,
		stylometry_object_name,
		stylometry_file,
		stylometry_file_header.Size,
		minio.PutObjectOptions{ContentType: stylometry_content_type},
	)
	if stylometry_err != nil {
		return "", stylometry_err
	}

	return stylometryPublicMediaURL(stylometry_object_name), nil
}
