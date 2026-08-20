param(
    [string]$TargetHost = '127.0.0.1',
    [int]$MySQL57Port = 3307,
    [Parameter(Mandatory = $true)][string]$MySQL57Password,
    [int]$MySQL80Port = 3306,
    [Parameter(Mandatory = $true)][string]$MySQL80Password,
    [string]$TargetUser = 'root',
    [string]$GatewayUser = 'archery',
    [Parameter(Mandatory = $true)][string]$GatewayPassword,
    [string]$Config = '..\..\config\config.toml.default',
    [string]$Executable = '..\..\bin\windows-amd64\goinception-plus.exe',
    [string]$Python = 'python',
    [switch]$ConfirmDestructive
)
$ErrorActionPreference = 'Stop'
if (-not $ConfirmDestructive) {
    throw 'This suite drops/recreates gip_manual on both target instances. Re-run with -ConfirmDestructive.'
}
$common = @{
    TargetHost = $TargetHost; TargetUser = $TargetUser
    GatewayUser = $GatewayUser; GatewayPassword = $GatewayPassword
    Config = $Config; Executable = $Executable; Python = $Python
    ConfirmDestructive = $true
}
& (Join-Path $PSScriptRoot 'run-phase1-7.ps1') @common -TargetPort $MySQL57Port -TargetPassword $MySQL57Password
& (Join-Path $PSScriptRoot 'run-rules.ps1') @common -TargetPort $MySQL57Port -TargetPassword $MySQL57Password
& (Join-Path $PSScriptRoot 'run-phase1-7.ps1') @common -TargetPort $MySQL80Port -TargetPassword $MySQL80Password
& (Join-Path $PSScriptRoot 'run-rules.ps1') @common -TargetPort $MySQL80Port -TargetPassword $MySQL80Password
Write-Host 'MySQL 5.7/8.0 phase and rule regression runs completed.' -ForegroundColor Green

