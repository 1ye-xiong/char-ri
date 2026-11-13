$ErrorActionPreference = 'Stop'
$base = 'http://127.0.0.1:8081/api/v1/fleetlog'

function Call($method, $path, $body, $token, $vehicleToken) {
    $headers = @{}
    if ($token) { $headers['Authorization'] = "Bearer $token" }
    if ($vehicleToken) { $headers['X-Vehicle-Token'] = $vehicleToken }
    $params = @{ Method = $method; Uri = "$base$path"; Headers = $headers; UseBasicParsing = $true }
    if ($null -ne $body) {
        $params['ContentType'] = 'application/json'
        $params['Body'] = ($body | ConvertTo-Json -Depth 8 -Compress)
    }
    try {
        $r = Invoke-WebRequest @params
        return [pscustomobject]@{ Status = [int]$r.StatusCode; Data = ($r.Content | ConvertFrom-Json) }
    } catch {
        $resp = $_.Exception.Response
        if ($null -eq $resp) { throw }
        $text = $_.ErrorDetails.Message
        if (-not $text) {
            $reader = New-Object System.IO.StreamReader($resp.GetResponseStream())
            $text = $reader.ReadToEnd()
        }
        return [pscustomobject]@{ Status = [int]$resp.StatusCode; Data = ($text | ConvertFrom-Json) }
    }
}

function Login($account, $password) {
    $r = Call 'POST' '/login' @{ account = $account; password = $password } $null $null
    if ($r.Status -ne 200) { throw "login failed: $account" }
    return $r.Data.data.token
}

function Step($label, $result) {
    Write-Host ("--- {0}: HTTP {1}" -f $label, $result.Status)
    if ($result.Status -ge 400) {
        Write-Host ("    error: {0} | {1}" -f $result.Data.error.code, $result.Data.error.message)
    }
    return $result
}

Write-Host '=== S5 identity, role, single-activity and offline constraints ==='
$rd = Login 'rd_wang' 'rd123456'
$lead = Login 'lead_li' 'lead123456'
$ops = Login 'ops_zhao' 'ops123456'

Step 'list activities without token' (Call 'GET' '/activities' $null $null $null) | Out-Null
Step 'list activities with bogus token' (Call 'GET' '/activities' $null 'bogus-token' $null) | Out-Null

$createdApp = Call 'POST' '/applications' @{ reason = 'S5 identity and constraint matrix'; interval_seconds = 15 } $rd $null
$appId = $createdApp.Data.data.id
Step 'submit application' (Call 'POST' "/applications/$appId/submit" $null $rd $null) | Out-Null
Step 'requester approves own application (role denied)' (Call 'POST' "/applications/$appId/approve" $null $rd $null) | Out-Null
Step 'approver creates application (role denied)' (Call 'POST' '/applications' @{ reason = 'approver must not author'; interval_seconds = 15 } $lead $null) | Out-Null
Step 'approve application' (Call 'POST' "/applications/$appId/approve" $null $lead $null) | Out-Null

$created = Step 'create activity A pilots=V1,V3' (Call 'POST' '/activities' @{ application_id = $appId; pilot_vehicles = @('V1', 'V3') } $ops $null)
$activityId = $created.Data.data.activity.id
$pilot = Step 'start pilot' (Call 'POST' "/activities/$activityId/pilot" $null $ops $null)

$app2 = Call 'POST' '/applications' @{ reason = 'S5 second application for conflict'; interval_seconds = 20 } $rd $null
Call 'POST' "/applications/$($app2.Data.data.id)/submit" $null $rd $null | Out-Null
Call 'POST' "/applications/$($app2.Data.data.id)/approve" $null $lead $null | Out-Null
Step 'create second activity while A open (expect conflict)' (Call 'POST' '/activities' @{ application_id = $app2.Data.data.id; pilot_vehicles = @('V2', 'V5') } $ops $null) | Out-Null
Step 'complete during piloting (expect conflict)' (Call 'POST' "/activities/$activityId/complete" $null $lead $null) | Out-Null

Step 'ops sets V2 offline' (Call 'POST' '/vehicles/V2/online' @{ online = $false } $ops $null) | Out-Null
Step 'V2 pull while offline (expect reject)' (Call 'POST' '/vehicle/pull' @{ vehicle_id = 'V2' } $null 'v2-token') | Out-Null
Step 'ops sets V2 online' (Call 'POST' '/vehicles/V2/online' @{ online = $true } $ops $null) | Out-Null

$v1 = ($pilot.Data.data.rows | Where-Object { $_.vehicle_id -eq 'V1' })
$v3 = ($pilot.Data.data.rows | Where-Object { $_.vehicle_id -eq 'V3' })
$v1r = [int]$v1.version
$v3r = [int]$v3.version
Write-Host ("    pilot versions: V1=v{0} V3=v{1}" -f $v1r, $v3r)
$r1 = Step 'V1 pilot success' (Call 'POST' '/vehicle/receipts' @{ vehicle_id = 'V1'; version = $v1r; result = 'success' } $null 'v1-token')
$r3 = Step 'V3 pilot success' (Call 'POST' '/vehicle/receipts' @{ vehicle_id = 'V3'; version = $v3r; result = 'success' } $null 'v3-token')
Write-Host ("    V1 accepted={0} V3 accepted={1}" -f $r1.Data.data.accepted, $r3.Data.data.accepted)

$promoted = Step 'confirm promotion' (Call 'POST' "/activities/$activityId/promote" $null $ops $null)
Step 'complete before promotion receipts (expect unconfirmed)' (Call 'POST' "/activities/$activityId/complete" $null $lead $null) | Out-Null
$versions = @{}
foreach ($row in $promoted.Data.data.rows) { $versions[$row.vehicle_id] = [int]$row.version }
foreach ($vehicle in @('V2', 'V4', 'V5')) {
    $token = ($vehicle.ToLower()) + '-token'
    $pull = Step "$vehicle pull config" (Call 'POST' '/vehicle/pull' @{ vehicle_id = $vehicle } $null $token)
    Write-Host ("    pull version=v{0} interval={1}s" -f $pull.Data.data.version, $pull.Data.data.interval_seconds)
    $r = Step "$vehicle success" (Call 'POST' '/vehicle/receipts' @{ vehicle_id = $vehicle; version = $versions[$vehicle]; result = 'success' } $null $token)
    Write-Host ("    accepted={0}" -f $r.Data.data.accepted)
}

$completed = Step 'lead marks completed' (Call 'POST' "/activities/$activityId/complete" $null $lead $null)
Write-Host ("    status={0} success={1} effective={2}" -f $completed.Data.data.activity.status, $completed.Data.data.summary.success, $completed.Data.data.summary.effective)
$detail = Step 'activity detail' (Call 'GET' "/activities/$activityId" $null $ops $null)
Write-Host ("    audits on activity={0} rejected_receipts={1} timeout_events={2}" -f $detail.Data.data.audits.Count, $detail.Data.data.rejected_receipts.Count, $detail.Data.data.timeout_events.Count)
foreach ($a in $detail.Data.data.audits) { Write-Host ("    audit {0} by {1}({2}): {3}" -f $a.action, $a.actor_id, $a.actor_role, $a.detail) }
