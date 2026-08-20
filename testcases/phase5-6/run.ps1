param(
    [string]$TargetPassword = "123456",
    [string]$GatewayPassword = "archerypass",
    [int]$GatewayPort = 4000,
    [string]$ConfigPath = ""
)
$ErrorActionPreference = "Stop"
$root = [System.IO.Path]::GetFullPath((Join-Path (Split-Path -Parent $MyInvocation.MyCommand.Path) "..\.."))
$env:GOINCEPTION_PLUS_PASSWORD = $GatewayPassword
$env:GOINCEPTION_PLUS_BACKUP_PASSWORD = $TargetPassword
$env:GOINCEPTION_PLUS_TARGET_PASSWORD = $TargetPassword
$env:GOINCEPTION_PLUS_GATEWAY_PORT = [string]$GatewayPort
$stdout = Join-Path $env:TEMP "goinception-plus-phase56.out"
$stderr = Join-Path $env:TEMP "goinception-plus-phase56.err"
if ([string]::IsNullOrWhiteSpace($ConfigPath)) { $ConfigPath = Join-Path $root "config\config.integration.toml" }
$process = Start-Process -FilePath (Join-Path $root "bin\windows-amd64\goinception-plus.exe") `
    -ArgumentList "-config",$ConfigPath `
    -WorkingDirectory $root -WindowStyle Hidden -PassThru `
    -RedirectStandardOutput $stdout -RedirectStandardError $stderr
try {
    $ready = $false
    for ($i=0; $i -lt 50; $i++) {
        try { $tcp=[Net.Sockets.TcpClient]::new();$tcp.Connect("127.0.0.1",$GatewayPort);$tcp.Close();$ready=$true;break }
        catch { Start-Sleep -Milliseconds 200 }
    }
    if (-not $ready) { throw "$GatewayPort port did not become ready" }
    python (Join-Path $root "testcases\phase5-6\protocol_smoke.py")
    if ($LASTEXITCODE -ne 0) { throw "phase5-6 Python test failed" }
} finally {
    if (-not $process.HasExited) { Stop-Process -Id $process.Id; Wait-Process -Id $process.Id -ErrorAction SilentlyContinue }
    if (Test-Path $stderr) { Get-Content $stderr }
}
