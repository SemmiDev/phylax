package storage

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/semmidev/phylax/internal/config"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type GDriveStorage struct {
	service  *drive.Service
	folderID string
}

func NewGDrive(cfg *config.UploadTarget) (*GDriveStorage, error) {
	ctx := context.Background()

	service, err := drive.NewService(ctx, option.WithCredentialsFile(cfg.CredentialsFile))
	if err != nil {
		return nil, fmt.Errorf("failed to create drive service: %w", err)
	}

	return &GDriveStorage{
		service:  service,
		folderID: cfg.FolderID,
	}, nil
}

func (g *GDriveStorage) Upload(ctx context.Context, localPath string, remoteName string) error {
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	fileMetadata := &drive.File{
		Name:    remoteName,
		Parents: []string{g.folderID},
	}

	_, err = g.service.Files.Create(fileMetadata).
		Media(file).
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("failed to upload to gdrive: %w", err)
	}

	return nil
}

func (g *GDriveStorage) List(ctx context.Context) ([]string, error) {
	query := fmt.Sprintf("'%s' in parents and trashed=false", g.folderID)

	fileList, err := g.service.Files.List().
		Q(query).
		Fields("files(id, name, createdTime)").
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	var files []string
	for _, file := range fileList.Files {
		files = append(files, file.Name)
	}

	return files, nil
}

func (g *GDriveStorage) Delete(ctx context.Context, remoteName string) error {
	query := fmt.Sprintf("'%s' in parents and name='%s' and trashed=false", g.folderID, remoteName)

	fileList, err := g.service.Files.List().
		Q(query).
		Fields("files(id)").
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("failed to find file: %w", err)
	}

	if len(fileList.Files) == 0 {
		return fmt.Errorf("file not found: %s", remoteName)
	}

	err = g.service.Files.Delete(fileList.Files[0].Id).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

func (g *GDriveStorage) GetOldFiles(ctx context.Context, cutoffTime time.Time) ([]string, error) {
	query := fmt.Sprintf("'%s' in parents and trashed=false and createdTime < '%s'",
		g.folderID,
		cutoffTime.Format(time.RFC3339))

	fileList, err := g.service.Files.List().
		Q(query).
		Fields("files(id, name)").
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list old files: %w", err)
	}

	var files []string
	for _, file := range fileList.Files {
		files = append(files, file.Name)
	}

	return files, nil
}
