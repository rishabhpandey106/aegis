$ErrorActionPreference = "Stop"

$Headers = @{
    "Authorization" = "Bearer super-secret-aegis-token"
    "Content-Type"  = "application/json"
}

Write-Host "======================================"
Write-Host "Testing Aegis Control Plane APIs"
Write-Host "======================================"

Write-Host "`n[1/5] Creating Organization..." -ForegroundColor Cyan
$OrgBody = '{"name": "Aegis Security Corp", "plan": "enterprise"}'
$Org = Invoke-RestMethod -Uri "http://localhost:8081/api/v1/organizations" -Method Post -Headers $Headers -Body $OrgBody
Write-Host "Success! Created Org ID: $($Org.id)" -ForegroundColor Green

Write-Host "`n[2/5] Creating User..." -ForegroundColor Cyan
$UserBody = "{`"org_id`": `"$($Org.id)`", `"email`": `"admin@aegis.corp`", `"role`": `"admin`"}"
$User = Invoke-RestMethod -Uri "http://localhost:8081/api/v1/users" -Method Post -Headers $Headers -Body $UserBody
Write-Host "Success! Created User: $($User.email)" -ForegroundColor Green

Write-Host "`n[3/5] Creating Project..." -ForegroundColor Cyan
$ProjBody = "{`"org_id`": `"$($Org.id)`", `"name`": `"Main API`", `"upstream_url`": `"http://httpbin.org`"}"
$Project = Invoke-RestMethod -Uri "http://localhost:8081/api/v1/projects" -Method Post -Headers $Headers -Body $ProjBody
Write-Host "Success! Created Project ID: $($Project.id)" -ForegroundColor Green

Write-Host "`n[4/5] Creating Dynamic Security Rule (Rate Limit)..." -ForegroundColor Cyan
$RuleBody = '{"rule_type": "rate_limit", "configuration": {"limit": 2, "window": 60}, "action": "block"}'
$Rule = Invoke-RestMethod -Uri "http://localhost:8081/api/v1/projects/$($Project.id)/rules" -Method Post -Headers $Headers -Body $RuleBody
Write-Host "Success! Created Rule Type: $($Rule.rule_type)" -ForegroundColor Green

Write-Host "`n[5/5] Fetching Analytics Data..." -ForegroundColor Cyan
$Analytics = Invoke-RestMethod -Uri "http://localhost:8081/api/v1/projects/$($Project.id)/analytics" -Method Get -Headers $Headers
Write-Host "Success! Retrieved Analytics Data: Total Requests = $($Analytics.total_requests)" -ForegroundColor Green

Write-Host "`n======================================"
Write-Host "ALL TESTS PASSED!" -ForegroundColor Green
Write-Host "======================================"
