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

Write-Host '=== S4 receipt decisions: duplicate / unknown / stale / cross / invalid payload ==='
$rd = Login 'rd_wang' 'rd123456'
$lead = Login 'lead_li' 'lead123456'
$ops = Login 'ops_zhao' 'ops123456'

$app = Step 'create application interval=30' (Call 'POST' '/applications' @{ reason = 'S4 receipt decision matrix'; interval_seconds = 30 } $rd $null)
$appId = $app.Data.data.id
Step 'submit application' (Call 'POST' "/applications/$appId/submit" $null $rd $null) | Out-Null
Step 'approve application' (Call 'POST' "/applications/$appId/approve" $null $lead $null) | Out-Null

$created = Step 'create activity pilots=V1,V4' (Call 'POST' '/activities' @{ application_id = $appId; pilot_vehicles = @('V1', 'V4') } $ops $null)
$activityId = $created.Data.data.activity.id
$pilot = Step 'start pilot' (Call 'POST' "/activities/$activityId/pilot" $null $ops $null)
$v1 = ($pilot.Data.data.rows | Where-Object { $_.vehicle_id -eq 'V1' })
$v4 = ($pilot.Data.data.rows | Where-Object { $_.vehicle_id -eq 'V4' })
$v1r = [int]$v1.version
$v4r = [int]$v4.version
Write-Host ("    pilot versions: V1=v{0} V4=v{1}" -f $v1r, $v4r)

$pull = Step 'V1 pull config' (Call 'POST' '/vehicle/pull' @{ vehicle_id = 'V1' } $null 'v1-token')
Write-Host ("    pull version=v{0} interval={1}s" -f $pull.Data.data.version, $pull.Data.data.interval_seconds)

$ok = Step 'V1 success first time' (Call 'POST' '/vehicle/receipts' @{ vehicle_id = 'V1'; version = $v1r; result = 'success' } $null 'v1-token')
Write-Host ("    accepted={0} status={1}" -f $ok.Data.data.accepted, $ok.Data.data.release_status)

$dup = Step 'V1 duplicate success' (Call 'POST' '/vehicle/receipts' @{ vehicle_id = 'V1'; version = $v1r; result = 'success' } $null 'v1-token')
Write-Host ("    accepted={0} reason={1}" -f $dup.Data.data.accepted, $dup.Data.data.reason)

$unknown = Step 'V1 unknown version v9999' (Call 'POST' '/vehicle/receipts' @{ vehicle_id = 'V1'; version = 9999; result = 'success' } $null 'v1-token')
Write-Host ("    accepted={0} reason={1}" -f $unknown.Data.data.accepted, $unknown.Data.data.reason)

$stale = Step 'V1 stale version' (Call 'POST' '/vehicle/receipts' @{ vehicle_id = 'V1'; version = ($v1r - 1); result = 'success' } $null 'v1-token')
Write-Host ("    accepted={0} reason={1}" -f $stale.Data.data.accepted, $stale.Data.data.reason)

$cross = Step 'V1 token claims V4' (Call 'POST' '/vehicle/receipts' @{ vehicle_id = 'V4'; version = $v4r; result = 'success' } $null 'v1-token')
Write-Host ("    accepted={0} reason={1}" -f $cross.Data.data.accepted, $cross.Data.data.reason)

$invalid = Step 'V4 invalid payload' (Call 'POST' '/vehicle/receipts' @{ vehicle_id = 'V4'; version = $v4r; result = 'totally-unknown' } $null 'v4-token')
Write-Host ("    accepted={0} reason={1}" -f $invalid.Data.data.accepted, $invalid.Data.data.reason)

$v4ok = Step 'V4 success' (Call 'POST' '/vehicle/receipts' @{ vehicle_id = 'V4'; version = $v4r; result = 'success' } $null 'v4-token')
Write-Host ("    accepted={0} status={1}" -f $v4ok.Data.data.accepted, $v4ok.Data.data.release_status)

$promoted = Step 'confirm promotion' (Call 'POST' "/activities/$activityId/promote" $null $ops $null)
$versions = @{}
foreach ($row in $promoted.Data.data.rows) { $versions[$row.vehicle_id] = [int]$row.version }
Write-Host ("    status={0} V2=v{1} V4=v{2} V5=v{3}" -f $promoted.Data.data.activity.status, $versions['V2'], $versions['V4'], $versions['V5'])

foreach ($vehicle in @('V2', 'V3', 'V5')) {
    $token = ($vehicle.ToLower()) + '-token'
    $r = Step "$vehicle promotion success" (Call 'POST' '/vehicle/receipts' @{ vehicle_id = $vehicle; version = $versions[$vehicle]; result = 'success' } $null $token)
    Write-Host ("    accepted={0}" -f $r.Data.data.accepted)
}

$completed = Step 'lead marks completed' (Call 'POST' "/activities/$activityId/complete" $null $lead $null)
Write-Host ("    status={0} success={1} failed={2} timeout={3} effective={4}" -f $completed.Data.data.activity.status, $completed.Data.data.summary.success, $completed.Data.data.summary.failed, $completed.Data.data.summary.timeout, $completed.Data.data.summary.effective)

$rejected = Step 'list rejected receipts' (Call 'GET' '/rejected-receipts' $null $ops $null)
Write-Host ("    rejected total={0}" -f $rejected.Data.data.items.Count)
foreach ($item in $rejected.Data.data.items) {
    Write-Host ("    {0} v{1} reason={2} detail={3}" -f $item.vehicle_id, $item.version, $item.reason, $item.detail)
}
$audit = Step 'list audits' (Call 'GET' '/audits' $null $ops $null)
Write-Host ("    audits total={0}" -f $audit.Data.data.items.Count)
