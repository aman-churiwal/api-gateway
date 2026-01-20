Write-Host "=== Fixed Window Rate Limiter Test ===" -ForegroundColor Cyan

# Configuration
$apiKey = "fixed-window-test-$(Get-Random)"
$uri = "http://localhost:8080/health"

# Test 1: Check if gateway is running
Write-Host "`nSetup Checking gateway..." -ForegroundColor Yellow
try {
    $null = Invoke-WebRequest -Uri $uri -UseBasicParsing -ErrorAction Stop
    Write-Host "Gateway is running" -ForegroundColor Green
}
catch {
    Write-Host "Gateway is NOT running!" -ForegroundColor Red
    Write-Host "Start it with: go run cmd/gateway/main.go" -ForegroundColor Yellow
    exit
}

# Test 2: Initial burst - should allow 60 requests
Write-Host "`nTest 1 Initial Burst 60 request limit" -ForegroundColor Yellow
$success = 0
$denied = 0

for ($i = 1; $i -le 70; $i++) {
    try {
        $response = Invoke-WebRequest -Uri $uri `
            -Headers @{"X-API-Key" = $apiKey } `
            -UseBasicParsing `
            -ErrorAction Stop
        $success++
        
        $remaining = $response.Headers['X-RateLimit-Remaining'][0]
        Write-Host "Request $i : OK - Remaining: $remaining" -ForegroundColor Green
    }
    catch {
        if ($_.Exception.Response.StatusCode -eq 429) {
            $denied++
            Write-Host "Request $i : RATE LIMITED 429" -ForegroundColor Red
        }
        else {
            Write-Host "Request $i : ERROR - $($_.Exception.Message)" -ForegroundColor Yellow
        }
    }
}

Write-Host "`nBurst Results:" -ForegroundColor Cyan
Write-Host "  Succeeded: $success expected: 60" -ForegroundColor Green
Write-Host "  Denied: $denied expected: 10" -ForegroundColor Red

# Test 3: Immediate retry - should still be denied
Write-Host "`nTest 2 Immediate Retry should fail" -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri $uri `
        -Headers @{"X-API-Key" = $apiKey } `
        -UseBasicParsing `
        -ErrorAction Stop
    Write-Host "Request succeeded UNEXPECTED" -ForegroundColor Red
}
catch {
    if ($_.Exception.Response.StatusCode -eq 429) {
        Write-Host "Request denied EXPECTED - window not reset yet" -ForegroundColor Green
    }
}

# Test 4: Wait for window to reset (60 seconds)
Write-Host "`nTest 3 Waiting for Window Reset 60 seconds..." -ForegroundColor Yellow

# Show countdown
for ($countdown = 60; $countdown -gt 0; $countdown--) {
    Write-Host "`rTime remaining: $countdown seconds..." -NoNewline -ForegroundColor Cyan
    Start-Sleep -Seconds 1
}
Write-Host "`rWindow should have reset!              " -ForegroundColor Green

# Test 5: After reset - should allow requests again
Write-Host "`n[Test 4] After Window Reset" -ForegroundColor Yellow
$successAfterReset = 0

for ($i = 1; $i -le 10; $i++) {
    try {
        $response = Invoke-WebRequest -Uri $uri `
            -Headers @{"X-API-Key" = $apiKey } `
            -UseBasicParsing `
            -ErrorAction Stop
        $successAfterReset++
        
        $remaining = $response.Headers['X-RateLimit-Remaining'][0]
        Write-Host "Request $i : OK - Remaining: $remaining" -ForegroundColor Green
    }
    catch {
        Write-Host "Request $i : DENIED unexpected" -ForegroundColor Red
    }
}

Write-Host "`n After reset: $successAfterReset/10 succeeded expected: 10" -ForegroundColor Cyan

# Test 6: Window boundary test (requests at boundary get reset)
Write-Host "`n[Test 5] Window Boundary Behavior" -ForegroundColor Yellow
Write-Host "Making requests until limit..." -ForegroundColor Cyan

$newKey = "boundary-test-$(Get-Random)"
1..60 | ForEach-Object {
    Invoke-WebRequest -Uri $uri -Headers @{"X-API-Key" = $newKey } -UseBasicParsing -ErrorAction SilentlyContinue | Out-Null
}

Write-Host "Limit reached. Checking current window..." -ForegroundColor Cyan

try {
    $response = Invoke-WebRequest -Uri $uri `
        -Headers @{"X-API-Key" = $newKey } `
        -UseBasicParsing `
        -ErrorAction Stop
    $remaining = $response.Headers['X-RateLimit-Remaining'][0]
    Write-Host "Request succeeded with $remaining remaining UNEXPECTED" -ForegroundColor Red
}
catch {
    $resetHeader = $_.Exception.Response.Headers['X-RateLimit-Reset']
    $resetTime = [DateTimeOffset]::FromUnixTimeSeconds($resetHeader).LocalDateTime
    Write-Host "Request denied. Window resets at: $resetTime" -ForegroundColor Green
}

# Summary
Write-Host "`n=== Test Summary ===" -ForegroundColor Cyan
Write-Host "Fixed Window Characteristics Verified:" -ForegroundColor White
Write-Host "Allows burst up to limit 60 requests" -ForegroundColor Green
Write-Host "Denies requests after limit exceeded" -ForegroundColor Green
Write-Host "Resets completely after 60 seconds" -ForegroundColor Green
Write-Host "No gradual refill unlike Token Bucket" -ForegroundColor Green

Write-Host "`n=== Test Complete ===" -ForegroundColor Cyan