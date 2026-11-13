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

Write-Output '=== S2 pilot failure -> stop -> restore (R-04 / R-08 / R-09) ==='
$rd = Login 'rd_wang' 'rd123456'
$lead = Login 'lead_li' 'lead123456'
$ops = Login 'ops_zhao' 'ops123456'

$app = Step 'create application interval=20' (Call 'POST' '/applications' @{ reason = 'S2 pilot failure and restore rehearsal'; interval_seconds = 20 } $rd $null)
$appId = $app.Data.data.id
Step 'submit application' (Call 'POST' "/applications/$appId/submit" $null $rd $null) | Out-Null
Step 'approve application' (Call 'POST' "/applications/$appId/approve" $null $lead $null) | Out-Null

$created = Step 'create activity pilots=V1,V4' (Call 'POST' '/activities' @{ application_id = $appId; pilot_vehicles = @('V1', 'V4') } $ops $null)
$activityId = $created.Data.data.activity.id
$pilot = Step 'start pilot' (Call 'POST' "/activities/$activityId/pilot" $null $ops $null)
$v1r = [int](($pilot.Data.data.rows | Where-Object { $_.vehicle_id -eq 'V1' }).version)
$v4r = [int](($pilot.Data.data.rows | Where-Object { $_.vehicle_id -eq 'V4' }).version)
Write-Host ("    pilot versions: V1=v{0} V4=v{1}" -f $v1r, $v4r)

$r1 = Step "V1 receipt success v$v1r" (Call 'POST' '/vehicle/receipts' @{ vehicle_id = 'V1'; version = $v1r; result = 'success' } $null 'v1-token')
Write-Host ("    accepted={0} effective={1}s" -f $r1.Data.data.accepted, $r1.Data.data.effective_interval)
$r4 = Step "V4 receipt failure v$v4r (write failed)" (Call 'POST' '/vehicle/receipts' @{ vehicle_id = 'V4'; version = $v4r; result = 'failure'; failure_reason = 'write config failed: checksum mismatch' } $null 'v4-token')
Write-Host ("    accepted={0} status={1} effective={2}s" -f $r4.Data.data.accepted, $r4.Data.data.release_status, $r4.Data.data.effective_interval)

$blocked = Step 'confirm promotion must be rejected' (Call 'POST' "/activities/$activityId/promote" $null $ops $null)
$detail = Step 'get current activity' (Call 'GET' '/current-activity' $null $ops $null)
Write-Host ("    activity status={0} success={1} failed={2}" -f $detail.Data.data.activity.status, $detail.Data.data.summary.success, $detail.Data.data.summary.failed)
Write-Host ("    promote_blocked_reason={0}" -f $detail.Data.data.actions.promote_blocked_reason)
foreach ($row in $detail.Data.data.rows) {
    Write-Host ("    {0}: released={1} batch={2} version={3} status={4} failure={5}" -f $row.vehicle_id, $row.released, $row.batch, $row.version, $row.status, $row.failure_reason)
}

$stopped = Step 'stop activity (pilot failure)' (Call 'POST' "/activities/$activityId/stop" @{ reason = 'pilot V4 failed, stop before promotion' } $ops $null)
Write-Host ("    status={0} stopped_by={1}" -f $stopped.Data.data.activity.status, $stopped.Data.data.activity.stopped_by)
Write-Host ("    stop_scope={0}" -f $stopped.Data.data.activity.stop_scope)
Write-Host ("    V2 row: released={0} status={1}" -f ($stopped.Data.data.rows | Where-Object { $_.vehicle_id -eq 'V2' }).released, ($stopped.Data.data.rows | Where-Object { $_.vehicle_id -eq 'V2' }).status)

Step 'restore without approval must be rejected' (Call 'POST' "/activities/$activityId/restore/start" $null $ops $null) | Out-Null
Step 'lead approves restore' (Call 'POST' "/activities/$activityId/restore/approve" $null $lead $null) | Out-Null
$restoring = Step 'ops starts restore' (Call 'POST' "/activities/$activityId/restore/start" $null $ops $null)
Write-Host ("    status={0}" -f $restoring.Data.data.activity.status)
foreach ($row in $restoring.Data.data.rows) {
    Write-Host ("    {0}: released={1} batch={2} version={3} expected={4}s status={5}" -f $row.vehicle_id, $row.released, $row.batch, $row.version, $row.expected_interval, $row.status)
}

$restoreVersion = ($restoring.Data.data.rows | Where-Object { $_.vehicle_id -eq 'V1' }).version
foreach ($vehicle in @('V1', 'V4')) {
    $token = ($vehicle.ToLower()) + '-token'
    Step "$vehicle restore receipt success v$restoreVersion" (Call 'POST' '/vehicle/receipts' @{ vehicle_id = $vehicle; version = $restoreVersion; result = 'success' } $null $token) | Out-Null
}
$restored = Step 'lead marks restore completed' (Call 'POST' "/activities/$activityId/restored" $null $lead $null)
Write-Host ("    status={0} effective count={1}" -f $restored.Data.data.activity.status, $restored.Data.data.summary.effective)
foreach ($row in $restored.Data.data.rows) {
    Write-Host ("    {0}: status={1} effective={2}s" -f $row.vehicle_id, $row.status, $row.effective_interval)
}
