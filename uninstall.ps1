# Removes the logify binary, prunes PATH, prompts about config.

$ErrorActionPreference = "Stop"

function Write-Done ($m) { Write-Host "✓ $m" -ForegroundColor Green }
function Write-Warn ($m) { Write-Host "⚠  $m" -ForegroundColor Yellow }

$Prefix = if ($env:LOGIFY_PREFIX) { $env:LOGIFY_PREFIX } else { Join-Path $env:LOCALAPPDATA "Programs\logify" }
$exe    = Join-Path $Prefix "logify.exe"

if (Test-Path $exe) {
    Remove-Item -Force $exe
    Write-Done "removed $exe"
    # Drop the (now empty) install dir if appropriate.
    if (Test-Path $Prefix) {
        $remaining = Get-ChildItem $Prefix -Force | Measure-Object | Select-Object -ExpandProperty Count
        if ($remaining -eq 0) { Remove-Item -Recurse -Force $Prefix }
    }
} else {
    Write-Warn "no binary at $exe"
}

# Strip the install dir from the user PATH.
$userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($userPath) {
    $parts = $userPath -split ';' | Where-Object { $_ -ne "" -and $_ -ne $Prefix }
    $new = $parts -join ';'
    if ($new -ne $userPath) {
        [Environment]::SetEnvironmentVariable("PATH", $new, "User")
        Write-Done "removed $Prefix from user PATH"
        Write-Warn "open a new terminal for the PATH change to take effect"
    }
}

$cfg = Join-Path $env:USERPROFILE ".config\logify"
if (Test-Path $cfg) {
    Write-Host ""
    Write-Warn "config still exists at $cfg"
    Write-Host "  remove it too?  Remove-Item -Recurse -Force '$cfg'"
}
