param(
    [string]$HostName = "127.0.0.1",
    [int]$Port = 4000,
    [string]$User = "archery",
    [string]$Password = "123456",
    [string]$TargetHost = "127.0.0.1",
    [int]$TargetPort = 3306,
    [string]$TargetUser = "root",
    [string]$TargetPassword = "123456",
    [string]$Database = "test9"
)

$ErrorActionPreference = "Stop"
$sizes = @(100, 1000, 10000)
foreach ($size in $sizes) {
    $path = Join-Path $env:TEMP "gip-capacity-$size.sql"
    $header = "/*--host=$TargetHost;--port=$TargetPort;--user=$TargetUser;--password=$TargetPassword;--check=1;--execute=0;--backup=0;--trace-id=capacity:$size;*/`ninception_magic_start;`nUSE ``$Database``;`n"
    $body = [System.Text.StringBuilder]::new($header)
    # USE also counts as one statement at the request boundary.
    for ($i = 1; $i -lt $size; $i++) {
        [void]$body.Append("SELECT $i;`n")
    }
    [void]$body.Append("inception_magic_commit;`n")
    [System.IO.File]::WriteAllText($path, $body.ToString(), [System.Text.UTF8Encoding]::new($false))
    try {
        $elapsed = Measure-Command {
            Get-Content $path -Raw | mysql --host=$HostName --port=$Port --user=$User --password=$Password --batch --raw | Out-Null
        }
        [pscustomobject]@{ Statements = $size; Milliseconds = [math]::Round($elapsed.TotalMilliseconds, 2); RequestBytes = (Get-Item $path).Length }
    }
    finally { Remove-Item -LiteralPath $path -Force -ErrorAction SilentlyContinue }
}
