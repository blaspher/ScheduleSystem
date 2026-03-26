param(
    [string]$EnvFile = ".env",
    [string]$BaseUrl = "",
    [switch]$KeepData,
    [switch]$Cleanup
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

if ($KeepData -and $Cleanup) {
    throw "Use either -KeepData or -Cleanup, not both"
}
if (-not $KeepData -and -not $Cleanup) {
    $KeepData = $true
}

function Import-DotEnv {
    param([string]$Path)

    if (-not (Test-Path $Path)) {
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

function Invoke-JsonRequest {
    param(
        [Parameter(Mandatory=$true)][string]$Method,
        [Parameter(Mandatory=$true)][string]$Url,
        [object]$Body = $null,
        [string]$Token = ""
    )

    $headers = @{}
    if ($Token -ne "") {
        $headers["Authorization"] = "Bearer $Token"
    }

    if ($null -ne $Body) {
        $json = $Body | ConvertTo-Json -Depth 10
        return Invoke-RestMethod -Method $Method -Uri $Url -Headers $headers -ContentType "application/json" -Body $json -TimeoutSec 20
    }

    return Invoke-RestMethod -Method $Method -Uri $Url -Headers $headers -TimeoutSec 20
}

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
Import-DotEnv -Path (Join-Path $repoRoot $EnvFile)

if ($BaseUrl -eq "") {
    $port = if ($env:SERVER_PORT) { $env:SERVER_PORT } else { "8080" }
    $BaseUrl = "http://127.0.0.1:$port"
}

$summary = New-Object System.Collections.Generic.List[object]
function Add-Step {
    param([string]$Name, [bool]$Passed, [string]$Detail)
    $summary.Add([PSCustomObject]@{ Step = $Name; Passed = $Passed; Detail = $Detail }) | Out-Null
    if ($Passed) {
        Write-Host "[PASS] $Name - $Detail"
    } else {
        Write-Host "[FAIL] $Name - $Detail"
    }
}

$timestamp = Get-Date -Format "yyyyMMddHHmmss"
$suffix = "$timestamp-$([int](Get-Random -Minimum 100 -Maximum 999))"
$userA = "smokeA_$suffix"
$userB = "smokeB_$suffix"
$pwd = "Passw0rd!123"

$tokenA = ""
$tokenB = ""
$userAID = 0
$userBID = 0
$eventID = 0
$meetingID = 0

try {
    $null = Invoke-JsonRequest -Method "GET" -Url "$BaseUrl/healthz"
    Add-Step -Name "health check" -Passed $true -Detail "service reachable"

    $regA = Invoke-JsonRequest -Method "POST" -Url "$BaseUrl/api/v1/auth/register" -Body @{ username = $userA; password = $pwd }
    $tokenA = $regA.data.access_token
    $userAID = [int]$regA.data.user.id
    Add-Step -Name "register userA" -Passed $true -Detail "id=$userAID"

    $loginA = Invoke-JsonRequest -Method "POST" -Url "$BaseUrl/api/v1/auth/login" -Body @{ username = $userA; password = $pwd }
    $tokenA = $loginA.data.access_token
    Add-Step -Name "login userA" -Passed $true -Detail "token issued"

    $regB = Invoke-JsonRequest -Method "POST" -Url "$BaseUrl/api/v1/auth/register" -Body @{ username = $userB; password = $pwd }
    $tokenB = $regB.data.access_token
    $userBID = [int]$regB.data.user.id
    Add-Step -Name "register userB" -Passed $true -Detail "id=$userBID"

    $loginB = Invoke-JsonRequest -Method "POST" -Url "$BaseUrl/api/v1/auth/login" -Body @{ username = $userB; password = $pwd }
    $tokenB = $loginB.data.access_token
    Add-Step -Name "login userB" -Passed $true -Detail "token issued"

    $startEvent = (Get-Date).AddHours(1)
    $endEvent = $startEvent.AddHours(1)
    $eventResp = Invoke-JsonRequest -Method "POST" -Url "$BaseUrl/api/v1/events" -Token $tokenA -Body @{
        title = "Smoke Event"
        description = "created by smoke test"
        event_type = "personal"
        visibility = "public"
        start_time = $startEvent.ToString("o")
        end_time = $endEvent.ToString("o")
        location = "desk"
    }
    $eventID = [int]$eventResp.data.id
    Add-Step -Name "create event" -Passed $true -Detail "event_id=$eventID"

    $eventsResp = Invoke-JsonRequest -Method "GET" -Url "$BaseUrl/api/v1/events" -Token $tokenA
    $eventCount = @($eventsResp.data).Count
    Add-Step -Name "list events" -Passed $true -Detail "count=$eventCount"

    # Relation direction is explicit and aligned with permission model:
    # userB is viewer, userA is calendar owner -> set relation userB -> userA.
    $relResp = Invoke-JsonRequest -Method "POST" -Url "$BaseUrl/api/v1/relations" -Token $tokenB -Body @{
        target_user_id = $userAID
        can_view_calendar = $true
    }
    Add-Step -Name "set relation userB->userA" -Passed $true -Detail "can_view_calendar=$($relResp.data.can_view_calendar)"

    $calendarDate = $startEvent.ToString("yyyy-MM-dd")
    $calendarResp = Invoke-JsonRequest -Method "GET" -Url "$BaseUrl/api/v1/users/$userAID/calendar?view=day&date=$calendarDate" -Token $tokenB
    $calendarCount = @($calendarResp.data.events).Count
    Add-Step -Name "get shared calendar" -Passed $true -Detail "events=$calendarCount"

    $meetingStart = (Get-Date).AddHours(3)
    $meetingEnd = $meetingStart.AddHours(1)
    $meetingResp = Invoke-JsonRequest -Method "POST" -Url "$BaseUrl/api/v1/meetings" -Token $tokenA -Body @{
        title = "Smoke Meeting"
        description = "meeting smoke"
        visibility = "private"
        start_time = $meetingStart.ToString("o")
        end_time = $meetingEnd.ToString("o")
        location = "online"
        attendee_ids = @($userBID)
    }
    $meetingID = [int]$meetingResp.data.meeting_id
    Add-Step -Name "create meeting" -Passed $true -Detail "meeting_id=$meetingID"

    $invitesResp = Invoke-JsonRequest -Method "GET" -Url "$BaseUrl/api/v1/meetings/invitations" -Token $tokenB
    $inviteCount = @($invitesResp.data).Count
    Add-Step -Name "list invitations" -Passed $true -Detail "count=$inviteCount"

    $acceptResp = Invoke-JsonRequest -Method "POST" -Url "$BaseUrl/api/v1/meetings/$meetingID/accept" -Token $tokenB
    Add-Step -Name "accept invitation" -Passed $true -Detail "status=$($acceptResp.data.status)"
}
catch {
    Add-Step -Name "smoke flow" -Passed $false -Detail $_.Exception.Message
}

if ($Cleanup) {
    Write-Host "[INFO] Cleanup mode: best-effort cleanup started"
    try {
        if ($tokenA -ne "" -and $eventID -gt 0) {
            $null = Invoke-JsonRequest -Method "DELETE" -Url "$BaseUrl/api/v1/events/$eventID" -Token $tokenA
            Add-Step -Name "cleanup delete event" -Passed $true -Detail "event_id=$eventID"
        }
    }
    catch {
        Add-Step -Name "cleanup delete event" -Passed $false -Detail $_.Exception.Message
    }

    try {
        if ($tokenB -ne "" -and $userAID -gt 0) {
            $null = Invoke-JsonRequest -Method "POST" -Url "$BaseUrl/api/v1/relations" -Token $tokenB -Body @{
                target_user_id = $userAID
                can_view_calendar = $false
            }
            Add-Step -Name "cleanup relation reset" -Passed $true -Detail "userB->userA=false"
        }
    }
    catch {
        Add-Step -Name "cleanup relation reset" -Passed $false -Detail $_.Exception.Message
    }
} else {
    Write-Host "[INFO] KeepData mode: leaving smoke data in database"
}

Write-Host ""
Write-Host "========== Smoke Test Summary =========="
$summary | ForEach-Object {
    $flag = if ($_.Passed) { "PASS" } else { "FAIL" }
    Write-Host ("[{0}] {1} - {2}" -f $flag, $_.Step, $_.Detail)
}

$failed = @($summary | Where-Object { -not $_.Passed }).Count
if ($failed -gt 0) {
    Write-Host "[RESULT] FAILED ($failed step(s))"
    exit 1
}

Write-Host "[RESULT] PASSED"
exit 0
