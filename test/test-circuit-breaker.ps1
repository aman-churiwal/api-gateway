Write-Host "=== Circuit Breaker Full State Test ===" -ForegroundColor Cyan
Write-Host "Tests all circuit breaker states: closed -> open -> half-open -> closed" -ForegroundColor Gray
Write-Host ""

$baseUrl = "http://localhost:8080"
$backendUrl = "http://localhost:3001"
$authToken = $null

# Step 0: Check if dummy backend is running
Write-Host "Step 0: Checking prerequisites..." -ForegroundColor Yellow

try {
    $null = Invoke-WebRequest -Uri "$backendUrl/control/status" -UseBasicParsing -TimeoutSec 2
    Write-Host "Dummy backend is running on :3001" -ForegroundColor Green
}
catch {
    Write-Host "ERROR: Dummy backend not running!" -ForegroundColor Red
    Write-Host "Start it with: go run test/dummy-backend.go" -ForegroundColor Yellow
    exit 1
}

try {
    $null = Invoke-WebRequest -Uri "$baseUrl/health" -UseBasicParsing -TimeoutSec 2
    Write-Host "API Gateway is running on :8080" -ForegroundColor Green
}
catch {
    Write-Host "ERROR: API Gateway not running!" -ForegroundColor Red
    Write-Host "Start it with: go run cmd/gateway/main.go" -ForegroundColor Yellow
    exit 1
}

Write-Host ""

# Step 1: Authenticate
Write-Host "Step 1: Setting up authentication..." -ForegroundColor Yellow

