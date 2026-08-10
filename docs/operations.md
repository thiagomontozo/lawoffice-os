# Operations Runbook

This runbook covers the first operational baseline for a single LawOffice OS deployment. It does not replace an organization-specific disaster recovery plan.

## Service health

- `GET /healthz` proves that the HTTP process is alive.
- `GET /readyz` checks PostgreSQL and the configured object storage before declaring the instance ready for traffic.
- `/metrics` is registered only when `METRICS_TOKEN` is set. Scrapers must send `Authorization: Bearer <token>`.

The Prometheus text endpoint exports request totals, active requests, response status classes, cumulative request time, process uptime, goroutines and heap allocation. It intentionally excludes firm IDs, user IDs, request bodies, document names and other legal data.

Suggested first alerts:

| Signal | Starting condition | Action |
|---|---|---|
| readiness | `/readyz` fails for 2 minutes | inspect PostgreSQL and storage availability |
| server errors | 5xx responses exceed 2% for 5 minutes | inspect structured logs by request ID |
| latency | request duration growth is sustained | inspect database saturation and slow routes |
| memory | heap grows continuously across normal traffic cycles | capture runtime profile in a controlled environment |
| backup age | no successful archive in 24 hours | stop relying on the current recovery objective and repair backups |

When SMTP is enabled, also alert on terminal outbound jobs. The following operational query contains no decrypted recipient or message content:

```sql
SELECT status, count(*)
FROM outbound_jobs
GROUP BY status;
```

Investigate recurring `failed` rows using structured worker logs and job IDs. Do not expose or manually decrypt queue payloads during routine support.

Users opt into deadline and overdue-task e-mails under **Settings**. In-app notifications remain the source of truth. The scheduler uses a stable `notification:<uuid>` key before accepting a job and records `email_queued_at` afterward, preventing duplicate queue entries during normal multi-instance operation.

Thresholds must be tuned from real traffic. The values above are cautious starting points, not product guarantees.

## Backup

For `STORAGE_DRIVER=local`, `scripts/backup.ps1` creates one ZIP containing a PostgreSQL custom-format dump, object-storage files, a format manifest and SHA-256 checksums. S3 deployments must pair the database backup with provider-native bucket versioning, replication or an S3-native export; the local script refuses a missing storage directory rather than producing a silently incomplete archive.

```powershell
$env:DATABASE_URL = "postgres://..."
$env:STORAGE_PATH = "C:\lawoffice\storage"
./scripts/backup.ps1 -BackupDirectory "D:\lawoffice-backups"
```

Schedule backups outside the repository, encrypt them at rest, restrict access and copy them to a separate failure domain. Because PostgreSQL metadata and filesystem objects are captured sequentially, use a maintenance window or temporarily pause uploads when a strict point-in-time pair is required.

Recommended baseline:

- daily encrypted backup with at least 30 retained restore points;
- separate weekly copy with longer retention;
- quarterly restore exercise in an isolated environment;
- database-native point-in-time recovery for deployments with a stricter recovery point objective.

## Restore

Restoration is destructive to database state and therefore requires `-Force`. Existing storage is moved to a timestamped sibling directory instead of being deleted.

```powershell
$env:DATABASE_URL = "postgres://..."
$env:STORAGE_PATH = "C:\lawoffice\storage"
./scripts/restore.ps1 -Archive "D:\lawoffice-backups\lawoffice-backup-20260810T120000Z.zip" -Force
```

Restore into an isolated environment first whenever possible. After restoration:

1. confirm `manifest.json` checksum verification completed;
2. start one API instance and check `/readyz`;
3. authenticate with a designated recovery account;
4. sample Matters, permissions, document metadata and file downloads;
5. verify audit continuity and the newest expected records;
6. rotate session and metrics secrets if compromise was involved;
7. document the actual recovery point and recovery time.

Do not delete the `.pre-restore-*` storage directory until validation and retention policy permit it.

## Incident triage

1. Preserve logs, timestamps and request IDs; do not place document contents or credentials in incident chat.
2. Remove an unhealthy instance from traffic using readiness rather than deleting data.
3. Revoke affected sessions or disable users when credentials may be compromised.
4. Preserve database and object-storage evidence before corrective deletion.
5. Restore only after identifying the target recovery point and recording data-loss implications.

## Scaling boundaries

PostgreSQL coordinates migrations, persists and distributes realtime events, and locks the scheduler. The current rate limiter remains per API instance, so a multi-instance production rollout still needs an edge/shared limiter. SSE retains seven days and returns at most 500 events per reconnect; monitor clients that exceed that recovery window.
