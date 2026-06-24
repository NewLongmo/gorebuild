param(
  [string]$ApiBaseUrl = "http://localhost:8080",
  [string]$FrontendUrl = "http://localhost:8081",
  [string]$Account = "",
  [string]$Password = "",
  [int]$TimeoutSec = 10,
  [int]$RetryCount = 30,
  [int]$RetryDelaySec = 2,
  [switch]$ExerciseWrites
)

$ErrorActionPreference = "Stop"
[Console]::OutputEncoding = [System.Text.UTF8Encoding]::new($false)

if ([string]::IsNullOrWhiteSpace($Account)) {
  $Account = if ($env:BOOTSTRAP_ADMIN_ACCOUNT) { $env:BOOTSTRAP_ADMIN_ACCOUNT } else { "admin" }
}
if ([string]::IsNullOrWhiteSpace($Password)) {
  $Password = if ($env:BOOTSTRAP_ADMIN_PASSWORD) { $env:BOOTSTRAP_ADMIN_PASSWORD } else { "admin123" }
}

$ApiBaseUrl = $ApiBaseUrl.TrimEnd("/")
$FrontendUrl = $FrontendUrl.TrimEnd("/")

function Invoke-WithRetry {
  param(
    [string]$Name,
    [scriptblock]$Action
  )

  $attempt = 1
  while ($true) {
    try {
      return & $Action
    } catch {
      if ($attempt -ge $RetryCount) {
        throw "$Name failed after $attempt attempt(s): $($_.Exception.Message)"
      }
      Write-Host "Waiting for $Name (attempt $attempt/$RetryCount)..."
      Start-Sleep -Seconds $RetryDelaySec
      $attempt++
    }
  }
}

function Assert-ApiOk {
  param(
    [string]$Name,
    [object]$Response
  )

  if ($null -eq $Response) {
    throw "$Name returned an empty response"
  }
  if ($Response.code -ne 0) {
    throw "$Name failed: code=$($Response.code) message=$($Response.message)"
  }
  return $Response.data
}

function Invoke-ApiGet {
  param(
    [string]$Path,
    [hashtable]$Headers = @{}
  )
  return Invoke-RestMethod -Method Get -Uri "$ApiBaseUrl$Path" -Headers $Headers -TimeoutSec $TimeoutSec
}

function Invoke-ApiPost {
  param(
    [string]$Path,
    [object]$Body,
    [hashtable]$Headers = @{}
  )
  $json = $Body | ConvertTo-Json -Depth 8
  return Invoke-RestMethod -Method Post -Uri "$ApiBaseUrl$Path" -Headers $Headers -ContentType "application/json" -Body $json -TimeoutSec $TimeoutSec
}

function Invoke-ApiDelete {
  param(
    [string]$Path,
    [hashtable]$Headers = @{}
  )
  return Invoke-RestMethod -Method Delete -Uri "$ApiBaseUrl$Path" -Headers $Headers -TimeoutSec $TimeoutSec
}

function Assert-Page {
  param(
    [string]$Name,
    [object]$Page
  )

  foreach ($field in @("items", "total", "page", "perPage")) {
    if ($null -eq $Page.$field) {
      throw "$Name page missing field: $field"
    }
  }
}

Write-Host "==> Backend liveness"
Invoke-WithRetry "backend liveness" {
  Assert-ApiOk "healthz" (Invoke-ApiGet "/healthz") | Out-Null
}

Write-Host "==> Backend readiness"
Invoke-WithRetry "backend readiness" {
  Assert-ApiOk "readyz" (Invoke-ApiGet "/readyz") | Out-Null
}

Write-Host "==> Auth login"
$loginData = Invoke-WithRetry "auth login" {
  Assert-ApiOk "login" (Invoke-ApiPost "/api/v1/auth/login" @{
    account = $Account
    password = $Password
  })
}
if ([string]::IsNullOrWhiteSpace($loginData.token)) {
  throw "login did not return a token"
}

$authHeaders = @{ Authorization = "Bearer $($loginData.token)" }

Write-Host "==> Auth session"
$me = Assert-ApiOk "me" (Invoke-ApiGet "/api/v1/auth/me" $authHeaders)
if ($me.account -ne $Account) {
  throw "authenticated account mismatch: expected $Account got $($me.account)"
}
if ($me.role -ne "admin") {
  throw "smoke account must be an admin user, got role=$($me.role)"
}

