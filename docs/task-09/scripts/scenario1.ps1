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
        $reader = New-Object System.IO.StreamReader($resp.GetResponseStream())
        $text = $reader.ReadToEnd()
        return [pscustomobject]@{ Status = [int]$resp.StatusCode; Data = ($text | ConvertFrom-Json) }
    }
}

function Login($account, $password) {
    $r = Call 'POST' '/login' @{ account = $account; password = $password } $null $null
    if ($r.Status -ne 200) { throw "login failed: $account" }
    return $r.Data.data.token
}

function Step($label, $result) {
    Write-Output ("--- {0}: HTTP {1}" -f $label, $result.Status)
    if ($result.Status -ge 400) {
        Write-Output ("    error: {0} | {1}" -f $result.Data.error.code, $result.Data.error.message)
    }
    return $result
}

Write-Output '=== S1 happy path: application -> approval -> pilot -> promotion -> completed ==='
$rd = Login 'rd_wang' 'rd123456'
$lead = Login 'lead_li' 'lead123456'
$ops = Login 'ops_zhao' 'ops123456'
Write-Output ("tokens acquired: rd={0} lead={1} ops={2}" -f $rd.Length, $lead.Length, $ops.Length)

$app = Step 'create application interval=10' (Call 'POST' '/applications' @{ reason = 'S1 happy path: investigate motor controller frame loss'; interval_seconds = 10 } $rd $null)
$appId = $app.Data.data.id
Step 'submit application' (Call 'POST' "/applications/$appId/submit" $null $rd $null) | Out-Null
$approved = Step 'approve application' (Call 'POST' "/applications/$appId/approve" $null $lead $null)
Write-Output ("    application status={0} decided_by={1}" -f $approved.Data.data.status, $approved.Data.data.decided_by)

$created = Step 'create activity pilots=V1,V3' (Call 'POST' '/activities' @{ application_id = $appId; pilot_vehicles = @('V1', 'V3') } $ops $null)
$activityId = $created.Data.data.activity.id
Write-Output ("    activity={0} status={1} promotion={2}" -f $activityId, $created.Data.data.activity.status, ($created.Data.data.activity.promotion_vehicles -join ','))

$pilot = Step 'start pilot' (Call 'POST' "/activities/$activityId/pilot" $null $ops $null)
foreach ($row in $pilot.Data.data.rows) {
    Write-Output ("    {0}: released={1} version={2} expected={3} status={4}" -f $row.vehicle_id, $row.released, $row.version, $row.expected_interval, $row.status)
}

$pull1 = Step 'V1 pull config' (Call 'POST' '/vehicle/pull' @{ vehicle_id = 'V1' } $null 'v1-token')
Write-Output ("    V1 pull: v{0} / {1}s has_release={2}" -f $pull1.Data.data.version, $pull1.Data.data.interval_seconds, $pull1.Data.data.has_release)
$r1 = Step 'V1 receipt success v2' (Call 'POST' '/vehicle/receipts' @{ vehicle_id = 'V1'; version = 2; result = 'success' } $null 'v1-token')
Write-Output ("    accepted={0} effective={1}s status={2}" -f $r1.Data.data.accepted, $r1.Data.data.effective_interval, $r1.Data.data.release_status)

$pull3 = Step 'V3 pull config' (Call 'POST' '/vehicle/pull' @{ vehicle_id = 'V3' } $null 'v3-token')
Write-Output ("    V3 pull: v{0} / {1}s" -f $pull3.Data.data.version, $pull3.Data.data.interval_seconds)
$r3 = Step 'V3 receipt success v2' (Call 'POST' '/vehicle/receipts' @{ vehicle_id = 'V3'; version = 2; result = 'success' } $null 'v3-token')
Write-Output ("    accepted={0} activity_status={1}" -f $r3.Data.data.accepted, $r3.Data.data.activity_status)

$detail = Step 'get current activity' (Call 'GET' '/current-activity' $null $ops $null)
Write-Output ("    activity status={0} success={1} timeout={2} can_promote={3}" -f $detail.Data.data.activity.status, $detail.Data.data.summary.success, $detail.Data.data.summary.timeout, $detail.Data.data.actions.can_promote)

$offline = Step 'V2 pull while offline (expect reject)' (Call 'POST' '/vehicle/pull' @{ vehicle_id = 'V2' } $null 'v2-token')
Step 'ops sets V2 online' (Call 'POST' '/vehicles/V2/online' @{ online = $true } $ops $null) | Out-Null
$pull2 = Step 'V2 pull after online' (Call 'POST' '/vehicle/pull' @{ vehicle_id = 'V2' } $null 'v2-token')
Write-Output ("    V2 pull: v{0} / {1}s" -f $pull2.Data.data.version, $pull2.Data.data.interval_seconds)

$promoted = Step 'confirm promotion' (Call 'POST' "/activities/$activityId/promote" $null $ops $null)
Write-Output ("    status={0} rows:" -f $promoted.Data.data.activity.status)
foreach ($row in $promoted.Data.data.rows) {
    Write-Output ("    {0}: released={1} batch={2} version={3}" -f $row.vehicle_id, $row.released, $row.batch, $row.version)
}

foreach ($vehicle in @('V2', 'V4', 'V5')) {
    $token = ($vehicle.ToLower()) + '-token'
    Step "$vehicle receipt success v2" (Call 'POST' '/vehicle/receipts' @{ vehicle_id = $vehicle; version = 2; result = 'success' } $null $token) | Out-Null
}

$completed = Step 'lead marks completed' (Call 'POST' "/activities/$activityId/complete" $null $lead $null)
Write-Output ("    status={0} success={1} effective={2} released={3}" -f $completed.Data.data.activity.status, $completed.Data.data.summary.success, $completed.Data.data.summary.effective, $completed.Data.data.summary.released)

$vehicles = Step 'list vehicles' (Call 'GET' '/vehicles' $null $ops $null)
foreach ($item in $vehicles.Data.data.items) {
    Write-Output ("    {0}: online={1} expected=v{2}/{3}s effective=v{4}/{5}s" -f $item.id, $item.online, $item.current_version, $item.expected_interval, $item.effective_version, $item.effective_interval)
}
