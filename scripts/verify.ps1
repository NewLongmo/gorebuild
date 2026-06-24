param(
  [switch]$SkipFrontend,
  [switch]$SkipBackendBuild
)

$ErrorActionPreference = "Stop"
$Root = Resolve-Path (Join-Path $PSScriptRoot "..")
$GoCache = Join-Path $Root ".cache\go-build"
New-Item -ItemType Directory -Force -Path $GoCache | Out-Null
$env:GOCACHE = $GoCache

function Resolve-CommandPath {
  param(
    [string[]]$Names,
    [string[]]$Fallbacks = @()
  )

  foreach ($name in $Names) {
    $command = Get-Command $name -ErrorAction SilentlyContinue
    if ($command) {
      return $command.Source
    }
  }

  foreach ($fallback in $Fallbacks) {
    if (Test-Path -LiteralPath $fallback) {
      return $fallback
    }
  }

  throw "Unable to find command: $($Names -join ', ')"
}

function Invoke-Step {
  param(
    [string]$Name,
    [string]$WorkingDirectory,
    [string]$FilePath,
    [string[]]$Arguments
  )

  Write-Host "==> $Name"
  Push-Location $WorkingDirectory
  try {
    & $FilePath @Arguments
    if ($LASTEXITCODE -ne 0) {
      throw "$Name failed with exit code $LASTEXITCODE"
    }
  } finally {
    Pop-Location
  }
}

function Test-PowerShellSyntax {
  param(
    [string]$Name,
    [string]$Path
  )

  Write-Host "==> $Name syntax"
  $tokens = $null
  $parseErrors = $null
  [System.Management.Automation.Language.Parser]::ParseFile($Path, [ref]$tokens, [ref]$parseErrors) | Out-Null
  if ($parseErrors.Count -gt 0) {
    $messages = $parseErrors | ForEach-Object { $_.Message }
    throw "$Name syntax failed: $($messages -join '; ')"
  }
}

$Go = Resolve-CommandPath `
  -Names @("go.exe", "go") `
  -Fallbacks @("C:\Program Files\Go\bin\go.exe")

$Npm = Resolve-CommandPath -Names @("npm.cmd", "npm")

Test-PowerShellSyntax `
  -Name "Smoke script" `
  -Path (Join-Path $Root "scripts\smoke.ps1")

Invoke-Step `
  -Name "Backend tests" `
  -WorkingDirectory (Join-Path $Root "backend") `
  -FilePath $Go `
  -Arguments @("test", "./...")

if (-not $SkipBackendBuild) {
  Invoke-Step `
    -Name "Backend build" `
    -WorkingDirectory (Join-Path $Root "backend") `
    -FilePath $Go `
    -Arguments @("build", "-o", "dw0rdwk-api.exe", "./cmd/server")
}

if (-not $SkipFrontend) {
  Invoke-Step `
    -Name "Frontend build" `
    -WorkingDirectory (Join-Path $Root "frontend") `
    -FilePath $Npm `
    -Arguments @("run", "build")
}

Write-Host "Verification completed."
