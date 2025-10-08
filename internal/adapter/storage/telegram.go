package storage

import (
	"context"
	"fmt"
	"os"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/semmidev/phylax/internal/config"
)

type TelegramStorage struct {
	bot        *tgbotapi.BotAPI
	chatID     int64
	sendFile   bool
	notifyOnly bool
}

func NewTelegram(cfg *config.UploadTarget) (*TelegramStorage, error) {
	bot, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot: %w", err)
	}

	var chatID int64
	fmt.Sscanf(cfg.ChatID, "%d", &chatID)

	return &TelegramStorage{
		bot:        bot,
		chatID:     chatID,
		sendFile:   cfg.SendFile,
		notifyOnly: cfg.NotifyOnly,
	}, nil
}

func (t *TelegramStorage) Upload(ctx context.Context, localPath string, remoteName string) error {
	fileInfo, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	fileSizeMB := float64(fileInfo.Size()) / (1024 * 1024)

	if t.notifyOnly || !t.sendFile || fileSizeMB > 50 {
		// Send notification only
		message := fmt.Sprintf(
			"✅ Backup Created\n\n"+
				"📁 File: %s\n"+
				"📊 Size: %.2f MB\n"+
				"🕐 Time: %s",
			remoteName,
			fileSizeMB,
			fileInfo.ModTime().Format("2006-01-02 15:04:05"),
		)

		msg := tgbotapi.NewMessage(t.chatID, message)
		_, err = t.bot.Send(msg)
		if err != nil {
			return fmt.Errorf("failed to send telegram notification: %w", err)
		}
	} else {
		// Send file (for files < 50MB)
		file := tgbotapi.NewDocument(t.chatID, tgbotapi.FilePath(localPath))
		file.Caption = fmt.Sprintf("📦 Backup: %s (%.2f MB)", remoteName, fileSizeMB)

		_, err = t.bot.Send(file)
		if err != nil {
			return fmt.Errorf("failed to send telegram file: %w", err)
		}
	}

	return nil
}

func (t *TelegramStorage) List(ctx context.Context) ([]string, error) {
	// Telegram doesn't support listing files
	return []string{}, nil
}

func (t *TelegramStorage) Delete(ctx context.Context, remoteName string) error {
	// Telegram doesn't support deleting files
	return nil
}

func (t *TelegramStorage) GetOldFiles(ctx context.Context, cutoffTime time.Time) ([]string, error) {
	// Telegram doesn't support getting old files
	return []string{}, nil
}

func (t *TelegramStorage) SendNotification(message string) error {
	msg := tgbotapi.NewMessage(t.chatID, message)
	_, err := t.bot.Send(msg)
	return err
}
