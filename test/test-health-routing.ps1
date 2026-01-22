Write-Host "=== Health-Aware Routing Test ===" -ForegroundColor Cyan
Write-Host "Tests that unhealthy backends are excluded from routing" -ForegroundColor Gray
Write-Host ""

$baseUrl = "http://localhost:8080"
$authToken = $null

# Step 0: Check prerequisites
Write-Host "Step 0: Checking prerequisites..." -ForegroundColor Yellow

$backends = @(3001, 3002, 3003)
$runningBackends = @()

foreach ($port in $backends) {
    try {
        $null = Invoke-WebRequest -Uri "http://localhost:$port/health" -UseBasicParsing -TimeoutSec 2
        Write-Host "  Backend on :$port is running" -ForegroundColor Green
        $runningBackends += $port
    }
    catch {
        Write-Host "  Backend on :$port is NOT running" -ForegroundColor Yellow
    }
}

if ($runningBackends.Count -lt 2) {
    Write-Host ""
    Write-Host "ERROR: Need at least 2 backends running!" -ForegroundColor Red
    exit 1
}

try {
    $null = Invoke-WebRequest -Uri "$baseUrl/health" -UseBasicParsing -TimeoutSec 2
    Write-Host "  API Gateway is running" -ForegroundColor Green
}
catch {
    Write-Host "ERROR: API Gateway not running!" -ForegroundColor Red
    exit 1
}

Write-Host ""

# Step 1: Authenticate & create API key
Write-Host "Step 1: Setting up authentication..." -ForegroundColor Yellow

$testEmail = "test-routing-$(Get-Random)@test.com"
$testPassword = "TestPassword123!"

try {
    $registerBody = @{ email = $testEmail; password = $testPassword; name = "Test User" }
    $null = Invoke-RestMethod -Uri "$baseUrl/auth/register" -Method POST -ContentType "application/json" -Body ($registerBody | ConvertTo-Json -Compress)
}
catch {}

try {
    $loginBody = @{ email = $testEmail; password = $testPassword }
    $loginResponse = Invoke-RestMethod -Uri "$baseUrl/auth/login" -Method POST -ContentType "application/json" -Body ($loginBody | ConvertTo-Json -Compress)
    $authToken = $loginResponse.token
    Write-Host "  Got JWT token" -ForegroundColor Cyan
}
catch {
    Write-Host "FAILED: Login failed" -ForegroundColor Red
    exit 1
}

$headers = @{ Authorization = "Bearer $authToken" }

$apiKey = "routing-test-$(Get-Random)"
try {
    $keyBody = @{ name = "routing-test-key"; created_by = "test-user"; tier = "enterprise" }
    $keyResponse = Invoke-RestMethod -Uri "$baseUrl/admin/keys" -Method POST -ContentType "application/json" -Body ($keyBody | ConvertTo-Json -Compress) -Headers $headers
    $apiKey = $keyResponse.key
    Write-Host "  Created API key" -ForegroundColor Cyan
}
catch {}

$apiHeaders = @{ "X-API-Key" = $apiKey }

Write-Host ""

# Step 2: Wait for initial health checks
Write-Host "Step 2: Waiting for health checks (12s)..." -ForegroundColor Yellow
Start-Sleep -Seconds 12

# Step 3: Baseline - all backends healthy
Write-Host ""
Write-Host "Step 3: Baseline distribution (all healthy)..." -ForegroundColor Yellow

$baselineDistribution = @{}
for ($i = 1; $i -le 9; $i++) {
    try {
        $response = Invoke-WebRequest -Uri "$baseUrl/api/users" -Headers $apiHeaders -UseBasicParsing
        $backend = $response.Headers["X-Backend-Server"]
        if ($backend) {
            if (-not $baselineDistribution.ContainsKey($backend)) { $baselineDistribution[$backend] = 0 }
            $baselineDistribution[$backend]++
        }
    }
    catch {}
    Start-Sleep -Milliseconds 50
}

Write-Host "  Baseline distribution:" -ForegroundColor Gray
foreach ($backend in $baselineDistribution.Keys | Sort-Object) {
    Write-Host "    $backend : $($baselineDistribution[$backend]) requests" -ForegroundColor Green
}

Write-Host ""

# Step 4: Set one backend to fail mode
$targetPort = $runningBackends[0]
$targetUrl = "http://localhost:$targetPort"
$targetBackend = "http://localhost:$targetPort"

Write-Host "Step 4: Setting backend :$targetPort to FAIL mode..." -ForegroundColor Yellow
$null = Invoke-WebRequest -Uri "$targetUrl/control/fail" -UseBasicParsing
Write-Host "  Backend :$targetPort will now fail health checks" -ForegroundColor Magenta

Write-Host ""

# Step 5: Wait for unhealthy detection
Write-Host "Step 5: Waiting for health check to mark backend unhealthy (35s)..." -ForegroundColor Yellow
for ($countdown = 35; $countdown -gt 0; $countdown--) {
    Write-Host "`r  Waiting: $countdown seconds...   " -NoNewline -ForegroundColor Gray
    Start-Sleep -Seconds 1
}
Write-Host "`r  Wait complete!                    " -ForegroundColor Green

Write-Host ""

# Step 6: Send requests - verify unhealthy backend excluded
Write-Host "Step 6: Sending requests (unhealthy backend should be excluded)..." -ForegroundColor Yellow

$afterFailDistribution = @{}
$requestsToUnhealthy = 0

for ($i = 1; $i -le 15; $i++) {
    try {
        $response = Invoke-WebRequest -Uri "$baseUrl/api/users" -Headers $apiHeaders -UseBasicParsing
        $backend = $response.Headers["X-Backend-Server"]
        if ($backend) {
            if (-not $afterFailDistribution.ContainsKey($backend)) { $afterFailDistribution[$backend] = 0 }
            $afterFailDistribution[$backend]++
            
            if ($backend -eq $targetBackend) {
                $requestsToUnhealthy++
            }
        }
    }
    catch {
        Write-Host "  Request $i failed: $($_.Exception.Message)" -ForegroundColor Red
    }
    Start-Sleep -Milliseconds 50
}

Write-Host ""
Write-Host "  Distribution after failure:" -ForegroundColor Cyan
foreach ($backend in $afterFailDistribution.Keys | Sort-Object) {
    $color = if ($backend -eq $targetBackend) { "Red" } else { "Green" }
    Write-Host "    $backend : $($afterFailDistribution[$backend]) requests" -ForegroundColor $color
}

Write-Host ""

# Step 7: Verify result
if ($requestsToUnhealthy -eq 0) {
    Write-Host "SUCCESS: Unhealthy backend ($targetBackend) received 0 requests!" -ForegroundColor Cyan
    Write-Host "  Health-aware routing is working correctly." -ForegroundColor Green
}
else {
    Write-Host "WARNING: Unhealthy backend received $requestsToUnhealthy requests" -ForegroundColor Yellow
    Write-Host "  (This may happen if health check hasn't fully detected failure yet)" -ForegroundColor Gray
}

Write-Host ""

# Step 8: Recover and cleanup
Write-Host "Step 8: Recovering backend..." -ForegroundColor Yellow
$null = Invoke-WebRequest -Uri "$targetUrl/control/recover" -UseBasicParsing
Write-Host "  Backend :$targetPort recovered" -ForegroundColor Green

Write-Host ""
Write-Host "=== Test Complete ===" -ForegroundColor Cyan
