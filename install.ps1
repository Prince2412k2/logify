# logify installer for Windows. iwr -useb .../install.ps1 | iex
$ErrorActionPreference = "Stop"
$Repo    = if ($env:LOGIFY_REPO)    { $env:LOGIFY_REPO }    else { "Prince2412k2/logify" }
$Version = if ($env:LOGIFY_VERSION) { $env:LOGIFY_VERSION } else { "latest" }
$Mode    = if ($env:INSTALL_MODE)   { $env:INSTALL_MODE }   else { "release" }
$Prefix  = if ($env:LOGIFY_PREFIX)  { $env:LOGIFY_PREFIX }  else { Join-Path $env:LOCALAPPDATA "Programs\logify" }
function Write-Step ($m) { Write-Host "▸ $m" -ForegroundColor Yellow }
function Write-Done ($m) { Write-Host "✓ $m" -ForegroundColor Green }
function Write-Warn ($m) { Write-Host "⚠  $m" -ForegroundColor Yellow }
function Die ($m) { Write-Host "✕ $m" -ForegroundColor Red; exit 1 }
function Get-Arch {
  switch ($env:PROCESSOR_ARCHITECTURE) {
    "AMD64" { return "amd64" }
    "ARM64" { return "arm64" }
    default { Die "unsupported arch: $env:PROCESSOR_ARCHITECTURE" }
  }
}
function Resolve-LatestTag {
  try {
    $r = Invoke-RestMethod -UseBasicParsing -Headers @{ "User-Agent" = "logify-install" } -Uri "https://api.github.com/repos/$Repo/releases/latest"
    return $r.tag_name
  } catch { Die "could not resolve latest release for $Repo (try LOGIFY_VERSION=nightly)" }
}
function Add-ToUserPath ($dir) {
  $up = [Environment]::GetEnvironmentVariable("PATH", "User")
  if (-not (($up -split ';') | Where-Object { $_ -ieq $dir })) {
    $new = if ([string]::IsNullOrEmpty($up)) { $dir } else { "$up;$dir" }
    [Environment]::SetEnvironmentVariable("PATH", $new, "User")
    Write-Done "added $dir to your user PATH"
    Write-Warn "open a new terminal for the PATH change to take effect"
  }
}
function Test-Binary ($d) {
  try {
    $o = & $d version 2>&1
    if ($LASTEXITCODE -ne 0 -or -not $o) { throw "no output" }
    Write-Done $o
  } catch { Die "installed binary failed to run" }
}
function Install-Release {
  $arch = Get-Arch
  $tag  = if ($Version -eq "latest") { Resolve-LatestTag } else { $Version }
  $url  = "https://github.com/$Repo/releases/download/$tag/logify-windows-$arch.exe"
  Write-Step "downloading $tag for windows-$arch"
  New-Item -ItemType Directory -Force -Path $Prefix | Out-Null
  $dest = Join-Path $Prefix "logify.exe"
  try {
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    Invoke-WebRequest -UseBasicParsing -Uri $url -OutFile $dest
  } catch { Die "download failed: $url" }
  Write-Done "installed $dest"
  Test-Binary $dest
  Add-ToUserPath $Prefix
  Write-Host ""
  Write-Host "next: open a new terminal and run 'logify login'"
}
function Install-FromSource {
  if (-not (Get-Command go  -ErrorAction SilentlyContinue)) { Die "Go required: https://go.dev/dl/" }
  if (-not (Get-Command git -ErrorAction SilentlyContinue)) { Die "Git required" }
  $tmp = New-Item -ItemType Directory -Path (Join-Path $env:TEMP ("logify-" + [guid]::NewGuid().Guid))
  Write-Step "cloning $Repo"
  git clone --depth 1 "https://github.com/$Repo.git" $tmp.FullName | Out-Null
  Push-Location $tmp.FullName
  try {
    go build -ldflags="-s -w" -o logify.exe ./cmd/logify
    if ($LASTEXITCODE -ne 0) { Die "go build failed" }
    New-Item -ItemType Directory -Force -Path $Prefix | Out-Null
    $dest = Join-Path $Prefix "logify.exe"
    Move-Item -Force (Join-Path $tmp.FullName "logify.exe") $dest
    Write-Done "built and installed $dest"
    Test-Binary $dest
  } finally {
    Pop-Location
    Remove-Item -Recurse -Force $tmp.FullName -ErrorAction SilentlyContinue
  }
  Add-ToUserPath $Prefix
  Write-Host ""
  Write-Host "next: open a new terminal and run 'logify login'"
}
switch ($Mode) {
  "release" { Install-Release }
  "source"  { Install-FromSource }
  default   { Die "INSTALL_MODE must be 'release' or 'source', got '$Mode'" }
}
