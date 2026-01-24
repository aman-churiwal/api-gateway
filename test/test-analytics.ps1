# Test script for Analytics feature
# Tests the following endpoints:
# - GET /admin/analytics - Summary analytics
# - GET /admin/analytics/timeseries - Time series data
# - GET /admin/analytics/keys/:id - API key specific stats
# - GET /admin/logs - Request logs with filters

$BASE_URL = "http://localhost:8080"
$AUTH_TOKEN = ""

# Colors for output
function Write-Success { param($msg) Write-Host $msg -ForegroundColor Green }
function Write-Failure { param($msg) Write-Host $msg -ForegroundColor Red }
function Write-Info { param($msg) Write-Host $msg -ForegroundColor Cyan }
function Write-Title { param($msg) Write-Host "`n========================================" -ForegroundColor Yellow; Write-Host $msg -ForegroundColor Yellow; Write-Host "========================================" -ForegroundColor Yellow }

Write-Title "Analytics Feature Test Suite"

# Step 1: Register and Login to get JWT token
Write-Info "`n1. Setting up authentication..."

$random = Get-Random
$email = "analyticstest_$random@test.com"
$registerBody = @{
    name     = "Analytics Test User"
    email    = $email
    password = "TestPassword123!"
} | ConvertTo-Json

try {
    Invoke-RestMethod -Uri "$BASE_URL/auth/register" -Method POST -Body $registerBody -ContentType "application/json" -ErrorAction Stop | Out-Null
    Write-Success "   User registered successfully"
}
catch {
    Write-Info "   Registration skipped: $($_.Exception.Message)"
}

# Login
$loginBody = @{
    email    = $email
    password = "TestPassword123!"
} | ConvertTo-Json

try {
    $loginResponse = Invoke-RestMethod -Uri "$BASE_URL/auth/login" -Method POST -Body $loginBody -ContentType "application/json"
    $AUTH_TOKEN = $loginResponse.token
    Write-Success "   Authentication successful"
}
catch {
    Write-Failure "   Login failed: $($_.Exception.Message)"
    exit 1
}

$headers = @{
    "Authorization" = "Bearer $AUTH_TOKEN"
    "Content-Type"  = "application/json"
}

# Step 2: Create an API key for testing
Write-Title "2. Creating API Key for testing"

$keyName = "analytics-test-key-$random"
$apiKeyBody = @{
    name = $keyName
    tier = "basic"
} | ConvertTo-Json

$API_KEY = ""
$API_KEY_ID = ""

try {
    $apiKeyResponse = Invoke-RestMethod -Uri "$BASE_URL/admin/keys" -Method POST -Headers $headers -Body $apiKeyBody
    $API_KEY = $apiKeyResponse.key
    Write-Success "   API Key created: $API_KEY"
    
    # Get the key ID by listing keys and finding the one we just created
    Start-Sleep -Milliseconds 500
    $keysResponse = Invoke-RestMethod -Uri "$BASE_URL/admin/keys" -Method GET -Headers $headers
    foreach ($key in $keysResponse) {
        if ($key.name -eq $keyName) {
            $API_KEY_ID = $key.id
            Write-Success "   API Key ID: $API_KEY_ID"
            break
        }
    }
}
catch {
    Write-Failure "   Failed to create API key: $($_.Exception.Message)"
    # Try to list keys and use existing one
    try {
        $keysResponse = Invoke-RestMethod -Uri "$BASE_URL/admin/keys" -Method GET -Headers $headers
        if ($keysResponse.Count -gt 0) {
            $API_KEY = $keysResponse[0].key
            $API_KEY_ID = $keysResponse[0].id
            Write-Info "   Using existing API key ID: $API_KEY_ID"
        }
    }
    catch {
        Write-Failure "   Could not retrieve existing keys"
    }
}

# Step 3: Generate some traffic to create analytics data
Write-Title "3. Generating test traffic"

Write-Info "   Making requests to generate analytics data..."

$trafficHeaders = @{
    "X-API-Key" = $API_KEY
}

# Make various requests to generate traffic
for ($i = 1; $i -le 10; $i++) {
    try {
        # These will likely return errors (no backend), but will still be logged
        Invoke-WebRequest -Uri "$BASE_URL/api/users" -Method GET -Headers $trafficHeaders -TimeoutSec 2 -ErrorAction SilentlyContinue | Out-Null
        Write-Host "." -NoNewline -ForegroundColor Green
    }
    catch {
        Write-Host "." -NoNewline -ForegroundColor Yellow
    }
}

