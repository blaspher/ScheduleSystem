param(
    [string]$EnvFile = ".env",
    [int]$ReadyTimeoutSec = 15
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Import-DotEnv {
    param([string]$Path)

    if (-not (Test-Path $Path)) {
        Write-Host "[WARN] Env file not found: $Path"
        return
    }

    Get-Content $Path | ForEach-Object {
        $line = $_.Trim()
        if ($line -eq "" -or $line.StartsWith("#")) {
            return
        }

        $parts = $line.Split("=", 2)
        if ($parts.Count -ne 2) {
            return
        }

        $name = $parts[0].Trim()
        $value = $parts[1].Trim()

        if (($value.StartsWith('"') -and $value.EndsWith('"')) -or ($value.StartsWith("'") -and $value.EndsWith("'"))) {
            if ($value.Length -ge 2) {
                $value = $value.Substring(1, $value.Length - 2)
            }
        }

        [System.Environment]::SetEnvironmentVariable($name, $value)
    }
}

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

Import-DotEnv -Path (Join-Path $repoRoot $EnvFile)

$goCmd = Get-Command go -ErrorAction SilentlyContinue
if (-not $goCmd) {
    throw "go command not found in PATH"
}

$port = if ($env:SERVER_PORT) { $env:SERVER_PORT } else { "8080" }
$baseUrl = "http://127.0.0.1:$port"

Write-Host "[INFO] Repo root: $repoRoot"
Write-Host "[INFO] Base URL: $baseUrl"
Write-Host "[INFO] Health URL: $baseUrl/healthz"
Write-Host "[INFO] API URL: $baseUrl/api/v1"
Write-Host "[INFO] Starting backend..."

$proc = Start-Process -FilePath "go" -ArgumentList @("run", "./cmd/server/main.go") -WorkingDirectory $repoRoot -NoNewWindow -PassThru

$ready = $false
for ($i = 0; $i -lt $ReadyTimeoutSec; $i++) {
    if ($proc.HasExited) {
        break
    }

    Start-Sleep -Seconds 1
    try {
        $null = Invoke-RestMethod -Uri "$baseUrl/healthz" -Method GET -TimeoutSec 2
        $ready = $true
        break
    }
    catch {
        # keep polling until timeout
    }
}

if (-not $ready) {
    Write-Host "[ERROR] Server did not become ready within $ReadyTimeoutSec seconds"
    if (-not $proc.HasExited) {
        Stop-Process -Id $proc.Id -Force
    }
    exit 1
}

Write-Host "[READY] Backend is healthy: $baseUrl/healthz"
Write-Host "[READY] Press Ctrl+C to stop if running in this console"

Wait-Process -Id $proc.Id
exit $proc.ExitCode
