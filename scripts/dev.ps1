# Local development. One entry: API on the host, pages from source.
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

function Import-EnvFile([string]$path) {
    if (-not (Test-Path $path)) { return }
    Get-Content $path | ForEach-Object {
        $line = $_.Trim()
        if ($line -eq "" -or $line.StartsWith("#")) { return }
        $i = $line.IndexOf("=")
        if ($i -lt 1) { return }
        Set-Item -Path ("Env:" + $line.Substring(0, $i).Trim()) -Value $line.Substring($i + 1).Trim()
    }
}

function Stop-Listeners([int[]]$ports) {
    foreach ($port in $ports) {
        $pids = @()
        netstat -ano | ForEach-Object {
            if ($_ -match ":$port\s" -and $_ -match "LISTENING\s+(\d+)\s*$") {
                $pids += [int]$Matches[1]
            }
        }
        $pids | Select-Object -Unique | ForEach-Object {
            if ($_ -gt 0) { taskkill /PID $_ /T /F 2>$null | Out-Null }
        }
    }
}

Import-EnvFile (Join-Path $root ".env")
$env:NEXA_DEV = "1"

docker compose -f (Join-Path $root "docker-compose.yml") stop server | Out-Null
docker compose -f (Join-Path $root "docker-compose.yml") up -d postgres
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

foreach ($app in @("admin", "public")) {
    $dir = Join-Path $root "web\$app"
    if (-not (Test-Path (Join-Path $dir "node_modules"))) {
        npm install --prefix $dir
        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    }
}

Stop-Listeners @(8080, 8081, 5173, 5174)
Start-Sleep -Seconds 1

$go = Start-Process -FilePath "go" -ArgumentList @("run", "-tags", "dev", "./cmd/nexa") -WorkingDirectory $root -PassThru -NoNewWindow
$admin = Start-Process -FilePath "node" -ArgumentList @("node_modules/vite/bin/vite.js") -WorkingDirectory (Join-Path $root "web\admin") -PassThru -NoNewWindow
$public = Start-Process -FilePath "node" -ArgumentList @("node_modules/vite/bin/vite.js") -WorkingDirectory (Join-Path $root "web\public") -PassThru -NoNewWindow
$children = @($go, $admin, $public)

Write-Host ""
Write-Host "公开页  http://127.0.0.1:5173"
Write-Host "管理页  http://127.0.0.1:5174"
Write-Host "接口    http://127.0.0.1:8080  和  http://127.0.0.1:8081"
Write-Host ""

try {
    while ($true) {
        foreach ($p in $children) {
            if ($p.HasExited) { throw "进程 $($p.Id) 已退出，代码 $($p.ExitCode)" }
        }
        Start-Sleep -Seconds 2
    }
} finally {
    foreach ($p in $children) {
        if (-not $p.HasExited) { taskkill /PID $p.Id /T /F 2>$null | Out-Null }
    }
}