# Also hit health endpoint (should succeed)
for ($i = 1; $i -le 5; $i++) {
    try {
        Invoke-RestMethod -Uri "$BASE_URL/health" -Method GET -TimeoutSec 2 | Out-Null
        Write-Host "." -NoNewline -ForegroundColor Green
    }
    catch {
        Write-Host "." -NoNewline -ForegroundColor Yellow
    }
}

Write-Host ""
Write-Success "   Generated 15 requests for analytics"

# Wait a moment for async batch logging
Write-Info "   Waiting 6 seconds for batch logging to flush..."
Start-Sleep -Seconds 6

# Step 4: Test Analytics Summary endpoint
Write-Title "4. Testing GET /admin/analytics (Summary)"

try {
    $summaryResponse = Invoke-RestMethod -Uri "$BASE_URL/admin/analytics" -Method GET -Headers $headers
    Write-Success "   Analytics Summary Response:"
    Write-Host ($summaryResponse | ConvertTo-Json -Depth 10) -ForegroundColor White
    
    # Validate expected fields
    $expectedFields = @("total_requests", "avg_response_time_ms", "error_rate", "success_rate")
    foreach ($field in $expectedFields) {
        if ($null -ne $summaryResponse.$field) {
            Write-Success "   Field '$field' present"
        }
        else {
            Write-Failure "   Field '$field' missing"
        }
    }
}
catch {
    Write-Failure "   Failed to get analytics summary: $($_.Exception.Message)"
    if ($_.Exception.Response) {
        $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
        Write-Failure "   Response: $($reader.ReadToEnd())"
    }
}

# Step 5: Test Analytics with time range (using Unix timestamp)
Write-Title "5. Testing GET /admin/analytics with time range"

$fromTimestamp = [int][double]::Parse((Get-Date).AddHours(-1).ToUniversalTime().Subtract([datetime]'1970-01-01').TotalSeconds.ToString())
$toTimestamp = [int][double]::Parse((Get-Date).ToUniversalTime().Subtract([datetime]'1970-01-01').TotalSeconds.ToString())
$analyticsWithRangeUrl = "$BASE_URL/admin/analytics?from=$fromTimestamp" + "&" + "to=$toTimestamp"

Write-Info "   Using Unix timestamps: from=$fromTimestamp to=$toTimestamp"

try {
    $summaryWithRange = Invoke-RestMethod -Uri $analyticsWithRangeUrl -Method GET -Headers $headers
    Write-Success "   Analytics with time range:"
    Write-Host "   Total Requests: $($summaryWithRange.total_requests)" -ForegroundColor White
    Write-Host "   Avg Response Time: $($summaryWithRange.avg_response_time_ms)ms" -ForegroundColor White
    Write-Host "   Error Rate: $($summaryWithRange.error_rate)%" -ForegroundColor White
    Write-Host "   Success Rate: $($summaryWithRange.success_rate)%" -ForegroundColor White
}
catch {
    Write-Failure "   Failed: $($_.Exception.Message)"
}

# Step 6: Test Time Series endpoint
Write-Title "6. Testing GET /admin/analytics/timeseries"

try {
    $timeSeriesResponse = Invoke-RestMethod -Uri "$BASE_URL/admin/analytics/timeseries" -Method GET -Headers $headers
    Write-Success "   Time Series Response:"
    if ($timeSeriesResponse -is [array]) {
        Write-Host "   Total data points: $($timeSeriesResponse.Count)" -ForegroundColor White
        if ($timeSeriesResponse.Count -gt 0) {
            Write-Host "   First entry: $($timeSeriesResponse[0] | ConvertTo-Json -Compress)" -ForegroundColor White
        }
    }
    else {
        Write-Host ($timeSeriesResponse | ConvertTo-Json -Depth 5) -ForegroundColor White
    }
}
catch {
    Write-Failure "   Failed to get time series data: $($_.Exception.Message)"
    if ($_.Exception.Response) {
        $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
        Write-Failure "   Response: $($reader.ReadToEnd())"
    }
}

# Step 7: Test API Key Stats endpoint
Write-Title "7. Testing GET /admin/analytics/keys/:id (API Key Stats)"

if ($API_KEY_ID -ne "") {
    try {
        $keyStatsResponse = Invoke-RestMethod -Uri "$BASE_URL/admin/analytics/keys/$API_KEY_ID" -Method GET -Headers $headers
        Write-Success "   API Key Stats Response:"
        Write-Host ($keyStatsResponse | ConvertTo-Json -Depth 5) -ForegroundColor White
    }
    catch {
        Write-Failure "   Failed to get API key stats: $($_.Exception.Message)"
        if ($_.Exception.Response) {
            $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
            Write-Failure "   Response: $($reader.ReadToEnd())"
        }
    }
}
else {
    Write-Info "   Skipped - No API Key ID available"
}

