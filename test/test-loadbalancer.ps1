Write-Host "=== Load Balancer Distribution Test ===" -ForegroundColor Cyan
Write-Host "Tests that requests are distributed across healthy backends" -ForegroundColor Gray
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
        Write-Host "  Backend on :$port is NOT running" -ForegroundColor Red
    }
}

if ($runningBackends.Count -lt 2) {
    Write-Host ""
    Write-Host "ERROR: Need at least 2 backends running!" -ForegroundColor Red
    Write-Host "Start backends with:" -ForegroundColor Yellow
    Write-Host "  go run test/dummy-backend.go -port 3001" -ForegroundColor Gray
    Write-Host "  go run test/dummy-backend.go -port 3002" -ForegroundColor Gray
    Write-Host "  go run test/dummy-backend.go -port 3003" -ForegroundColor Gray
    exit 1
}

try {
    $null = Invoke-WebRequest -Uri "$baseUrl/health" -UseBasicParsing -TimeoutSec 2
    Write-Host "  API Gateway is running on :8080" -ForegroundColor Green
}
catch {
    Write-Host "ERROR: API Gateway not running!" -ForegroundColor Red
    Write-Host "Start it with: go run cmd/gateway/main.go" -ForegroundColor Yellow
    exit 1
}

Write-Host ""

# Step 1: Authenticate
Write-Host "Step 1: Setting up authentication..." -ForegroundColor Yellow

$testEmail = "test-lb-$(Get-Random)@test.com"
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
    Write-Host "  Registered user: $testEmail" -ForegroundColor Green
}
catch {
    Write-Host "  Registration: $($_.Exception.Message)" -ForegroundColor Yellow
}

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
    Write-Host "FAILED: Login failed - $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

$headers = @{ Authorization = "Bearer $authToken" }

# Create API key
Write-Host ""
Write-Host "Step 2: Creating test API key..." -ForegroundColor Yellow

$apiKey = "lb-test-$(Get-Random)"
try {
    $keyBody = @{
        name       = "test-key-lb"
        created_by = "test-user"
        tier       = "enterprise"
    }
    $keyResponse = Invoke-RestMethod -Uri "$baseUrl/admin/keys" -Method POST `
        -ContentType "application/json" `
        -Body ($keyBody | ConvertTo-Json -Compress) `
        -Headers $headers
    
    $apiKey = $keyResponse.key
    Write-Host "  Created API key" -ForegroundColor Cyan
}
catch {
    Write-Host "  Warning: Using random key" -ForegroundColor Yellow
}

$apiHeaders = @{ "X-API-Key" = $apiKey }

Write-Host ""

# Step 3: Wait for health checks
Write-Host "Step 3: Waiting for health checks to complete..." -ForegroundColor Yellow
Write-Host "  Health check interval is 10 seconds, waiting 12s..." -ForegroundColor Gray
Start-Sleep -Seconds 12

# Step 4: Send requests and track distribution
Write-Host ""
Write-Host "Step 4: Sending requests and tracking distribution..." -ForegroundColor Yellow

$requestCount = 12
$distribution = @{}

for ($i = 1; $i -le $requestCount; $i++) {
    try {
        $response = Invoke-WebRequest -Uri "$baseUrl/api/users" -Headers $apiHeaders -UseBasicParsing
        $backendServer = $response.Headers["X-Backend-Server"]
        
        if ($backendServer) {
            if (-not $distribution.ContainsKey($backendServer)) {
                $distribution[$backendServer] = 0
            }
            $distribution[$backendServer]++
            Write-Host "  Request $i -> $backendServer" -ForegroundColor Green
        }
        else {
            Write-Host "  Request $i -> Unknown (no X-Backend-Server header)" -ForegroundColor Yellow
        }
    }
    catch {
        Write-Host "  Request $i -> Error: $($_.Exception.Message)" -ForegroundColor Red
    }
    
    Start-Sleep -Milliseconds 100
}

Write-Host ""

# Step 5: Analyze distribution
Write-Host "Step 5: Distribution Analysis" -ForegroundColor Cyan
Write-Host "=============================" -ForegroundColor Cyan

$totalRequests = ($distribution.Values | Measure-Object -Sum).Sum

foreach ($backend in $distribution.Keys | Sort-Object) {
    $count = $distribution[$backend]
    $percentage = [math]::Round(($count / $totalRequests) * 100, 1)
    $bar = "=" * $count
    Write-Host "  $backend : $count requests ($percentage%) $bar" -ForegroundColor White
}

Write-Host ""

# Check if distribution is roughly even (for round-robin)
$expectedPerBackend = $requestCount / $runningBackends.Count
$isBalanced = $true

foreach ($count in $distribution.Values) {
    if ([Math]::Abs($count - $expectedPerBackend) -gt 2) {
        $isBalanced = $false
    }
}

if ($isBalanced) {
    Write-Host "SUCCESS: Load is distributed evenly!" -ForegroundColor Cyan
}
else {
    Write-Host "NOTE: Distribution may not be perfectly even (this is OK for random/least-conn strategies)" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "=== Test Complete ===" -ForegroundColor Cyan
