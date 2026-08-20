param(
    [string]$TargetHost = '127.0.0.1',
    [int]$TargetPort = 3306,
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
. (Join-Path $PSScriptRoot 'common.ps1')
Invoke-GIPRegression -Suite rules @PSBoundParameters