# Step 8: Test Logs endpoint
Write-Title "8. Testing GET /admin/logs"

try {
    $logsResponse = Invoke-RestMethod -Uri "$BASE_URL/admin/logs" -Method GET -Headers $headers
    Write-Success "   Logs Response:"
    Write-Host "   Total logs returned: $($logsResponse.logs.Count)" -ForegroundColor White
    Write-Host "   Limit: $($logsResponse.limit)" -ForegroundColor White
    Write-Host "   Offset: $($logsResponse.offset)" -ForegroundColor White
    
    if ($logsResponse.logs.Count -gt 0) {
        Write-Host "`n   Sample log entry:" -ForegroundColor Cyan
        Write-Host ($logsResponse.logs[0] | ConvertTo-Json -Depth 3) -ForegroundColor White
    }
}
catch {
    Write-Failure "   Failed to get logs: $($_.Exception.Message)"
    if ($_.Exception.Response) {
        $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
        Write-Failure "   Response: $($reader.ReadToEnd())"
    }
}

# Step 9: Test Logs with pagination
Write-Title "9. Testing GET /admin/logs with pagination"

$logsPagedUrl = "$BASE_URL/admin/logs?limit=5" + "&" + "offset=0"

try {
    $logsPagedResponse = Invoke-RestMethod -Uri $logsPagedUrl -Method GET -Headers $headers
    Write-Success "   Paginated Logs (limit=5 offset=0):"
    Write-Host "   Logs count: $($logsPagedResponse.logs.Count)" -ForegroundColor White
}
catch {
    Write-Failure "   Failed: $($_.Exception.Message)"
}

# Step 10: Test Logs with status filter
Write-Title "10. Testing GET /admin/logs with status filter"

try {
    $logs200Response = Invoke-RestMethod -Uri "$BASE_URL/admin/logs?status=200" -Method GET -Headers $headers
    Write-Success "   Logs with status=200:"
    Write-Host "   Count: $($logs200Response.logs.Count)" -ForegroundColor White
}
catch {
    Write-Failure "   Failed: $($_.Exception.Message)"
}

try {
    $logs500Response = Invoke-RestMethod -Uri "$BASE_URL/admin/logs?status=502" -Method GET -Headers $headers
    Write-Success "   Logs with status=502:"
    Write-Host "   Count: $($logs500Response.logs.Count)" -ForegroundColor White
}
catch {
    Write-Failure "   Failed: $($_.Exception.Message)"
}

# Step 11: Test invalid API key ID
Write-Title "11. Testing error handling - Invalid API Key ID"

try {
    Invoke-RestMethod -Uri "$BASE_URL/admin/analytics/keys/invalid-uuid" -Method GET -Headers $headers
    Write-Failure "   Expected error but got success"
}
catch {
    if ($_.Exception.Response.StatusCode -eq "BadRequest") {
        Write-Success "   Correctly returned BadRequest for invalid UUID"
    }
    else {
        Write-Info "   Got error (expected): $($_.Exception.Message)"
    }
}

# Step 12: Test without authentication
Write-Title "12. Testing authentication requirement"

try {
    Invoke-RestMethod -Uri "$BASE_URL/admin/analytics" -Method GET
    Write-Failure "   Should have required authentication"
}
catch {
    if ($_.Exception.Response.StatusCode -eq "Unauthorized") {
        Write-Success "   Correctly requires authentication"
    }
    else {
        Write-Info "   Got error (expected): $($_.Exception.Message)"
    }
}

# Summary
Write-Title "Test Suite Complete"
Write-Host ""
Write-Host "Analytics Endpoints Tested:" -ForegroundColor Cyan
Write-Host "  * GET /admin/analytics (Summary)" -ForegroundColor White
Write-Host "  * GET /admin/analytics?from=...to=... (Time Range)" -ForegroundColor White
Write-Host "  * GET /admin/analytics/timeseries (Time Series)" -ForegroundColor White
Write-Host "  * GET /admin/analytics/keys/:id (API Key Stats)" -ForegroundColor White
Write-Host "  * GET /admin/logs (Request Logs)" -ForegroundColor White
Write-Host "  * GET /admin/logs?limit=...offset=... (Pagination)" -ForegroundColor White
Write-Host "  * GET /admin/logs?status=... (Status Filter)" -ForegroundColor White
Write-Host "  * Error handling for invalid inputs" -ForegroundColor White
Write-Host "  * Authentication requirement" -ForegroundColor White
Write-Host ""
Write-Host "Please review the output above for any failures." -ForegroundColor Yellow
