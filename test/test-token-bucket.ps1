Write-Host "=== Token Bucket Rate Limiter Test ===" -ForegroundColor Cyan

$apiKey = "test-$(Get-Random)"

# Test 1: Burst capacity
Write-Host ""
Write-Host "Test 1 Burst Capacity - 60 tokens" -ForegroundColor Yellow
$burst = 0
1..65 | ForEach-Object {
    $response = curl.exe -s -w "`nHTTP_CODE:%{http_code}" -H "X-API-Key: $apiKey" http://localhost:8080/health
    if ($response -notmatch "429") { $burst++ }
}
Write-Host "Burst: $burst/65 succeeded (expected: 60)" -ForegroundColor Green

# Test 2: Refill after 3 seconds
Write-Host ""
Write-Host "Test 2 Refill Test - 10 tokens per sec" -ForegroundColor Yellow
Write-Host "Waiting 3 seconds..."
Start-Sleep -Seconds 3

$refilled = 0
1..35 | ForEach-Object {
    $response = curl.exe -s -w "`nHTTP_CODE:%{http_code}" -H "X-API-Key: $apiKey" http://localhost:8080/health
    if ($response -notmatch "429") { $refilled++ }
}
Write-Host "After 3s: $refilled/35 succeeded (expected: ~30)" -ForegroundColor Green

# Test 3: Gradual consumption
Write-Host ""
Write-Host "Test 3 Gradual Refill - 1 req per sec" -ForegroundColor Yellow
$gradual = 0
1..5 | ForEach-Object {
    Start-Sleep -Seconds 1
    $response = curl.exe -s -w "`nHTTP_CODE:%{http_code}" -H "X-API-Key: $apiKey" http://localhost:8080/health
    if ($response -notmatch "429") { 
        $gradual++
        Write-Host "  Second $_ : OK" -ForegroundColor Green
    }
    else {
        Write-Host "  Second $_ : DENIED" -ForegroundColor Red
    }
}
Write-Host "Gradual: $gradual/5 succeeded (expected: 5)" -ForegroundColor Green

Write-Host "`n=== All Tests Complete ===" -ForegroundColor Cyan