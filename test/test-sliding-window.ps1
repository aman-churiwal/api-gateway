Write-Host "=== Sliding Window Rate Limiter Test ===" -ForegroundColor Cyan

# Configuration
$apiKey = "sliding-window-test-$(Get-Random)"
$uri = "http://localhost:8080/health"

# Test 1: Check if gateway is running
Write-Host "`nSetup: Checking gateway..." -ForegroundColor Yellow
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
Write-Host "`n[Test 1] Initial Burst (60 request limit)" -ForegroundColor Yellow
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
        if ($i -le 5 -or $i -ge 58) {
            Write-Host "  Request $i : OK - Remaining: $remaining" -ForegroundColor Green
        }
    }
    catch {
        if ($_.Exception.Response.StatusCode -eq 429) {
            $denied++
            if ($denied -le 5) {
                Write-Host "  Request $i : RATE LIMITED (429)" -ForegroundColor Red
            }
        }
        else {
            Write-Host "  Request $i : ERROR - $($_.Exception.Message)" -ForegroundColor Yellow
        }
    }
}

Write-Host "`nBurst Results:" -ForegroundColor Cyan
Write-Host "  Succeeded: $success (expected: 60)" -ForegroundColor $(if ($success -eq 60) { "Green" } else { "Yellow" })
Write-Host "  Denied: $denied (expected: 10)" -ForegroundColor $(if ($denied -eq 10) { "Green" } else { "Yellow" })

# Test 3: Sliding window behavior - wait partially and test
Write-Host "`n[Test 2] Sliding Window Behavior" -ForegroundColor Yellow
Write-Host "Unlike fixed window, sliding window allows requests as old ones expire" -ForegroundColor Cyan
Write-Host "Waiting 10 seconds for some requests to expire..." -ForegroundColor Cyan

for ($countdown = 10; $countdown -gt 0; $countdown--) {
    Write-Host "`r  Time remaining: $countdown seconds..." -NoNewline -ForegroundColor Cyan
    Start-Sleep -Seconds 1
}
Write-Host "`r  Window partially expired!              " -ForegroundColor Green

$successAfterWait = 0
for ($i = 1; $i -le 15; $i++) {
    try {
        $response = Invoke-WebRequest -Uri $uri `
            -Headers @{"X-API-Key" = $apiKey } `
            -UseBasicParsing `
            -ErrorAction Stop
        $successAfterWait++
        
        $remaining = $response.Headers['X-RateLimit-Remaining'][0]
        Write-Host "  Request $i : OK - Remaining: $remaining" -ForegroundColor Green
    }
    catch {
        if ($_.Exception.Response.StatusCode -eq 429) {
            Write-Host "  Request $i : RATE LIMITED" -ForegroundColor Red
        }
    }
}

Write-Host "`n  After 10s wait: $successAfterWait/15 succeeded" -ForegroundColor Cyan
Write-Host "  (Sliding window should allow ~10 new requests as old ones expired)" -ForegroundColor Cyan

# Test 4: Continuous gradual requests
Write-Host "`n[Test 3] Gradual Request Rate (1 per second)" -ForegroundColor Yellow
Write-Host "Making 1 request per second - should all succeed as old requests expire" -ForegroundColor Cyan

$newKey = "gradual-test-$(Get-Random)"
# First use up some quota
Write-Host "  Using up 30 requests first..." -ForegroundColor Cyan
1..30 | ForEach-Object {
    $null = Invoke-WebRequest -Uri $uri -Headers @{"X-API-Key" = $newKey } -UseBasicParsing -ErrorAction SilentlyContinue
}

$gradualSuccess = 0
for ($i = 1; $i -le 5; $i++) {
    Start-Sleep -Seconds 1
    try {
        $response = Invoke-WebRequest -Uri $uri `
            -Headers @{"X-API-Key" = $newKey } `
            -UseBasicParsing `
            -ErrorAction Stop
        $gradualSuccess++
        
        $remaining = $response.Headers['X-RateLimit-Remaining'][0]
        Write-Host "  Second $i : OK - Remaining: $remaining" -ForegroundColor Green
    }
    catch {
        Write-Host "  Second $i : DENIED" -ForegroundColor Red
    }
}

Write-Host "`n  Gradual: $gradualSuccess/5 succeeded (expected: 5)" -ForegroundColor Cyan

# Summary
Write-Host "`n=== Test Summary ===" -ForegroundColor Cyan
Write-Host "Sliding Window Characteristics:" -ForegroundColor White
Write-Host "  [+] Smoother rate limiting than fixed window" -ForegroundColor Green
Write-Host "  [+] Requests expire individually (not all at once)" -ForegroundColor Green
Write-Host "  [+] More accurate request rate over time" -ForegroundColor Green
Write-Host "  [-] More memory usage (stores each request timestamp)" -ForegroundColor Yellow

Write-Host "`n=== Test Complete ===" -ForegroundColor Cyan
