package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/semmidev/phylax/internal/config"
	"github.com/semmidev/phylax/internal/infrastructure/logger"
	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// GDriveStorage implements the Storage interface for Google Drive.
type GDriveStorage struct {
	service  *drive.Service
	folderID string
	logger   *logger.Logger
}

// NewGDrive creates a new GDriveStorage instance.
// The cfg.FolderID must be non-empty; otherwise, an error is returned.
func NewGDrive(ctx context.Context, cfg *config.UploadTarget, oauthConfig *oauth2.Config, logger *logger.Logger) (*GDriveStorage, error) {
	if cfg == nil {
		return nil, errors.New("configuration cannot be nil")
	}
	if oauthConfig == nil {
		return nil, errors.New("oauth config cannot be nil")
	}
	if logger == nil {
		return nil, errors.New("logger cannot be nil")
	}
	if cfg.RefreshToken == "" {
		return nil, errors.New("refresh token is required")
	}
	if cfg.FolderID == "" {
		return nil, errors.New("folder ID is required and cannot be empty")
	}

	// Create token source with refresh token
	token := &oauth2.Token{
		RefreshToken: cfg.RefreshToken,
		TokenType:    "Bearer",
	}
	tokenSource := oauthConfig.TokenSource(ctx, token)

	// Initialize Google Drive service
	service, err := drive.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		logger.Errorf("Failed to create Google Drive service: %v", err)
		return nil, fmt.Errorf("failed to create drive service: %w", err)
	}

	logger.Infof("Initialized Google Drive storage with folder ID: %s", cfg.FolderID)
	return &GDriveStorage{
		service:  service,
		folderID: cfg.FolderID,
		logger:   logger,
	}, nil
}

// Upload uploads a file from localPath to Google Drive with the specified remoteName.
func (g *GDriveStorage) Upload(ctx context.Context, localPath, remoteName string) error {
	if remoteName == "" {
		return errors.New("remote file name cannot be empty")
	}
	if localPath == "" {
		return errors.New("local file path cannot be empty")
	}

	g.logger.Infof("Uploading file %s to Google Drive folder %s as %s", localPath, g.folderID, remoteName)

	file, err := os.Open(localPath)
	if err != nil {
		g.logger.Errorf("Failed to open file %s: %v", localPath, err)
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
		g.logger.Errorf("Failed to upload %s to Google Drive: %v", remoteName, err)
		return fmt.Errorf("failed to upload to Google Drive: %w", err)
	}

	g.logger.Infof("Successfully uploaded %s to Google Drive", remoteName)
	return nil
}

// List retrieves the names of files in the configured Google Drive folder.
func (g *GDriveStorage) List(ctx context.Context) ([]string, error) {
	query := fmt.Sprintf("'%s' in parents and trashed=false", sanitizeQuery(g.folderID))
	g.logger.Infof("Listing files in Google Drive folder %s", g.folderID)

	fileList, err := g.service.Files.List().
		Q(query).
		Fields("files(id, name, createdTime)").
		Context(ctx).
		Do()
	if err != nil {
		g.logger.Errorf("Failed to list files in folder %s: %v", g.folderID, err)
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	var files []string
	for _, file := range fileList.Files {
		if file.Name != "" {
			files = append(files, file.Name)
		}
	}

	g.logger.Infof("Found %d files in folder %s", len(files), g.folderID)
	return files, nil
}

// Delete removes a file with the specified remoteName from Google Drive.
func (g *GDriveStorage) Delete(ctx context.Context, remoteName string) error {
	if remoteName == "" {
		return errors.New("remote file name cannot be empty")
	}

	g.logger.Infof("Deleting file %s from Google Drive folder %s", remoteName, g.folderID)

	query := fmt.Sprintf("'%s' in parents and name='%s' and trashed=false",
		sanitizeQuery(g.folderID), sanitizeQuery(remoteName))

	fileList, err := g.service.Files.List().
		Q(query).
		Fields("files(id)").
		Context(ctx).
		Do()
	if err != nil {
		g.logger.Errorf("Failed to find file %s in folder %s: %v", remoteName, g.folderID, err)
		return fmt.Errorf("failed to find file: %w", err)
	}

	if len(fileList.Files) == 0 {
		g.logger.Warnf("File %s not found in folder %s", remoteName, g.folderID)
		return fmt.Errorf("file not found: %s", remoteName)
	}

	err = g.service.Files.Delete(fileList.Files[0].Id).Context(ctx).Do()
	if err != nil {
		g.logger.Errorf("Failed to delete file %s: %v", remoteName, err)
		return fmt.Errorf("failed to delete file: %w", err)
	}

	g.logger.Infof("Successfully deleted file %s", remoteName)
	return nil
}

// GetOldFiles retrieves the names of files created before cutoffTime.
func (g *GDriveStorage) GetOldFiles(ctx context.Context, cutoffTime time.Time) ([]string, error) {
	query := fmt.Sprintf("'%s' in parents and trashed=false and createdTime < '%s'",
		sanitizeQuery(g.folderID), cutoffTime.Format(time.RFC3339))
	g.logger.Infof("Listing files in folder %s older than %s", g.folderID, cutoffTime.Format(time.RFC3339))

	fileList, err := g.service.Files.List().
		Q(query).
		Fields("files(id, name)").
		Context(ctx).
		Do()
	if err != nil {
		g.logger.Errorf("Failed to list old files in folder %s: %v", g.folderID, err)
		return nil, fmt.Errorf("failed to list old files: %w", err)
	}

	var files []string
	for _, file := range fileList.Files {
		if file.Name != "" {
			files = append(files, file.Name)
		}
	}

	g.logger.Infof("Found %d old files in folder %s", len(files), g.folderID)
	return files, nil
}

// sanitizeQuery escapes single quotes in query strings to prevent injection.
func sanitizeQuery(input string) string {
	return strings.ReplaceAll(input, "'", "\\'")
}
