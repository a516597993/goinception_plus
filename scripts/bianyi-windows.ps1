$env:CGO_ENABLED = "0"
$env:GOOS = "windows"
$env:GOARCH = "amd64"
$version = "0.7.0"
$commit = (git rev-parse --short HEAD 2>$null)
if (-not $commit) { $commit = "unknown" }
$built = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")

.\.tools\go\bin\go.exe build `
  -trimpath `
  -ldflags="-s -w -X main.version=$version -X main.commit=$commit -X main.buildTime=$built" `
  -o bin\windows-amd64\goinception-plus.exe `
  .\cmd\goinception-plus
