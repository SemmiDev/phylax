# Advanced Multi-Target Database Backup System

Production-ready automated backup solution with support for multiple databases and multiple upload destinations.

## Features

### Database Support
- ✅ **MySQL** - Full mysqldump integration
- ✅ **Multiple Connections** - Backup multiple databases with different schedules
- ✅ **Flexible Scheduling** - Individual cron schedule per database

### Upload Destinations
- ✅ **Local Storage** - Always enabled
- ✅ **Google Drive** - Service account integration
- ✅ **AWS S3** - Compatible with S3-compatible storage
- ✅ **Telegram** - Notifications and file uploads
- ✅ **Parallel Uploads** - All destinations upload simultaneously
- ✅ **Flexible Configuration** - Enable/disable any combination

### Core Features
- ✅ **Compression** - Gzip compression (70-90% reduction)
- ✅ **Retention Policy** - Automatic cleanup across all destinations
- ✅ **Structured Logging** - JSON + console with rotation
- ✅ **Systemd Integration** - Run as daemon
- ✅ **Graceful Shutdown** - Proper cleanup
- ✅ **Error Resilience** - Failed uploads don't stop others

## 📦 Installation

### Prerequisites

```bash
# Database clients
sudo apt install mysql-client postgresql-client mongodb-clients

# Or on RHEL/CentOS
sudo yum install mysql postgresql mongodb-org-tools
```

### Build & Install

```bash
# Clone repository
git clone github.com/semmidev/phylax
cd phylax

# Install dependencies
make deps

# Build
make build

# Install as service
sudo make install
```

## ⚙️ Configuration

### Example: Multiple Databases + Multiple Targets

```yaml
databases:
  # Production MySQL
  - name: "prod-mysql"
    type: "mysql"
    host: "db1.example.com"
    port: 3306
    username: "backup"
    password: "secret"
    database: "production"
    enabled: true
    schedule: "0 0 2 * * *"  # 2 AM daily

  # Analytics PostgreSQL
  - name: "analytics-pg"
    type: "postgresql"
    host: "db2.example.com"
    port: 5432
    username: "postgres"
    password: "secret"
    database: "analytics"
    enabled: true
    schedule: "0 0 3 * * *"  # 3 AM daily

  # Logs MongoDB
  - name: "logs-mongo"
    type: "mongodb"
    host: "db3.example.com"
    port: 27017
    username: "admin"
    password: "secret"
    database: "logs"
    enabled: true
    schedule: "0 0 4 * * *"  # 4 AM daily

backup:
  local_path: "/var/backups/databases"
  retention_days: 7
  compress: true

  upload_targets:
    # Always keep local copy
    - type: "local"
      enabled: true

    # Upload to Google Drive
    - type: "gdrive"
      enabled: true
      credentials_file: "/etc/phylax/gdrive.json"
      folder_id: "1a2b3c4d5e6f"

    # Upload to AWS S3
    - type: "s3"
      enabled: true
      region: "us-east-1"
      bucket: "company-backups"
      access_key: "AKIAIOSFODNN7EXAMPLE"
      secret_key: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
      prefix: "database-backups/"

    # Send notification to Telegram
    - type: "telegram"
      enabled: true
      bot_token: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"
      chat_id: "-1001234567890"
      send_file: false  # Only notify (for large files)
      notify_only: true
```

## 🚀 Usage

### Service Management

```bash
# Start service
sudo systemctl start phylax

# Enable on boot
sudo systemctl enable phylax

# Check status
sudo systemctl status phylax

# View logs
sudo journalctl -u phylax -f

# Or use Makefile
make status
make logs
```

### Manual Backup

```bash
# Run once
phylax -config /etc/phylax/config.yaml

# Check version
phylax -version
```

## 📊 How It Works

```
┌─────────────────────────────────────────────────────────────┐
│                    Scheduled Triggers                        │
│  2 AM: prod-mysql  |  3 AM: analytics-pg  |  4 AM: logs-mongo│
└──────────────┬──────────────────┬──────────────────┬─────────┘
               │                  │                  │
               ▼                  ▼                  ▼
        ┌──────────┐       ┌──────────┐      ┌──────────┐
        │  Backup  │       │  Backup  │      │  Backup  │
        │  MySQL   │       │PostgreSQL│      │ MongoDB  │
        └─────┬────┘       └─────┬────┘      └─────┬────┘
              │                  │                  │
              ▼                  ▼                  ▼
         [Compress]         [Compress]         [Compress]
              │                  │                  │
              └──────────────────┴──────────────────┘
                                 │
                     ┌───────────┴───────────┐
                     │   Parallel Uploads     │
                     └───────────┬───────────┘
                                 │
        ┌────────────────────────┼────────────────────────┐
        │                        │                        │
        ▼                        ▼                        ▼
  ┌──────────┐            ┌──────────┐            ┌──────────┐
  │  Local   │            │  GDrive  │            │   S3     │
  │ Storage  │            │          │            │          │
  └──────────┘            └──────────┘            └──────────┘
        │                        │                        │
        ▼                        ▼                        ▼
   [Success]               [Success]               [Success]
                                 │
                                 ▼
                          ┌──────────┐
                          │ Telegram │
                          │  Notify  │
                          └──────────┘
```

