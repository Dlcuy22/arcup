# arcup

(short for archive backup) is a portable CLI backup tool that archives files using `tar` with configurable compression and uploads them via `rclone` to any supported backend.

## Install

```bash
go install github.com/dlcuy22/arcup@latest
```

## Requirements

The following binaries must be available on `$PATH`:

| Binary   | Required for       |
| -------- | ------------------ |
| `tar`    | All archive modes  |
| `zstd`   | `--algo zstd`      |
| `gzip`   | `--algo gz`        |
| `zip`    | `--algo zip`       |
| `rclone` | All upload/restore |

## Quick Start

**One-shot backup:**

```bash
arcup run -s /home/user/docs -s /etc -a zstd -r s3:bucket/backups
```

**Scheduled backups (watch mode):**

```bash
# Every 6 hours
arcup run -s /home/user/docs -r s3:bucket/backups -w -i 6h

# Every day at midnight
arcup run -s /home/user/docs -r s3:bucket/backups -w -i "@daily"

# Custom cron: 2 AM on weekdays
arcup run -s /home/user/docs -r s3:bucket/backups -w -i "0 2 * * 1-5"
```

Press `Ctrl+C` to stop watch mode gracefully.

**With retention policy:**

```bash
arcup run -s /home/user/docs -r s3:bucket/backups --keep-days 7 --keep-last 5
```

**With a config file:**

```bash
arcup run -c ~/.arcup.yaml
```

**Restore from remote:**

```bash
arcup restore -r s3:bucket/backups
```

## Commands

### `arcup run`

Archives the configured sources, uploads to the remote, and exits. Use `-w`/`--watch` with `-i`/`--interval` to repeat on a schedule.

```
arcup run [flags]

Flags:
  -s, --source strings      source file or directory (repeatable)
  -a, --algo string         compression algorithm: zstd, gz, zip (default "zstd")
  -A, --algo-args string    extra arguments for the compression tool
  -u, --upload-to string    upload backend (default "rclone")
  -o, --output-dir string   local staging directory (default "/tmp/arcup")
  -k, --keep int            keep N local copies, 0 = delete after upload
  -N, --name string         archive name prefix (default: hostname)

  # Watch Mode
  -w, --watch               run on a schedule instead of one-shot
  -i, --interval string     schedule: cron expr or duration (e.g. "@daily", "6h")

  # Retention Policy
  -d, --keep-days int       delete remote backups older than N days (0 = disabled)
  -K, --keep-last int       always keep at least N most recent (0 = disabled)

  # Retry Mechanism
      --retry-attempts int  max attempts for archive/upload (default 3)
      --retry-delay string  delay between retries (default "1m")
```

### `arcup restore`

Lists available backups on the remote and restores a selected archive.

```
arcup restore [flags]

Flags:
  -t, --to string       local destination for extraction (default ".")
  -S, --select int      skip prompt, pick backup by index
  -l, --list            list only, don't download or extract
  -L, --limit int       number of backups to show (default 5)
  -v, --verify          checksum verify after download (default true)
```

### `arcup validate`

Pre-flight checks: verifies sources exist, required binaries are on `$PATH`, and the rclone remote is reachable.

```
arcup validate
```

### Global Flags

```
  -c, --config string   config file (default: ~/.arcup.yaml)
  -n, --dry-run         simulate without writing or uploading
  -r, --remote string   rclone remote path (e.g. s3:bucket/backups)
```

## Configuration

Copy `.arcup.yaml.example` to `~/.arcup.yaml` and customize:

```yaml
name: my-backup
output-dir: /tmp/arcup
keep: 3
dry-run: false

sources:
  - /home/user/documents
  - /home/user/projects

algo: zstd
algo-args: ""

upload-to: rclone
remote: "s3:my-bucket/backups"

# Watch mode (use with: arcup run -w)
watch: false
interval: "@daily" # or "6h", "30m", "@weekly"

# Retry settings (for archive creation and upload)
retry:
  max-attempts: 3
  delay: "1m"

# Retention (runs after each backup if set)
retention:
  keep-days: 7
  keep-last: 5
```

Flags always take priority over config file values.

## Archive Naming

```
{name}_{hostname}_{timestamp}.tar.{ext}
```

Example:

```
my-backup_devbox_2026-05-06T14-30-00.tar.zst
```

Each archive is accompanied by a sidecar `.meta.json` file containing sources, checksum, compression settings, and version info. This enables listing and verifying backups without downloading the full archive.

## Interval Syntax

The `-i`/`--interval` flag accepts two formats:

**Durations** runs immediately, then repeats:

| Value   | Meaning          |
| ------- | ---------------- |
| `30m`   | Every 30 minutes |
| `6h`    | Every 6 hours    |
| `2h30m` | Every 2.5 hours  |

**Cron expressions** waits until next scheduled time:

| Value           | Meaning                   |
| --------------- | ------------------------- |
| `@hourly`       | Top of every hour         |
| `@daily`        | Midnight every day        |
| `@weekly`       | Sunday at midnight        |
| `0 2 * * *`     | 2:00 AM every day         |
| `0 */4 * * *`   | Every 4 hours on the hour |
| `30 18 * * 1-5` | 6:30 PM weekdays          |

## Retention Policy

When you provide `--keep-days` and/or `--keep-last` to the `run` command, arcup will apply this retention policy after a successful backup upload:

- **`keep-days`** delete anything older than N days
- **`keep-last`** always keep at least N most recent, regardless of age

Protected entries (keep-last) always win over expired entries. This prevents accidental deletion of all backups if no new backups ran for a while.

Use `--dry-run` to preview what would be deleted before committing.

## License

MIT
