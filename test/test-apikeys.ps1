Write-Host "=== API Key Database Test ===" -ForegroundColor Cyan
Write-Host "This test creates a temporary key, tests it, then cleans up." -ForegroundColor Gray
Write-Host ""

$baseUrl = "http://localhost:8080"
$createdKeyId = $null
$createdKey = $null

# Test 1: Create API Key
Write-Host "Test 1: Create API Key" -ForegroundColor Yellow
$keyName = "test-key-$(Get-Random)"

$body = @{
    name       = $keyName
    created_by = "test-user"
    tier       = "basic"
}

try {
    $createResponse = Invoke-RestMethod -Uri "$baseUrl/admin/keys" -Method POST `
        -ContentType "application/json" `
        -Body ($body | ConvertTo-Json -Compress)
    
    Write-Host "Response: $($createResponse | ConvertTo-Json -Compress)" -ForegroundColor Green
    
    if ($createResponse.key) {
        $createdKey = $createResponse.key
        Write-Host "SUCCESS: Created Key: $createdKey" -ForegroundColor Cyan
    }
}
catch {
    Write-Host "FAILED: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host ""

# Test 2: List All API Keys and find our key
Write-Host "Test 2: List API Keys (verify our key exists)" -ForegroundColor Yellow
try {
    $keys = Invoke-RestMethod -Uri "$baseUrl/admin/keys" -Method GET
    Write-Host "Total Keys: $($keys.Count)" -ForegroundColor Green
    
    # Find our key by name
    foreach ($key in $keys) {
        if ($key.name -eq $keyName) {
            $createdKeyId = $key.id
            Write-Host "SUCCESS: Found our key - ID: $createdKeyId" -ForegroundColor Cyan
            break
        }
    }
    
    if (-not $createdKeyId) {
        Write-Host "WARNING: Could not find created key in list" -ForegroundColor Yellow
    }
}
catch {
    Write-Host "FAILED: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host ""

# Test 3: Use created key to make a request
Write-Host "Test 3: Use Created Key for Health Check" -ForegroundColor Yellow
if ($createdKey) {
    $healthResponse = curl.exe -s -w "`n%{http_code}" -H "X-API-Key: $createdKey" "http://localhost:8080/health"
    $lines = $healthResponse -split "`n"
    $statusCode = $lines[-1].Trim()
    $body = ($lines[0..($lines.Length - 2)] -join "`n").Trim()
    
    if ($statusCode -eq "200") {
        Write-Host "Response: $body" -ForegroundColor Green
        Write-Host "SUCCESS: Key is valid and working!" -ForegroundColor Cyan
    }
    else {
        Write-Host "FAILED: Status $statusCode - $body" -ForegroundColor Red
    }
}
else {
    Write-Host "SKIPPED: No key to test" -ForegroundColor Yellow
}

Write-Host ""

# Test 4: Delete our created key (cleanup)
Write-Host "Test 4: Delete Our Test Key (cleanup)" -ForegroundColor Yellow
if ($createdKeyId) {
    try {
        $deleteResponse = Invoke-RestMethod -Uri "$baseUrl/admin/keys/$createdKeyId" -Method DELETE
        Write-Host "Response: $($deleteResponse | ConvertTo-Json -Compress)" -ForegroundColor Green
        Write-Host "SUCCESS: Test key deleted" -ForegroundColor Cyan
    }
    catch {
        Write-Host "FAILED: $($_.Exception.Message)" -ForegroundColor Red
    }
}
else {
    Write-Host "SKIPPED: No key ID to delete" -ForegroundColor Yellow
}

Write-Host ""

# Test 5: Verify our key is deleted
Write-Host "Test 5: Verify Our Key is Deleted" -ForegroundColor Yellow
try {
    $keys2 = Invoke-RestMethod -Uri "$baseUrl/admin/keys" -Method GET
    $found = $false
    foreach ($key in $keys2) {
        if ($key.name -eq $keyName) {
            $found = $true
            break
        }
    }
    
    if (-not $found) {
        Write-Host "SUCCESS: Test key no longer exists" -ForegroundColor Cyan
    }
    else {
        Write-Host "WARNING: Test key still exists!" -ForegroundColor Yellow
    }
}
catch {
    Write-Host "FAILED: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host ""
Write-Host "=== All Database Tests Complete ===" -ForegroundColor Cyan
Write-Host "State restored - only the test key was created and deleted." -ForegroundColor Gray
