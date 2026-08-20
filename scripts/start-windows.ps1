param(
    [Alias("ConfigPath")]
    [string]$Config = ".\config\config.toml.default"
)
$ErrorActionPreference = "Stop"
if (-not (Test-Path -LiteralPath $Config -PathType Leaf)) { throw "Config not found: $Config" }
$portLine = Select-String -LiteralPath $Config -Pattern '^\s*port\s*=\s*(\d+)\s*$' | Select-Object -First 1
$port = if ($portLine -and $portLine.Matches.Count) { [int]$portLine.Matches[0].Groups[1].Value } else { 4000 }
$listener = Get-NetTCPConnection -State Listen -LocalPort $port -ErrorAction SilentlyContinue | Select-Object -First 1
if ($listener) {
    $process = Get-CimInstance Win32_Process -Filter "ProcessId=$($listener.OwningProcess)" -ErrorAction SilentlyContinue
    $details = if ($process) { "PID=$($process.ProcessId), executable=$($process.ExecutablePath), command=$($process.CommandLine)" } else { "PID=$($listener.OwningProcess)" }
    throw "Port $port is already listening ($details). Stop the existing process before starting a new configuration."
}
& .\bin\windows-amd64\goinception-plus.exe -config $Config
exit $LASTEXITCODE
