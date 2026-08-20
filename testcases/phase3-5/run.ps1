param(
    [string]$HostName = "127.0.0.1",
    [int]$Port = 3306,
    [Parameter(Mandatory = $true)][string]$User,
    [Parameter(Mandatory = $true)][string]$Password,
    [string]$Executable = "..\..\bin\windows-amd64\goinception-plus.exe",
    [string]$MySQL = "mysql",
    [switch]$SkipSetup,
    [switch]$SkipVerify
)

$ErrorActionPreference = "Stop"
$suiteDirectory = Split-Path -Parent $MyInvocation.MyCommand.Path
$executablePath = [System.IO.Path]::GetFullPath((Join-Path $suiteDirectory $Executable))
$resultDirectory = Join-Path $suiteDirectory "results"
$requestDirectory = Join-Path $suiteDirectory "requests"

if (-not (Test-Path -LiteralPath $executablePath -PathType Leaf)) {
    throw "goinception-plus executable not found: $executablePath"
}
New-Item -ItemType Directory -Force -Path $resultDirectory | Out-Null

function Invoke-MySQLFile([string]$FileName) {
    # Do not pipe a PowerShell string to mysql.exe. Windows PowerShell may
    # prepend a BOM while encoding native-process stdin, which MySQL then
    # treats as part of the first SQL keyword. Let mysql read the UTF-8
    # no-BOM file directly instead.
    $sqlPath = (Resolve-Path -LiteralPath (Join-Path $suiteDirectory $FileName)).Path.Replace('\', '/')
    & $MySQL "--host=$HostName" "--port=$Port" "--user=$User" "--password=$Password" `
        --default-character-set=utf8mb4 "--execute=source $sqlPath"
    if ($LASTEXITCODE -ne 0) { throw "mysql failed for $FileName" }
}

if (-not $SkipSetup) { Invoke-MySQLFile "setup.sql" }

$manifest = Get-Content -LiteralPath (Join-Path $suiteDirectory "manifest.json") -Raw | ConvertFrom-Json
$passed = 0
$failed = 0
foreach ($case in $manifest) {
    $template = Get-Content -LiteralPath (Join-Path $requestDirectory $case.file) -Raw
    $request = $template.Replace("{{HOST}}", $HostName).
        Replace("{{PORT}}", [string]$Port).
        Replace("{{USER}}", $User).
        Replace("{{PASSWORD}}", $Password)
    $startInfo = [System.Diagnostics.ProcessStartInfo]::new()
    $startInfo.FileName = $executablePath
    $startInfo.Arguments = "audit"
    $startInfo.UseShellExecute = $false
    $startInfo.RedirectStandardInput = $true
    $startInfo.RedirectStandardOutput = $true
    $startInfo.RedirectStandardError = $true
    $startInfo.CreateNoWindow = $true
    $process = [System.Diagnostics.Process]::new()
    $process.StartInfo = $startInfo
    [void]$process.Start()
    $process.StandardInput.Write($request)
    $process.StandardInput.Close()
    $outputText = $process.StandardOutput.ReadToEnd()
    $errorText = $process.StandardError.ReadToEnd()
    $process.WaitForExit()
    if ($process.ExitCode -ne 0) {
        Write-Host "[FAIL] $($case.file): process exit $($process.ExitCode): $errorText" -ForegroundColor Red
        $failed++
        continue
    }
    [System.IO.File]::WriteAllText(
        (Join-Path $resultDirectory ($case.file + ".json")),
        $outputText,
        [System.Text.UTF8Encoding]::new($false)
    )
    try { $records = $outputText | ConvertFrom-Json } catch {
        Write-Host "[FAIL] $($case.file): invalid JSON" -ForegroundColor Red
        $failed++
        continue
    }
    $statusMatched = @($records | Where-Object { $_.StageStatus -eq $case.status }).Count -gt 0
    $ruleMatched = [string]::IsNullOrEmpty($case.rule) -or
        @($records.Issues | Where-Object { $_.RuleID -eq $case.rule }).Count -gt 0
    if ($statusMatched -and $ruleMatched) {
        Write-Host "[PASS] $($case.file) => $($case.status) $($case.rule)" -ForegroundColor Green
        $passed++
    } else {
        Write-Host "[FAIL] $($case.file): expected status=$($case.status), rule=$($case.rule)" -ForegroundColor Red
        $failed++
    }
}

if (-not $SkipVerify) { Invoke-MySQLFile "verify.sql" }
Write-Host "Completed: passed=$passed failed=$failed; JSON results: $resultDirectory"
if ($failed -gt 0) { exit 1 }
