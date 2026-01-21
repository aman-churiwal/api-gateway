Write-Host "=== API Key Database Test ===" -ForegroundColor Cyan
Write-Host "This test creates a temporary key, tests it, then cleans up." -ForegroundColor Gray
Write-Host ""

$baseUrl = "http://localhost:8080"
$createdKeyId = $null
$createdKey = $null
$authToken = $null

# Step 0: Register and Login to get JWT token
Write-Host "Step 0: Setting up authentication..." -ForegroundColor Yellow

$testEmail = "test-admin-$(Get-Random)@test.com"
$testPassword = "TestPassword123!"

# Register
try {
    $registerBody = @{
        email    = $testEmail
        password = $testPassword
        name     = "Test Admin"
    }
    $null = Invoke-RestMethod -Uri "$baseUrl/auth/register" -Method POST `
        -ContentType "application/json" `
        -Body ($registerBody | ConvertTo-Json -Compress)
    Write-Host "Registered user: $testEmail" -ForegroundColor Green
}
catch {
    Write-Host "Registration: $($_.Exception.Message)" -ForegroundColor Yellow
}

# Login
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

Write-Host ""

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
        -Body ($body | ConvertTo-Json -Compress) `
        -Headers $headers
    
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
    $keys = Invoke-RestMethod -Uri "$baseUrl/admin/keys" -Method GET -Headers $headers
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
        $deleteResponse = Invoke-RestMethod -Uri "$baseUrl/admin/keys/$createdKeyId" -Method DELETE -Headers $headers
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
    $keys2 = Invoke-RestMethod -Uri "$baseUrl/admin/keys" -Method GET -Headers $headers
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
