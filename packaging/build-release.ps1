param([string]$Version = '1.0.1', [switch]$SkipBuild)
$ErrorActionPreference = 'Stop'
$root = Split-Path $PSScriptRoot -Parent
Push-Location $root
try {
  if ($Version -notmatch '^\d+\.\d+\.\d+$') { throw 'Use a numeric three-part version.' }
  if (!(Select-String -Path 'internal/update/update.go' -SimpleMatch "const Version = `"$Version`"" -Quiet)) { throw 'Version differs from internal/update/update.go.' }
  if (!$SkipBuild) {
    & wails build -o elastic-ui-release.exe
    if ($LASTEXITCODE -ne 0) { throw 'Wails build failed.' }
  }
  $bootstrapper = Join-Path $PSScriptRoot 'MicrosoftEdgeWebview2Setup.exe'
  if (!(Test-Path $bootstrapper)) {
    Invoke-WebRequest 'https://go.microsoft.com/fwlink/p/?LinkId=2124703' -OutFile $bootstrapper
  }
  $signature = Get-AuthenticodeSignature $bootstrapper
  if ($signature.Status -ne 'Valid' -or $signature.SignerCertificate.Subject -notmatch 'O=Microsoft Corporation') {
    throw 'WebView2 bootstrapper signature is invalid.'
  }
  $compiler = Get-Command makensis.exe -ErrorAction SilentlyContinue
  if (!$compiler) { $compiler = Get-Item "${env:ProgramFiles(x86)}/NSIS/makensis.exe" }
  & $compiler.FullName '/INPUTCHARSET' 'UTF8' "/DAPP_VERSION=$Version" (Join-Path $PSScriptRoot 'installer.nsi')
  if ($LASTEXITCODE -ne 0) { throw 'Installer build failed.' }
  Copy-Item 'build/bin/elastic-ui-release.exe' 'build/bin/elastic-ui-windows-amd64.exe' -Force
  $names = @('elastic-ui-windows-amd64.exe', 'elastic-ui-windows-amd64-setup.exe')
  $checksums = foreach ($name in $names) {
    $hash = (Get-FileHash (Join-Path $root "build/bin/$name") -Algorithm SHA256).Hash.ToLowerInvariant()
    "$hash  $name"
  }
  [IO.File]::WriteAllLines((Join-Path $root 'build/bin/SHA256SUMS'), $checksums, [Text.UTF8Encoding]::new($false))
} finally { Pop-Location }
