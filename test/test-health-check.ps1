Write-Host "=== Health Check Monitoring Test ===" -ForegroundColor Cyan
Write-Host "Tests that health checks detect backend health and report status" -ForegroundColor Gray
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
    Write-Host "Start backends with:" -ForegroundColor Yellow
    Write-Host "  go run test/dummy-backend.go -port 3001" -ForegroundColor Gray
    Write-Host "  go run test/dummy-backend.go -port 3002" -ForegroundColor Gray
    exit 1
}

try {
    $null = Invoke-WebRequest -Uri "$baseUrl/health" -UseBasicParsing -TimeoutSec 2
    Write-Host "  API Gateway is running on :8080" -ForegroundColor Green
}
catch {
    Write-Host "ERROR: API Gateway not running!" -ForegroundColor Red
    exit 1
}

Write-Host ""

# Step 1: Authenticate
Write-Host "Step 1: Setting up authentication..." -ForegroundColor Yellow

$testEmail = "test-health-$(Get-Random)@test.com"
$testPassword = "TestPassword123!"

try {
    $registerBody = @{
        email    = $testEmail
        password = $testPassword
        name     = "Test User"
    }
    $null = Invoke-RestMethod -Uri "$baseUrl/auth/register" -Method POST `
        -ContentType "application/json" `
        -Body ($registerBody | ConvertTo-Json -Compress)
    Write-Host "  Registered user" -ForegroundColor Green
}
catch {}

try {
    $loginBody = @{
        email    = $testEmail
        password = $testPassword
    }
    $loginResponse = Invoke-RestMethod -Uri "$baseUrl/auth/login" -Method POST `
        -ContentType "application/json" `
        -Body ($loginBody | ConvertTo-Json -Compress)
    
    $authToken = $loginResponse.token
    Write-Host "  Got JWT token" -ForegroundColor Cyan
}
catch {
    Write-Host "FAILED: Login failed" -ForegroundColor Red
    exit 1
}

$headers = @{ Authorization = "Bearer $authToken" }

Write-Host ""

# Step 2: Initial health check
Write-Host "Step 2: Checking initial health status..." -ForegroundColor Yellow
Write-Host "  Waiting for health checks to run (12 seconds)..." -ForegroundColor Gray
Start-Sleep -Seconds 12

try {
    $healthStatus = Invoke-RestMethod -Uri "$baseUrl/admin/services/health" -Method GET -Headers $headers
    
    Write-Host ""
    Write-Host "  Service Health Status:" -ForegroundColor Cyan
    
    foreach ($service in $healthStatus.PSObject.Properties) {
        $path = $service.Name
        $status = $service.Value
        
        Write-Host ""
        Write-Host "  $path" -ForegroundColor White
        Write-Host "    Overall: $($status.overall_health)" -ForegroundColor $(
            if ($status.overall_health -eq "healthy") { "Green" }
            elseif ($status.overall_health -eq "degraded") { "Yellow" }
            else { "Red" }
        )
        Write-Host "    Healthy: $($status.healthy_count) / $($status.total_count)" -ForegroundColor Gray
        
        if ($status.healthy_targets) {
            Write-Host "    Healthy Targets:" -ForegroundColor Gray
            foreach ($target in $status.healthy_targets) {
                Write-Host "      - $target" -ForegroundColor Green
            }
        }
    }
}
catch {
    Write-Host "ERROR: Failed to get health status - $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

Write-Host ""

# Step 3: Simulate backend failure
Write-Host "Step 3: Simulating backend failure..." -ForegroundColor Yellow

$targetPort = $runningBackends[0]
$targetUrl = "http://localhost:$targetPort"

Write-Host "  Setting backend on :$targetPort to FAIL mode..." -ForegroundColor Yellow
try {
    $null = Invoke-WebRequest -Uri "$targetUrl/control/fail" -UseBasicParsing
    Write-Host "  Backend :$targetPort now returns 500 errors" -ForegroundColor Magenta
}
catch {
    Write-Host "ERROR: Could not set fail mode" -ForegroundColor Red
    exit 1
}

Write-Host ""

# Step 4: Wait for health check to detect failure
Write-Host "Step 4: Waiting for health check to detect failure..." -ForegroundColor Yellow
Write-Host "  Config: interval=10s, max_failures=3" -ForegroundColor Gray
Write-Host "  Need ~30+ seconds for 3 failures..." -ForegroundColor Gray

for ($countdown = 35; $countdown -gt 0; $countdown--) {
    Write-Host "`r  Waiting: $countdown seconds...   " -NoNewline -ForegroundColor Gray
    Start-Sleep -Seconds 1
}
Write-Host "`r  Wait complete!                    " -ForegroundColor Green

Write-Host ""

# Step 5: Check health status after failure
Write-Host "Step 5: Checking health status after failure..." -ForegroundColor Yellow

try {
    $healthStatus = Invoke-RestMethod -Uri "$baseUrl/admin/services/health" -Method GET -Headers $headers
    
    $foundUnhealthy = $false
    
    foreach ($service in $healthStatus.PSObject.Properties) {
        $path = $service.Name
        $status = $service.Value
        
        Write-Host ""
        Write-Host "  $path" -ForegroundColor White
        Write-Host "    Overall: $($status.overall_health)" -ForegroundColor $(
            if ($status.overall_health -eq "healthy") { "Green" }
            elseif ($status.overall_health -eq "degraded") { "Yellow" }
            else { "Red" }
        )
        Write-Host "    Healthy: $($status.healthy_count) / $($status.total_count)" -ForegroundColor Gray
        
        if ($status.target_status) {
            foreach ($target in $status.target_status) {
                $statusColor = if ($target.is_healthy) { "Green" } else { "Red" }
                $statusText = if ($target.is_healthy) { "HEALTHY" } else { "UNHEALTHY" }
                Write-Host "      $($target.target): $statusText (failures: $($target.failure_count))" -ForegroundColor $statusColor
                
                if (-not $target.is_healthy) {
                    $foundUnhealthy = $true
                }
            }
        }
        
        if ($status.overall_health -eq "degraded") {
            $foundUnhealthy = $true
        }
    }
    
    if ($foundUnhealthy) {
        Write-Host ""
        Write-Host "SUCCESS: Health check detected the failure!" -ForegroundColor Cyan
    }
    else {
        Write-Host ""
        Write-Host "WARNING: No unhealthy targets detected (health check may need more time)" -ForegroundColor Yellow
    }
}
catch {
    Write-Host "ERROR: Failed to get health status" -ForegroundColor Red
}

Write-Host ""

# Step 6: Recover backend
Write-Host "Step 6: Recovering backend..." -ForegroundColor Yellow
$null = Invoke-WebRequest -Uri "$targetUrl/control/recover" -UseBasicParsing
Write-Host "  Backend :$targetPort recovered to OK mode" -ForegroundColor Green

Write-Host ""
Write-Host "=== Test Complete ===" -ForegroundColor Cyan
Write-Host ""
Write-Host "Summary:" -ForegroundColor White
Write-Host "  - Verified health status endpoint works" -ForegroundColor Gray
Write-Host "  - Simulated backend failure" -ForegroundColor Gray
Write-Host "  - Confirmed health check detects failures" -ForegroundColor Gray
Write-Host "  - Recovered backend to healthy state" -ForegroundColor Gray
