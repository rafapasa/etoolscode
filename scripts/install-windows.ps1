# Instalador eToolsCode (Windows PowerShell)
# Uso: .\install-windows.ps1 -BinaryPath .\etoolscode.exe

param(
  [string]$BinaryPath = ".\etoolscode.exe"
)

$ErrorActionPreference = "Stop"
$InstallDir = Join-Path $env:LOCALAPPDATA "etoolscode"
$ConfigDir  = Join-Path $env:USERPROFILE ".ds-ac"
$DataDir    = Join-Path $env:USERPROFILE "etoolscode-data\issues"

Write-Host "=== eToolsCode — instalador Windows ==="

New-Item -ItemType Directory -Force -Path $InstallDir, $ConfigDir, $DataDir | Out-Null

if (-not (Test-Path $BinaryPath)) {
  Write-Error "Binario nao encontrado: $BinaryPath"
}
Copy-Item -Force $BinaryPath (Join-Path $InstallDir "etoolscode.exe")

$pass1 = Read-Host "Senha da UI" -AsSecureString
$pass2 = Read-Host "Confirme a senha" -AsSecureString
$bstr1 = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($pass1)
$bstr2 = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($pass2)
$p1 = [Runtime.InteropServices.Marshal]::PtrToStringBSTR($bstr1)
$p2 = [Runtime.InteropServices.Marshal]::PtrToStringBSTR($bstr2)
if ($p1 -ne $p2 -or [string]::IsNullOrEmpty($p1)) {
  Write-Error "Senhas nao conferem"
}

# Hash: preferir etoolscode --hash-password se existir
$hash = ""
try {
  $hash = & (Join-Path $InstallDir "etoolscode.exe") --hash-password $p1 2>$null
} catch {}

$envFile = Join-Path $InstallDir ".env"
@"
HTTP_PORT=8080
DSAC_UI_PASSWORD_HASH=$hash
DSAC_UI_PASSWORD=$p1
ISSUES_DIR=$DataDir
LLM_CLIENT=1
DEEPSEEK_API_KEY=
META_API_KEY=
GEMINI_API_KEY=
"@ | Set-Content -Path $envFile -Encoding UTF8

$ws = @{ issues_dir = $DataDir; projects_root = $env:USERPROFILE; projects = @() } | ConvertTo-Json
Set-Content -Path (Join-Path $ConfigDir "workspace.json") -Value $ws -Encoding UTF8

Write-Host ""
Write-Host "Instalado em $InstallDir"
Write-Host "1) Edite API keys no .env ou use Config > Modelos na UI"
Write-Host "2) Inicie: $InstallDir\etoolscode.exe --ui"
Write-Host "3) Abra: http://127.0.0.1:8080"
