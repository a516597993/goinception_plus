param(
    [string]$Version = "v8.5.3"
)

$ErrorActionPreference = "Stop"
$ProjectRoot = Split-Path -Parent $PSScriptRoot
$Target = Join-Path $ProjectRoot "third_party\\tidb"

if (Test-Path -LiteralPath $Target) {
    $Actual = git -C $Target describe --tags --exact-match 2>$null
    if ($LASTEXITCODE -ne 0 -or $Actual -ne $Version) {
        throw "TiDB source already exists at $Target but is not $Version."
    }
    Write-Host "TiDB $Version source is already present."
    exit 0
}

git clone --depth 1 --branch $Version --filter=blob:none https://github.com/pingcap/tidb.git $Target
if ($LASTEXITCODE -ne 0) {
    throw "Failed to clone TiDB $Version."
}