## 🔧 Advanced Configuration

### Google Drive Setup

1. Create service account at [Google Cloud Console](https://console.cloud.google.com)
2. Enable Google Drive API
3. Download credentials JSON
4. Share folder with service account email
5. Get folder ID from URL: `https://drive.google.com/drive/folders/FOLDER_ID_HERE`

### AWS S3 Setup

```bash
# Create IAM user with S3 permissions
# Policy example:
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": ["s3:PutObject", "s3:GetObject", "s3:ListBucket", "s3:DeleteObject"],
    "Resource": ["arn:aws:s3:::your-bucket/*", "arn:aws:s3:::your-bucket"]
  }]
}
```

### Telegram Bot Setup

1. Create bot via [@BotFather](https://t.me/botfather)
2. Get bot token
3. Add bot to your group/channel
4. Get chat ID:
```bash
# Send message to bot first, then:
curl https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getUpdates
```

### Cron Schedule Examples

```yaml
# Format: second minute hour day month weekday

"0 0 2 * * *"      # Every day at 2:00 AM
"0 */6 * * * *"    # Every 6 hours
"0 0 0 * * 0"      # Every Sunday at midnight
"0 30 1 * * 1-5"   # Weekdays at 1:30 AM
"0 0 */4 * * *"    # Every 4 hours
"0 0 2 1 * *"      # First day of month at 2 AM
```

### Multiple Databases Example

```yaml
databases:
  # Main application database
  - name: "app-primary"
    type: "mysql"
    host: "primary.db.local"
    port: 3306
    username: "backup"
    password: "pass1"
    database: "app"
    enabled: true
    schedule: "0 0 2 * * *"

  # Read replica (for analytics)
  - name: "app-replica"
    type: "mysql"
    host: "replica.db.local"
    port: 3306
    username: "backup"
    password: "pass2"
    database: "app"
    enabled: true
    schedule: "0 0 6 * * *"  # Different time

  # Separate service database
  - name: "auth-service"
    type: "postgresql"
    host: "auth.db.local"
    port: 5432
    username: "postgres"
    password: "pass3"
    database: "auth"
    enabled: true
    schedule: "0 0 3 * * *"

  # Logs database (less frequent)
  - name: "logs"
    type: "mongodb"
    host: "logs.db.local"
    port: 27017
    username: "admin"
    password: "pass4"
    database: "logs"
    enabled: true
    schedule: "0 0 0 * * 0"  # Weekly on Sunday
```

## 🔐 Security Best Practices

### 1. Protect Configuration

```bash
# Secure config file
sudo chmod 600 /etc/phylax/config.yaml
sudo chown root:root /etc/phylax/config.yaml

# Store credentials separately (optional)
sudo chmod 600 /etc/phylax/gdrive.json
sudo chmod 600 /etc/phylax/.aws-credentials
```

### 2. Database User Permissions

```sql
-- MySQL: Create dedicated backup user
CREATE USER 'backup'@'%' IDENTIFIED BY 'secure_password';
GRANT SELECT, LOCK TABLES, SHOW VIEW, EVENT, TRIGGER ON *.* TO 'backup'@'%';
FLUSH PRIVILEGES;

-- PostgreSQL: Create backup user
CREATE USER backup WITH PASSWORD 'secure_password';
GRANT CONNECT ON DATABASE mydb TO backup;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO backup;
```

### 3. Network Security

```bash
# Allow backup server IP only
# MySQL
mysql> CREATE USER 'backup'@'10.0.1.100' IDENTIFIED BY 'password';

# PostgreSQL pg_hba.conf
host    all    backup    10.0.1.100/32    md5
```

### 4. Encrypt Sensitive Data

```bash
# Encrypt backups before upload (optional enhancement)
# Add to your workflow:
gpg --symmetric --cipher-algo AES256 backup.sql.gz
```

## 📈 Monitoring & Alerts

### Log Analysis

```bash
# Check recent backups
sudo journalctl -u phylax --since "24 hours ago" | grep "completed successfully"

# Check failures
sudo journalctl -u phylax --since "7 days ago" | grep -i error

# Monitor backup sizes
ls -lh /var/backups/databases/

# Check log file
tail -f /var/log/phylax/backup.log
```

### Telegram Alerts

The system automatically sends Telegram notifications for:
- ✅ Successful backups
- ❌ Failed backups
- 📊 Backup statistics (size, duration)

### Integration with Monitoring Tools

```bash
# Prometheus metrics endpoint (optional enhancement)
# Add metrics exporter for:
# - backup_success_total
# - backup_failure_total
# - backup_duration_seconds
# - backup_size_bytes
```

## 🐛 Troubleshooting

### Database Connection Issues

```bash
# Test MySQL connection
mysql -h hostname -P 3306 -u username -p

# Test PostgreSQL connection
PGPASSWORD=password psql -h hostname -p 5432 -U username -d database

# Test MongoDB connection
mongosh "mongodb://username:password@hostname:27017/database"
```

### Permission Errors

```bash
# Check service user
sudo systemctl status phylax

# Fix backup directory permissions
sudo chown -R root:root /var/backups/databases
sudo chmod 755 /var/backups/databases

# Fix log directory
sudo mkdir -p /var/log/phylax
sudo chmod 755 /var/log/phylax
```

### Upload Failures

```bash
# Check Google Drive credentials
cat /etc/phylax/gdrive.json

# Test AWS S3 access
aws s3 ls s3://your-bucket --profile backup

# Test Telegram bot
curl https://api.telegram.org/bot<TOKEN>/getMe
```

### Service Won't Start

```bash
# Check syntax
phylax -config /etc/phylax/config.yaml

# View detailed logs
sudo journalctl -u phylax -n 100 --no-pager

# Validate config
sudo systemctl status phylax
```

## 📊 Performance Optimization

### Large Databases

```yaml
# For databases > 100GB, consider:
databases:
  - name: "large-db"
    type: "mysql"
    # Use dedicated backup server
    host: "backup-replica.local"
    # Schedule during low-traffic hours
    schedule: "0 0 3 * * *"
```

### Network Bandwidth

```bash
# Compress before upload (already enabled)
compress: true

# For S3, use multipart uploads (automatic for large files)
# For very large files, consider:
# - Streaming uploads
# - Resume capability
# - Bandwidth throttling
```

### Disk Space Management

```yaml
# Aggressive retention for large backups
backup:
  retention_days: 3  # Keep only 3 days

# Or use different retention per destination
# Enhancement: Add per-target retention policies
```

## 🔄 Backup Verification

### Automated Testing

```bash
# Add to cron or systemd timer
# /usr/local/bin/verify-backup.sh
#!/bin/bash
BACKUP_FILE=$(ls -t /var/backups/databases/*.gz | head -1)
gunzip -t "$BACKUP_FILE" && echo "✓ Backup integrity OK" || echo "✗ Backup corrupted"
```

### Restore Testing

```bash
# MySQL restore test
gunzip < backup.sql.gz | mysql -h localhost -u root -p test_restore

# PostgreSQL restore test
pg_restore -h localhost -U postgres -d test_restore backup.dump

# MongoDB restore test
mongorestore --archive=backup.archive --gzip --db test_restore
```

## 🤝 Contributing

Contributions welcome! Please ensure:

1. Tests pass: `make test`
2. Code formatted: `go fmt ./...`
3. Linting clean: `golangci-lint run`
4. Documentation updated

## 📄 License

MIT License - see LICENSE file

## 🆘 Support

- **Issues**: [GitHub Issues](https://github.com/yourusername/phylax/issues)
- **Discussions**: [GitHub Discussions](https://github.com/yourusername/phylax/discussions)
- **Email**: support@example.com

## 🎯 Roadmap

- [ ] PostgreSQL incremental backups
- [ ] MySQL binary log backups
- [ ] Backup encryption
- [ ] Restore command
- [ ] Web UI dashboard
- [ ] Metrics exporter (Prometheus)
- [ ] Email notifications
- [ ] Slack integration
- [ ] Backup validation
- [ ] Multi-region S3 replication
- [ ] Azure Blob Storage support
- [ ] Backblaze B2 support
- [ ] Webhook notifications

## 🙏 Acknowledgments

Built with:
- [robfig/cron](https://github.com/robfig/cron) - Cron scheduler
- [spf13/viper](https://github.com/spf13/viper) - Configuration
- [uber-go/zap](https://github.com/uber-go/zap) - Logging
- [aws-sdk-go](https://github.com/aws/aws-sdk-go) - AWS integration
- [telegram-bot-api](https://github.com/go-telegram-bot-api/telegram-bot-api) - Telegram
- [google-api-go-client](https://github.com/googleapis/google-api-go-client) - Google Drive

---

**Made with ❤️ for reliable database backups**
