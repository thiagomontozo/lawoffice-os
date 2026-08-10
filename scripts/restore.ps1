param(
    [Parameter(Mandatory = $true)]
    [string]$Archive,
    [string]$DatabaseUrl = $env:DATABASE_URL,
    [string]$StoragePath = $(if ($env:STORAGE_PATH) { $env:STORAGE_PATH } else { "./data/storage" }),
    [switch]$Force
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

if (-not $Force) {
    throw "Restore replaces database and storage state. Re-run with -Force after verifying the target and backup."
}
if ([string]::IsNullOrWhiteSpace($DatabaseUrl)) {
    throw "DATABASE_URL or -DatabaseUrl is required."
}
if (-not (Get-Command pg_restore -ErrorAction SilentlyContinue)) {
    throw "pg_restore was not found in PATH. Install the PostgreSQL client tools first."
}
$archivePath = [IO.Path]::GetFullPath($Archive)
if (-not (Test-Path -LiteralPath $archivePath -PathType Leaf)) {
    throw "Backup archive not found: $archivePath"
}

$temporaryRoot = [IO.Path]::GetFullPath((Join-Path ([IO.Path]::GetTempPath()) "lawoffice-restore-$([guid]::NewGuid().ToString('N'))"))
New-Item -ItemType Directory -Path $temporaryRoot | Out-Null
try {
    Expand-Archive -LiteralPath $archivePath -DestinationPath $temporaryRoot
    $manifestPath = Join-Path $temporaryRoot "manifest.json"
    $databaseDump = Join-Path $temporaryRoot "database.dump"
    if (-not (Test-Path -LiteralPath $manifestPath) -or -not (Test-Path -LiteralPath $databaseDump)) {
        throw "Archive is missing manifest.json or database.dump."
    }
    $manifest = Get-Content -Raw -LiteralPath $manifestPath | ConvertFrom-Json
    if ($manifest.formatVersion -ne 1) {
        throw "Unsupported backup format version: $($manifest.formatVersion)"
    }
    foreach ($item in $manifest.checksums.PSObject.Properties) {
        $candidate = [IO.Path]::GetFullPath((Join-Path $temporaryRoot $item.Name))
        if (-not $candidate.StartsWith($temporaryRoot + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)) {
            throw "Manifest contains an unsafe path."
        }
        $actual = (Get-FileHash -Algorithm SHA256 -LiteralPath $candidate).Hash.ToLowerInvariant()
        if ($actual -ne [string]$item.Value) {
            throw "Checksum mismatch for $($item.Name)."
        }
    }

    & pg_restore "--dbname=$DatabaseUrl" --clean --if-exists --no-owner --exit-on-error $databaseDump
    if ($LASTEXITCODE -ne 0) {
        throw "pg_restore failed with exit code $LASTEXITCODE. Storage was not changed."
    }

    $storageRoot = [IO.Path]::GetFullPath($StoragePath)
    $storageSource = Join-Path $temporaryRoot "storage"
    $parent = Split-Path -Parent $storageRoot
    New-Item -ItemType Directory -Path $parent -Force | Out-Null
    if (Test-Path -LiteralPath $storageRoot) {
        $previous = "$storageRoot.pre-restore-$((Get-Date).ToUniversalTime().ToString('yyyyMMddTHHmmssZ'))"
        Move-Item -LiteralPath $storageRoot -Destination $previous
        Write-Warning "Previous storage was preserved at $previous"
    }
    New-Item -ItemType Directory -Path $storageRoot | Out-Null
    if (Test-Path -LiteralPath $storageSource) {
        Get-ChildItem -LiteralPath $storageSource -Force | Copy-Item -Destination $storageRoot -Recurse -Force
    }
    Write-Output "Restore completed from $archivePath"
}
finally {
    $systemTemporaryRoot = [IO.Path]::GetFullPath([IO.Path]::GetTempPath())
    if ($temporaryRoot.StartsWith($systemTemporaryRoot, [StringComparison]::OrdinalIgnoreCase) -and (Test-Path -LiteralPath $temporaryRoot)) {
        Remove-Item -LiteralPath $temporaryRoot -Recurse -Force
    }
}
