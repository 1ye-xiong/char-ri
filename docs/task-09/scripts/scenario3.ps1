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

Write-Host '=== S3 timeout and late receipt (teaching clock, timeout=60s) ==='
$rd = Login 'rd_wang' 'rd123456'
$lead = Login 'lead_li' 'lead123456'
$ops = Login 'ops_zhao' 'ops123456'

$app = Step 'create application interval=12' (Call 'POST' '/applications' @{ reason = 'S3 timeout and late receipt'; interval_seconds = 12 } $rd $null)
$appId = $app.Data.data.id
Step 'submit application' (Call 'POST' "/applications/$appId/submit" $null $rd $null) | Out-Null
Step 'approve application' (Call 'POST' "/applications/$appId/approve" $null $lead $null) | Out-Null

$created = Step 'create activity pilots=V2,V3' (Call 'POST' '/activities' @{ application_id = $appId; pilot_vehicles = @('V2', 'V3') } $ops $null)
$activityId = $created.Data.data.activity.id
$pilot = Step 'start pilot' (Call 'POST' "/activities/$activityId/pilot" $null $ops $null)
$v2 = ($pilot.Data.data.rows | Where-Object { $_.vehicle_id -eq 'V2' })
$v3 = ($pilot.Data.data.rows | Where-Object { $_.vehicle_id -eq 'V3' })
$v2r = [int]$v2.version
$v3r = [int]$v3.version
Write-Host ("    pilot versions: V2=v{0} V3=v{1}" -f $v2r, $v3r)

Step 'V2 pull config (will not report)' (Call 'POST' '/vehicle/pull' @{ vehicle_id = 'V2' } $null 'v2-token') | Out-Null
$r3 = Step 'V3 success' (Call 'POST' '/vehicle/receipts' @{ vehicle_id = 'V3'; version = $v3r; result = 'success' } $null 'v3-token')
Write-Host ("    V3 accepted={0}" -f $r3.Data.data.accepted)

$deadline = (Get-Date).AddSeconds(85)
do {
    Start-Sleep -Seconds 5
    $snapshot = Call 'GET' "/activities/$activityId" $null $ops $null
    $row2 = ($snapshot.Data.data.rows | Where-Object { $_.vehicle_id -eq 'V2' })
    Write-Host ("    waiting: V2 status={0} timeout_at={1}" -f $row2.status, $row2.timeout_at)
} while ($row2.status -ne 'timeout' -and (Get-Date) -lt $deadline)

if ($row2.status -ne 'timeout') { throw 'timeout was not marked within deadline' }

$late = Step 'V2 late success after timeout' (Call 'POST' '/vehicle/receipts' @{ vehicle_id = 'V2'; version = $v2r; result = 'success' } $null 'v2-token')
Write-Host ("    accepted={0} status={1} effective={2}s" -f $late.Data.data.accepted, $late.Data.data.release_status, $late.Data.data.effective_interval)

$detail = Step 'activity detail after late receipt' (Call 'GET' "/activities/$activityId" $null $ops $null)
$row2 = ($detail.Data.data.rows | Where-Object { $_.vehicle_id -eq 'V2' })
Write-Host ("    V2 status={0} late={1} effective={2}s" -f $row2.status, $row2.late, $row2.effective_interval)
Write-Host ("    timeout_events={0}" -f $detail.Data.data.timeout_events.Count)
foreach ($e in $detail.Data.data.timeout_events) { Write-Host ("    timeout {0} v{1} at={2} resolved={3}" -f $e.vehicle_id, $e.version, $e.timed_out_at, $e.resolved_at) }

$stopped = Step 'stop activity after timeout' (Call 'POST' "/activities/$activityId/stop" @{ reason = 'S3 timeout rehearsal stop' } $ops $null)
Write-Host ("    status={0}" -f $stopped.Data.data.activity.status)
Step 'lead approves restore' (Call 'POST' "/activities/$activityId/restore/approve" $null $lead $null) | Out-Null
$restoring = Step 'ops starts restore' (Call 'POST' "/activities/$activityId/restore/start" $null $ops $null)
$versions = @{}
foreach ($row in $restoring.Data.data.rows) { if ($row.released) { $versions[$row.vehicle_id] = [int]$row.version } }
Write-Host ("    restore status={0} versions V2=v{1} V3=v{2}" -f $restoring.Data.data.activity.status, $versions['V2'], $versions['V3'])
foreach ($vehicle in @('V2', 'V3')) {
    $token = ($vehicle.ToLower()) + '-token'
    $r = Step "$vehicle restore success" (Call 'POST' '/vehicle/receipts' @{ vehicle_id = $vehicle; version = $versions[$vehicle]; result = 'success' } $null $token)
    Write-Host ("    accepted={0}" -f $r.Data.data.accepted)
}
$restored = Step 'lead marks restored' (Call 'POST' "/activities/$activityId/restored" $null $lead $null)
Write-Host ("    status={0} effective={1}" -f $restored.Data.data.activity.status, $restored.Data.data.summary.effective)
