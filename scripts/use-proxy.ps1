# Enable Clash SOCKS5 proxy for this shell session (port 1080).
# Usage: . .\scripts\use-proxy.ps1

$Proxy = "socks5h://127.0.0.1:1080"

$env:ALL_PROXY = $Proxy
$env:HTTP_PROXY = $Proxy
$env:HTTPS_PROXY = $Proxy
$env:http_proxy = $Proxy
$env:https_proxy = $Proxy
$env:GOPROXY = "https://goproxy.cn,direct"

Write-Host "Proxy enabled: $Proxy"
Write-Host "GOPROXY=$env:GOPROXY"
