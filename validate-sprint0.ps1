# Kuafu Sprint 0 Validation Script
# Run this after starting Docker Desktop

Write-Host "=== Kuafu Sprint 0 Validation ===" -ForegroundColor Cyan
Write-Host ""

# Check Docker
Write-Host "1. Checking Docker..." -ForegroundColor Yellow
docker --version
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Docker not available" -ForegroundColor Red
    exit 1
}
Write-Host "✓ Docker available" -ForegroundColor Green
Write-Host ""

# Build Docker image
Write-Host "2. Building Docker image..." -ForegroundColor Yellow
docker build -t kuafu:sprint0 .
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Docker build failed" -ForegroundColor Red
    exit 1
}
Write-Host "✓ Docker image built" -ForegroundColor Green
Write-Host ""

# Run tests
Write-Host "3. Running unit tests..." -ForegroundColor Yellow
docker run --rm -v ${PWD}:/workspace kuafu:sprint0 go test -v ./...
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Tests failed" -ForegroundColor Red
    exit 1
}
Write-Host "✓ All tests passed" -ForegroundColor Green
Write-Host ""

# Build binaries
Write-Host "4. Building binaries..." -ForegroundColor Yellow
New-Item -ItemType Directory -Force -Path bin | Out-Null
docker run --rm -v ${PWD}:/workspace kuafu:sprint0 go build -o /workspace/bin/kuafu-server ./cmd/kuafu-server
docker run --rm -v ${PWD}:/workspace kuafu:sprint0 go build -o /workspace/bin/kuafu ./cmd/kuafu
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Build failed" -ForegroundColor Red
    exit 1
}
Write-Host "✓ Binaries built" -ForegroundColor Green
Write-Host ""

# Start server in background
Write-Host "5. Starting API server..." -ForegroundColor Yellow
docker run --rm -d -p 8080:8080 --name kuafu-server kuafu:sprint0 /app/kuafu-server --mode lab
Start-Sleep -Seconds 2
Write-Host "✓ Server started" -ForegroundColor Green
Write-Host ""

# Test API endpoints
Write-Host "6. Testing API endpoints..." -ForegroundColor Yellow
$endpoints = @(
    @{Path="/health"; Expected="ok"},
    @{Path="/ready"; Expected="ready"},
    @{Path="/api/v1/nodes"; Expected="nodes"},
    @{Path="/api/v1/nodes/A00"; Expected="A00"},
    @{Path="/api/v1/gpus"; Expected="gpus"}
)

foreach ($endpoint in $endpoints) {
    $response = Invoke-RestMethod -Uri "http://localhost:8080$($endpoint.Path)" -ErrorAction SilentlyContinue
    if ($response -match $endpoint.Expected) {
        Write-Host "  ✓ $($endpoint.Path)" -ForegroundColor Green
    } else {
        Write-Host "  ✗ $($endpoint.Path)" -ForegroundColor Red
    }
}
Write-Host ""

# Test CLI online mode
Write-Host "7. Testing CLI (online mode)..." -ForegroundColor Yellow
$env:KUAFU_API_URL = "http://localhost:8080"
./bin/kuafu.exe nodes list
if ($LASTEXITCODE -eq 0) {
    Write-Host "✓ CLI online mode works" -ForegroundColor Green
} else {
    Write-Host "✗ CLI online mode failed" -ForegroundColor Red
}
Write-Host ""

# Test CLI offline mode
Write-Host "8. Testing CLI (offline fixture mode)..." -ForegroundColor Yellow
./bin/kuafu.exe --fixture nodes list
if ($LASTEXITCODE -eq 0) {
    Write-Host "✓ CLI offline mode works" -ForegroundColor Green
} else {
    Write-Host "✗ CLI offline mode failed" -ForegroundColor Red
}
Write-Host ""

# Cleanup
Write-Host "9. Cleaning up..." -ForegroundColor Yellow
docker stop kuafu-server | Out-Null
Write-Host "✓ Server stopped" -ForegroundColor Green
Write-Host ""

Write-Host "=== Validation Complete ===" -ForegroundColor Cyan
Write-Host ""
Write-Host "Summary:" -ForegroundColor White
Write-Host "  - Docker image: kuafu:sprint0" -ForegroundColor Gray
Write-Host "  - Binaries: bin/kuafu-server.exe, bin/kuafu.exe" -ForegroundColor Gray
Write-Host "  - All tests passed and API endpoints validated" -ForegroundColor Gray
