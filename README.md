# arcup

A portable CLI backup tool that archives files using `tar` with configurable compression and uploads them via `rclone` to any supported backend.

## Install

```bash
go install github.com/dlcuy22/arcup@latest
```

## Requirements

The following binaries must be available on `$PATH`:

| Binary   | Required for         |
|----------|----------------------|
| `tar`    | All archive modes    |
| `zstd`   | `--algo zstd`        |
| `gzip`   | `--algo gz`          |
| `zip`    | `--algo zip`         |
| `rclone` | All upload/restore   |

## Quick Start

**One-shot backup:**

```bash
arcup run -s /home/user/docs -s /etc -a zstd -r s3:bucket/backups
```

**With a config file:**

```bash
arcup run -c ~/.arcup.yaml
```

**Scheduled backups (daemon mode):**

```bash
arcup watch -c ~/.arcup.yaml
```

**Restore from remote:**

```bash
arcup restore -r s3:bucket/backups
```

## Commands

### `arcup run`

Performs a single backup cycle: archive sources, generate a metadata sidecar, upload both to the remote, and clean up local files.

```
arcup run [flags]

Flags:
  -s, --source strings      source file or directory (repeatable)
  -a, --algo string         compression algorithm: zstd, gz, zip (default "zstd")
  -A, --algo-args string    extra arguments for the compression tool
  -u, --upload-to string    upload backend (default "rclone")
  -r, --remote string       rclone remote path (e.g. s3:bucket/backups)
  -o, --output-dir string   local staging directory (default "/tmp/arcup")
  -k, --keep int            keep N local copies, 0 = delete after upload
  -N, --name string         archive name prefix (default: hostname)
```

### `arcup watch`

Runs backup cycles on a schedule. Blocks until interrupted.

```
arcup watch [flags]

Flags:
  -i, --interval string   cron expr or duration (e.g. "@daily", "6h")
```

### `arcup restore`

Lists available backups on the remote and restores a selected archive.

```
arcup restore [flags]

Flags:
  -r, --remote string   rclone remote path to list from
  -t, --to string       local destination for extraction (default ".")
  -S, --select int      skip prompt, pick backup by index
  -l, --list            list only, don't download or extract
  -L, --limit int       number of backups to show (default 5)
  -v, --verify          checksum verify after download (default true)
```

### `arcup retention`

Applies retention policy to remote backups: deletes expired archives while keeping a minimum count.

```
arcup retention [flags]

Flags:
  -r, --remote string   rclone remote path
  -d, --keep-days int   delete backups older than N days (default 7)
  -K, --keep-last int   always keep at least N most recent (default 5)
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

interval: "@daily"

retention:
  keep-days: 7
  keep-last: 5
  interval: "@daily"
  after-backup: true
  dry-run: false
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

## Retention Policy

Two rules work together:

- **`keep-days`** — delete anything older than N days
- **`keep-last`** — always keep at least N most recent, regardless of age

Protected entries (keep-last) always win over expired entries. This prevents accidental deletion of all backups if no new backups ran for a while.

Use `--dry-run` to preview what would be deleted before committing.

## License

MIT
