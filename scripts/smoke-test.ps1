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

function Get-StatusCodeFromError {
    param([System.Management.Automation.ErrorRecord]$ErrorRecord)

    if ($null -eq $ErrorRecord -or $null -eq $ErrorRecord.Exception) {
        return $null
    }
    if ($null -eq $ErrorRecord.Exception.Response) {
        return $null
    }

    $statusCode = $ErrorRecord.Exception.Response.StatusCode
    if ($null -eq $statusCode) {
        return $null
    }

    try { return [int]$statusCode.value__ } catch {}
    try { return [int]$statusCode } catch {}
    return $null
}

function Assert-ExpectedHttpStatus {
    param(
        [Parameter(Mandatory=$true)][scriptblock]$Action,
        [Parameter(Mandatory=$true)][int]$ExpectedCode
    )

    try {
        & $Action | Out-Null
    }
    catch {
        $actualCode = Get-StatusCodeFromError -ErrorRecord $_
        if ($actualCode -eq $ExpectedCode) {
            return $actualCode
        }
        throw "expected HTTP $ExpectedCode, got $actualCode, detail=$($_.Exception.Message)"
    }

    throw "expected HTTP $ExpectedCode, but request succeeded"
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
$crossEventID = 0
$meetingID = 0

try {
    $null = Invoke-JsonRequest -Method "GET" -Url "$BaseUrl/healthz"
    Add-Step -Name "health check" -Passed $true -Detail "service reachable"

    $blankStatus = Assert-ExpectedHttpStatus -ExpectedCode 400 -Action {
        Invoke-JsonRequest -Method "POST" -Url "$BaseUrl/api/v1/auth/register" -Body @{ username = "   "; password = $pwd }
    }
    Add-Step -Name "register blank username" -Passed $true -Detail "status=$blankStatus"

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

    $crossStart = (Get-Date).Date.AddDays(1).AddHours(23).AddMinutes(30)
    $crossEnd = $crossStart.AddHours(2)
    $crossResp = Invoke-JsonRequest -Method "POST" -Url "$BaseUrl/api/v1/events" -Token $tokenA -Body @{
        title = "Cross Day Event"
        description = "cross-day verification"
        event_type = "personal"
        visibility = "public"
        start_time = $crossStart.ToString("o")
        end_time = $crossEnd.ToString("o")
        location = "desk"
    }
    $crossEventID = [int]$crossResp.data.id
    Add-Step -Name "create cross-day event" -Passed $true -Detail "event_id=$crossEventID"

    $crossDayDate = $crossEnd.ToString("yyyy-MM-dd")
    $crossDayListResp = Invoke-JsonRequest -Method "GET" -Url "$BaseUrl/api/v1/events?view=day&date=$crossDayDate" -Token $tokenA
    $crossDayIDs = @($crossDayListResp.data | ForEach-Object { [int]$_.id })
    if ($crossDayIDs -notcontains $crossEventID) {
        throw "cross-day event missing in next-day view (event_id=$crossEventID, date=$crossDayDate)"
    }
    Add-Step -Name "cross-day event visible on next day" -Passed $true -Detail "date=$crossDayDate"

    # Relation direction is explicit and aligned with permission model:
    # userB is viewer, userA is calendar owner -> set relation userB -> userA.
    $relResp = Invoke-JsonRequest -Method "POST" -Url "$BaseUrl/api/v1/relations" -Token $tokenB -Body @{
        target_user_id = $userAID
        can_view_calendar = $true
    }
    Add-Step -Name "set relation userB->userA" -Passed $true -Detail "can_view_calendar=$($relResp.data.can_view_calendar)"

    $meetingStart = (Get-Date).AddHours(3)
    $meetingEnd = $meetingStart.AddHours(1)
    $meetingDate = $meetingStart.ToString("yyyy-MM-dd")

    $eventsBeforeResp = Invoke-JsonRequest -Method "GET" -Url "$BaseUrl/api/v1/events" -Token $tokenA
    $eventsBeforeCount = @($eventsBeforeResp.data).Count

    $calendarBeforeResp = Invoke-JsonRequest -Method "GET" -Url "$BaseUrl/api/v1/users/$userAID/calendar?view=day&date=$meetingDate" -Token $tokenB
    $calendarBeforeCount = @($calendarBeforeResp.data.events).Count

    $meetingResp = Invoke-JsonRequest -Method "POST" -Url "$BaseUrl/api/v1/meetings" -Token $tokenA -Body @{
        title = "Smoke Meeting"
        description = "meeting smoke"
        visibility = "public"
        start_time = $meetingStart.ToString("o")
        end_time = $meetingEnd.ToString("o")
        location = "online"
        attendee_ids = @($userBID)
    }
    $meetingID = [int]$meetingResp.data.meeting_id
    Add-Step -Name "create meeting" -Passed $true -Detail "meeting_id=$meetingID"

    $eventsAfterResp = Invoke-JsonRequest -Method "GET" -Url "$BaseUrl/api/v1/events" -Token $tokenA
    $eventsAfterCount = @($eventsAfterResp.data).Count
    if ($eventsAfterCount -ne ($eventsBeforeCount + 1)) {
        throw "event list cache invalidation failed (before=$eventsBeforeCount, after=$eventsAfterCount)"
    }
    Add-Step -Name "event list cache invalidation" -Passed $true -Detail "before=$eventsBeforeCount after=$eventsAfterCount"

    $calendarAfterResp = Invoke-JsonRequest -Method "GET" -Url "$BaseUrl/api/v1/users/$userAID/calendar?view=day&date=$meetingDate" -Token $tokenB
    $calendarAfterCount = @($calendarAfterResp.data.events).Count
    if ($calendarAfterCount -ne ($calendarBeforeCount + 1)) {
        throw "shared calendar cache invalidation failed (before=$calendarBeforeCount, after=$calendarAfterCount)"
    }
    Add-Step -Name "shared calendar cache invalidation" -Passed $true -Detail "before=$calendarBeforeCount after=$calendarAfterCount"

    $invitesResp = Invoke-JsonRequest -Method "GET" -Url "$BaseUrl/api/v1/meetings/invitations" -Token $tokenB
    $inviteCount = @($invitesResp.data).Count
    Add-Step -Name "list invitations" -Passed $true -Detail "count=$inviteCount"

    $acceptResp = Invoke-JsonRequest -Method "POST" -Url "$BaseUrl/api/v1/meetings/$meetingID/accept" -Token $tokenB
    Add-Step -Name "accept invitation" -Passed $true -Detail "status=$($acceptResp.data.status)"

    $conflictStatus = Assert-ExpectedHttpStatus -ExpectedCode 409 -Action {
        Invoke-JsonRequest -Method "POST" -Url "$BaseUrl/api/v1/meetings" -Token $tokenB -Body @{
            title = "Conflict Meeting"
            description = "expect conflict after accepted invitation"
            visibility = "public"
            start_time = $meetingStart.ToString("o")
            end_time = $meetingEnd.ToString("o")
            location = "online"
            attendee_ids = @($userAID)
        }
    }
    Add-Step -Name "create meeting conflict symmetry" -Passed $true -Detail "status=$conflictStatus"
}
catch {
    Add-Step -Name "smoke flow" -Passed $false -Detail $_.Exception.Message
}

if ($Cleanup) {
    Write-Host "[INFO] Cleanup mode: best-effort cleanup started"
    try {
        if ($tokenA -ne "") {
            $deleteIDs = @()
            if ($meetingID -gt 0) { $deleteIDs += $meetingID }
            if ($crossEventID -gt 0) { $deleteIDs += $crossEventID }
            if ($eventID -gt 0) { $deleteIDs += $eventID }

            foreach ($id in $deleteIDs) {
                $null = Invoke-JsonRequest -Method "DELETE" -Url "$BaseUrl/api/v1/events/$id" -Token $tokenA
            }
            Add-Step -Name "cleanup delete events" -Passed $true -Detail "count=$($deleteIDs.Count)"
        }
    }
    catch {
        Add-Step -Name "cleanup delete events" -Passed $false -Detail $_.Exception.Message
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
