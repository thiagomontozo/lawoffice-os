param(
    [string]$DatabaseUrl = $env:DATABASE_URL,
    [string]$StoragePath = $(if ($env:STORAGE_PATH) { $env:STORAGE_PATH } else { "./data/storage" }),
    [string]$BackupDirectory = "./backups"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

if ([string]::IsNullOrWhiteSpace($DatabaseUrl)) {
    throw "DATABASE_URL or -DatabaseUrl is required."
}
if (-not (Get-Command pg_dump -ErrorAction SilentlyContinue)) {
    throw "pg_dump was not found in PATH. Install the PostgreSQL client tools first."
}

$backupRoot = [IO.Path]::GetFullPath($BackupDirectory)
$storageRoot = [IO.Path]::GetFullPath($StoragePath)
New-Item -ItemType Directory -Path $backupRoot -Force | Out-Null
$stamp = (Get-Date).ToUniversalTime().ToString("yyyyMMddTHHmmssZ")
$staging = Join-Path $backupRoot ".lawoffice-backup-$stamp-$([guid]::NewGuid().ToString('N'))"
$archive = Join-Path $backupRoot "lawoffice-backup-$stamp.zip"
New-Item -ItemType Directory -Path $staging | Out-Null

try {
    $databaseDump = Join-Path $staging "database.dump"
    & pg_dump "--dbname=$DatabaseUrl" --format=custom --no-owner "--file=$databaseDump"
    if ($LASTEXITCODE -ne 0) {
        throw "pg_dump failed with exit code $LASTEXITCODE."
    }

    $storageBackup = Join-Path $staging "storage"
    New-Item -ItemType Directory -Path $storageBackup | Out-Null
    if (Test-Path -LiteralPath $storageRoot -PathType Container) {
        Get-ChildItem -LiteralPath $storageRoot -Force | Copy-Item -Destination $storageBackup -Recurse -Force
    }

    $checksums = @{}
    Get-ChildItem -LiteralPath $staging -File -Recurse | ForEach-Object {
        $relative = [IO.Path]::GetRelativePath($staging, $_.FullName).Replace("\", "/")
        $checksums[$relative] = (Get-FileHash -Algorithm SHA256 -LiteralPath $_.FullName).Hash.ToLowerInvariant()
    }
    $manifest = [ordered]@{
        formatVersion = 1
        createdAt = (Get-Date).ToUniversalTime().ToString("o")
        databaseFormat = "postgresql-custom"
        storageIncluded = $true
        checksums = $checksums
    }
    $manifest | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath (Join-Path $staging "manifest.json") -Encoding utf8
    Compress-Archive -Path (Join-Path $staging "*") -DestinationPath $archive -CompressionLevel Optimal
    Write-Output $archive
}
finally {
    $resolvedStaging = [IO.Path]::GetFullPath($staging)
    if ($resolvedStaging.StartsWith($backupRoot + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase) -and (Test-Path -LiteralPath $resolvedStaging)) {
        Remove-Item -LiteralPath $resolvedStaging -Recurse -Force
    }
}
