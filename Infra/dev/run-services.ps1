$env:AMANDACORE_STORE_PATH = Join-Path ([Environment]::GetFolderPath([Environment+SpecialFolder]::LocalApplicationData)) "amandacore\\platform-state.json"
$env:AMANDACORE_PVP_DUELS_ENABLED = "0"
$services = @(
  @{ Name = "auth-service"; Port = "8081" },
  @{ Name = "account-service"; Port = "8082" },
  @{ Name = "realm-service"; Port = "8083" },
  @{ Name = "character-service"; Port = "8084" },
  @{ Name = "world-service"; Port = "8085" },
  @{ Name = "admin-service"; Port = "8086" }
)

foreach ($service in $services) {
  Write-Host "Start $($service.Name) with: `$env:AMANDACORE_SERVICE_PORT=$($service.Port); `$env:AMANDACORE_PVP_DUELS_ENABLED=0; go run ./Services/cmd/$($service.Name)"
}
