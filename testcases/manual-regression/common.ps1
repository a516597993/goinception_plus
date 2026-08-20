Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Invoke-GIPRegression {
    param(
        [Parameter(Mandatory = $true)][ValidateSet('phases','rules')][string]$Suite,
        [string]$TargetHost = '127.0.0.1',
        [Parameter(Mandatory = $true)][int]$TargetPort,
        [string]$TargetUser = 'root',
        [Parameter(Mandatory = $true)][string]$TargetPassword,
        [string]$GatewayHost = '127.0.0.1',
        [int]$GatewayPort = 4400,
        [string]$GatewayUser = 'archery',
        [Parameter(Mandatory = $true)][string]$GatewayPassword,
        [string]$Config = '..\..\config\config.toml.default',
        [string]$Executable = '..\..\bin\windows-amd64\goinception-plus.exe',
        [string]$Python = 'python',
        [switch]$UseRunningService,
        [switch]$SkipSetup,
        [switch]$ConfirmDestructive
    )
    if (-not $SkipSetup -and -not $ConfirmDestructive) {
        throw 'setup.sql drops/recreates gip_manual. Re-run with -ConfirmDestructive after verifying this is a test instance.'
    }
    $configPath = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot $Config))
    $exePath = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot $Executable))
    $runnerPath = Join-Path $PSScriptRoot 'regression_runner.py'
    if (-not (Test-Path -LiteralPath $runnerPath -PathType Leaf)) { throw "runner not found: $runnerPath" }
    & $Python -c 'import pymysql' 2>$null
    if ($LASTEXITCODE -ne 0) { throw "PyMySQL missing. Install with: $Python -m pip install PyMySQL" }

    $ownedProcess = $null
    $tempRoot = $null
    try {
        if (-not $UseRunningService) {
            if (-not (Test-Path -LiteralPath $configPath -PathType Leaf)) { throw "config not found: $configPath" }
            if (-not (Test-Path -LiteralPath $exePath -PathType Leaf)) { throw "executable not found: $exePath" }
            $listener = Get-NetTCPConnection -State Listen -LocalPort $GatewayPort -ErrorAction SilentlyContinue | Select-Object -First 1
            if ($listener) { throw "isolated gateway port $GatewayPort is already occupied by PID $($listener.OwningProcess)" }
            $tempRoot = Join-Path ([IO.Path]::GetTempPath()) ("gip-regression-" + [guid]::NewGuid().ToString('N'))
            New-Item -ItemType Directory -Path $tempRoot | Out-Null
            $tempConfig = Join-Path $tempRoot 'config.toml'
            $text = [IO.File]::ReadAllText($configPath)
            $serverStart = $text.IndexOf('[server]')
            if ($serverStart -lt 0) { throw 'config has no [server] section' }
            $nextSection = $text.IndexOf("`n[", $serverStart + 8)
            if ($nextSection -lt 0) { $nextSection = $text.Length }
            $serverSection = $text.Substring($serverStart, $nextSection - $serverStart)
            if ($serverSection -notmatch '(?m)^\s*port\s*=\s*\d+\s*$') { throw '[server].port not found' }
            $newServer = [regex]::Replace($serverSection, '(?m)^(\s*port\s*=\s*)\d+(\s*)$', "`${1}$GatewayPort`${2}", 1)
            $text = $text.Substring(0, $serverStart) + $newServer + $text.Substring($nextSection)
            $logStart = $text.IndexOf('[log]')
            if ($logStart -ge 0) {
                $logEnd = $text.IndexOf("`n[", $logStart + 5)
                if ($logEnd -lt 0) { $logEnd = $text.Length }
                $logSection = $text.Substring($logStart, $logEnd - $logStart)
                $newLog = [regex]::Replace($logSection, '(?m)^(\s*format\s*=\s*)"[^"]*"(\s*)$', '${1}"json"${2}', 1)
                $text = $text.Substring(0, $logStart) + $newLog + $text.Substring($logEnd)
            }
            [IO.File]::WriteAllText($tempConfig, $text, [Text.UTF8Encoding]::new($false))
            $stdout = Join-Path $tempRoot 'server.stdout.log'
            $stderr = Join-Path $tempRoot 'server.stderr.log'
            $ownedProcess = Start-Process -FilePath $exePath -ArgumentList @('-config', $tempConfig) `
                -PassThru -WindowStyle Hidden -RedirectStandardOutput $stdout -RedirectStandardError $stderr
            $ready = $false
            for ($i = 0; $i -lt 60; $i++) {
                if ($ownedProcess.HasExited) {
                    $serverError = if (Test-Path $stderr) { Get-Content $stderr -Raw } else { '' }
                    throw "isolated service exited early ($($ownedProcess.ExitCode)): $serverError"
                }
                if (Test-NetConnection -ComputerName $GatewayHost -Port $GatewayPort -InformationLevel Quiet -WarningAction SilentlyContinue) {
                    $ready = $true; break
                }
                Start-Sleep -Milliseconds 250
            }
            if (-not $ready) { throw "isolated service did not listen on $GatewayHost`:$GatewayPort within 15 seconds" }
        }

        $stamp = Get-Date -Format 'yyyyMMdd-HHmmss'
        $resultDir = Join-Path $PSScriptRoot ("results\$stamp-$Suite-mysql$TargetPort")
        $arguments = @(
            $runnerPath, '--suite', $Suite,
            '--gateway-host', $GatewayHost, '--gateway-port', [string]$GatewayPort,
            '--gateway-user', $GatewayUser, '--gateway-password', $GatewayPassword,
            '--target-host', $TargetHost, '--target-port', [string]$TargetPort,
            '--target-user', $TargetUser, '--target-password', $TargetPassword,
            '--output', $resultDir
        )
        if ($SkipSetup) { $arguments += '--skip-setup' }
        if ($ownedProcess) { $arguments += @('--server-log', $stderr) }
        & $Python @arguments
        $exitCode = $LASTEXITCODE
        Write-Host "Reports: $resultDir" -ForegroundColor Cyan
        if ($exitCode -ne 0) { throw "$Suite regression has FAIL/ERROR cases; inspect $resultDir\summary.md" }
    }
    finally {
        if ($ownedProcess -and -not $ownedProcess.HasExited) {
            Stop-Process -Id $ownedProcess.Id -Force -ErrorAction SilentlyContinue
            $ownedProcess.WaitForExit(5000) | Out-Null
        }
        if ($tempRoot -and (Test-Path -LiteralPath $tempRoot)) {
            Remove-Item -LiteralPath $tempRoot -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
}