Write-Host "==> Dashboard"
$dashboard = Assert-ApiOk "dashboard" (Invoke-ApiGet "/api/v1/dashboard" $authHeaders)
foreach ($field in @("users", "classes", "orders", "pending", "flashOrders", "flashPending", "queueOrders", "queueRefreshes", "queueSubmit", "queueSubmitFlash", "queueRefresh", "queueRefreshFlash", "activeUsers", "onlineClasses")) {
  if ($null -eq $dashboard.$field) {
    throw "dashboard missing field: $field"
  }
}

Write-Host "==> Settings"
Assert-ApiOk "settings" (Invoke-ApiGet "/api/v1/settings" $authHeaders) | Out-Null

Write-Host "==> Orders and logs"
Assert-Page "orders" (Assert-ApiOk "orders" (Invoke-ApiGet "/api/v1/orders?perPage=5" $authHeaders))
Assert-Page "flash orders" (Assert-ApiOk "flash orders" (Invoke-ApiGet "/api/v1/orders?flashMode=true&perPage=5" $authHeaders))
Assert-Page "normal orders" (Assert-ApiOk "normal orders" (Invoke-ApiGet "/api/v1/orders?flashMode=false&perPage=5" $authHeaders))
Assert-Page "logs" (Assert-ApiOk "logs" (Invoke-ApiGet "/api/v1/logs?perPage=5" $authHeaders))

if ($ExerciseWrites) {
  Write-Host "==> Flash order write path"
  $suffix = [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds()
  $connectorId = $null
  $orderId = $null

  try {
    $connector = Assert-ApiOk "create smoke connector" (Invoke-ApiPost "/api/v1/connectors" @{
      name = "smoke-flash-$suffix"
      baseUrl = "http://127.0.0.1:1"
      kind = "smoke"
      status = "active"
      timeoutMs = 1000
    } $authHeaders)
    $connectorId = [int]$connector.id

    $order = Assert-ApiOk "create smoke flash order" (Invoke-ApiPost "/api/v1/orders" @{
      connectorId = $connectorId
      account = "smoke-$suffix"
      courseName = "Smoke Flash Queue"
      flashMode = $true
      fee = 0
    } $authHeaders)
    $orderId = [int]$order.id

    if (-not $order.flashMode) {
      throw "smoke order was not created in flash mode"
    }
    if ($order.status -ne "queued") {
      throw "smoke order status mismatch: expected queued got $($order.status)"
    }
    if ($order.dockingStatus -ne "pending") {
      throw "smoke order docking status mismatch: expected pending got $($order.dockingStatus)"
    }
  } finally {
    if ($orderId) {
      try {
        Assert-ApiOk "delete smoke order" (Invoke-ApiDelete "/api/v1/orders/$orderId" $authHeaders) | Out-Null
      } catch {
        Write-Warning "Failed to delete smoke order ${orderId}: $($_.Exception.Message)"
      }
    }
    if ($connectorId) {
      try {
        Assert-ApiOk "delete smoke connector" (Invoke-ApiDelete "/api/v1/connectors/$connectorId" $authHeaders) | Out-Null
      } catch {
        Write-Warning "Failed to delete smoke connector ${connectorId}: $($_.Exception.Message)"
      }
    }
  }
}

Write-Host "==> Frontend entry"
$frontend = Invoke-WithRetry "frontend entry" {
  Invoke-WebRequest -Method Get -Uri "$FrontendUrl/" -TimeoutSec $TimeoutSec
}
if ($frontend.StatusCode -lt 200 -or $frontend.StatusCode -ge 300) {
  throw "frontend returned HTTP $($frontend.StatusCode)"
}
if ($frontend.Content -notmatch '<div id="app"></div>') {
  throw "frontend entry does not look like the Vue app shell"
}

Write-Host "==> Frontend API proxy"
$proxiedHealth = Invoke-WithRetry "frontend API proxy" {
  Invoke-RestMethod -Method Get -Uri "$FrontendUrl/api/v1/health" -TimeoutSec $TimeoutSec
}
Assert-ApiOk "frontend API proxy" $proxiedHealth | Out-Null

Write-Host "Smoke checks completed."