$testEmail = "test-cb-full-$(Get-Random)@test.com"
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
    Write-Host "Registered user: $testEmail" -ForegroundColor Green
}
catch {
    Write-Host "Registration: $($_.Exception.Message)" -ForegroundColor Yellow
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
    Write-Host "SUCCESS: Got JWT token" -ForegroundColor Cyan
}
catch {
    Write-Host "FAILED: Login failed - $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

$headers = @{
    Authorization = "Bearer $authToken"
}

$apiKey = "cb-full-test-$(Get-Random)"
$apiHeaders = @{ "X-API-Key" = $apiKey }

Write-Host ""

# Step 2: Create API key for testing (using admin endpoint)
Write-Host "Step 2: Creating test API key..." -ForegroundColor Yellow

try {
    $keyBody = @{
        name       = "test-key-$apiKey"
        created_by = "test-user"
        tier       = "basic"
    }
    $keyResponse = Invoke-RestMethod -Uri "$baseUrl/admin/keys" -Method POST `
        -ContentType "application/json" `
        -Body ($keyBody | ConvertTo-Json -Compress) `
        -Headers $headers
    
    $apiKey = $keyResponse.key
    $apiHeaders = @{ "X-API-Key" = $apiKey }
    Write-Host "SUCCESS: Created API key" -ForegroundColor Cyan
}
catch {
    Write-Host "WARNING: Could not create API key, using random key" -ForegroundColor Yellow
}

Write-Host ""

# Step 3: Ensure backend is in OK mode
Write-Host "Step 3: Ensuring backend is healthy..." -ForegroundColor Yellow
$null = Invoke-WebRequest -Uri "$backendUrl/control/recover" -UseBasicParsing
Write-Host "Backend set to OK mode" -ForegroundColor Green

# Reset circuit breaker
try {
    $null = Invoke-RestMethod -Uri "$baseUrl/admin/circuit-breakers/api/users" -Method POST -Headers $headers
}
catch {}

Write-Host ""

# Test A: Verify initial CLOSED state
Write-Host "=== Test A: Verify CLOSED State ===" -ForegroundColor Cyan
$status = Invoke-RestMethod -Uri "$baseUrl/admin/circuit-breakers" -Method GET -Headers $headers
$usersState = $status."/api/users".state
Write-Host "Circuit breaker state: $usersState" -ForegroundColor $(if ($usersState -eq "closed") { "Green" } else { "Red" })

if ($usersState -ne "closed") {
    Write-Host "WARNING: Circuit not closed, resetting..." -ForegroundColor Yellow
    $null = Invoke-RestMethod -Uri "$baseUrl/admin/circuit-breakers/api/users" -Method POST -Headers $headers
}

# Make a successful request
try {
    $response = Invoke-WebRequest -Uri "$baseUrl/api/users" -Headers $apiHeaders -UseBasicParsing
    Write-Host "Request succeeded (Status: $($response.StatusCode))" -ForegroundColor Green
}
catch {
    Write-Host "Request failed: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host ""

# Test B: Trigger OPEN state by causing failures
Write-Host "=== Test B: Trigger OPEN State ===" -ForegroundColor Cyan
Write-Host "Setting backend to FAIL mode..." -ForegroundColor Yellow

$null = Invoke-WebRequest -Uri "$backendUrl/control/fail" -UseBasicParsing
Write-Host "Backend now returns 500 errors" -ForegroundColor Yellow

Write-Host "Sending requests to trigger circuit breaker (max_failures = 5)..." -ForegroundColor Yellow

for ($i = 1; $i -le 7; $i++) {
    try {
        $response = Invoke-WebRequest -Uri "$baseUrl/api/users" -Headers $apiHeaders -UseBasicParsing -ErrorAction Stop
        Write-Host "Request $i : SUCCESS (Status: $($response.StatusCode))" -ForegroundColor Green
    }
    catch {
        $statusCode = $_.Exception.Response.StatusCode.value__
        if ($statusCode -eq 503) {
            Write-Host "Request $i : CIRCUIT OPEN (503)" -ForegroundColor Magenta
        }
        elseif ($statusCode -eq 502) {
            Write-Host "Request $i : Backend error forwarded (502)" -ForegroundColor Yellow
        }
        else {
            Write-Host "Request $i : Error $statusCode" -ForegroundColor Red
        }
    }
    Start-Sleep -Milliseconds 100
}

# Check circuit state
$status = Invoke-RestMethod -Uri "$baseUrl/admin/circuit-breakers" -Method GET -Headers $headers
$usersState = $status."/api/users".state
$failures = $status."/api/users".failure_count

Write-Host ""
Write-Host "Circuit breaker state: $usersState (failures: $failures)" -ForegroundColor $(if ($usersState -eq "open") { "Magenta" } else { "Yellow" })

if ($usersState -eq "open") {
    Write-Host "SUCCESS: Circuit breaker is OPEN!" -ForegroundColor Cyan
}
else {
    Write-Host "INFO: Circuit may not have opened (backend errors might not be 5xx)" -ForegroundColor Yellow
}

Write-Host ""

# Test C: Wait for HALF-OPEN transition
Write-Host "=== Test C: Wait for HALF-OPEN State ===" -ForegroundColor Cyan

# Config says timeout_seconds: 30 for /api/users
$timeoutSeconds = 30
Write-Host "Circuit breaker timeout is $timeoutSeconds seconds..." -ForegroundColor Yellow
Write-Host "Waiting for timeout to allow half-open transition..." -ForegroundColor Yellow

# Wait with countdown
for ($countdown = $timeoutSeconds; $countdown -gt 0; $countdown--) {
    Write-Host "`rWaiting: $countdown seconds...   " -NoNewline -ForegroundColor Gray
    Start-Sleep -Seconds 1
}
Write-Host "`rTimeout reached!                      " -ForegroundColor Green

# Recover backend before testing half-open
Write-Host "Recovering backend..." -ForegroundColor Yellow
$null = Invoke-WebRequest -Uri "$backendUrl/control/recover" -UseBasicParsing
Write-Host "Backend now returns 200 OK" -ForegroundColor Green

Write-Host ""

# Test D: Verify HALF-OPEN allows one request and closes
Write-Host "=== Test D: Test HALF-OPEN -> CLOSED Transition ===" -ForegroundColor Cyan

Write-Host "Making request (should trigger half-open test)..." -ForegroundColor Yellow

try {
    $response = Invoke-WebRequest -Uri "$baseUrl/api/users" -Headers $apiHeaders -UseBasicParsing -ErrorAction Stop
    Write-Host "Request succeeded (Status: $($response.StatusCode))" -ForegroundColor Green
}
catch {
    $statusCode = $_.Exception.Response.StatusCode.value__
    Write-Host "Request returned: $statusCode" -ForegroundColor Yellow
}

# Check final state
$status = Invoke-RestMethod -Uri "$baseUrl/admin/circuit-breakers" -Method GET -Headers $headers
$usersState = $status."/api/users".state
$failures = $status."/api/users".failure_count
$successes = $status."/api/users".success_count

Write-Host ""
Write-Host "Final circuit breaker state: $usersState" -ForegroundColor $(if ($usersState -eq "closed") { "Green" } else { "Yellow" })
Write-Host "  Failures: $failures, Successes: $successes" -ForegroundColor Gray

if ($usersState -eq "closed") {
    Write-Host "SUCCESS: Circuit breaker returned to CLOSED!" -ForegroundColor Cyan
}

Write-Host ""
Write-Host "=== Full State Test Complete ===" -ForegroundColor Cyan
Write-Host ""
Write-Host "Summary of State Transitions:" -ForegroundColor White
Write-Host "  1. CLOSED  - Normal operation, requests pass through" -ForegroundColor Green
Write-Host "  2. OPEN    - After 5 failures, requests blocked (503)" -ForegroundColor Magenta
Write-Host "  3. HALF-OPEN - After 30s timeout, allows test request" -ForegroundColor Yellow
Write-Host "  4. CLOSED  - After successful test, fully recovered" -ForegroundColor Green
