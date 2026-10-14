# Misoku installer for Windows (PowerShell 5.1+ or PowerShell 7).
#
#   irm https://raw.githubusercontent.com/pwn27sbx/misoku/main/install.ps1 | iex
#
# Environment overrides: MISOKU_VERSION (e.g. v0.1.0), MISOKU_INSTALL_DIR,
# MISOKU_BASE_URL.
$ErrorActionPreference = 'Stop'
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

$repo = 'pwn27sbx/misoku'

# One x64 build: it also runs on Windows on ARM through emulation.
if ($env:PROCESSOR_ARCHITECTURE -eq 'x86' -and -not $env:PROCESSOR_ARCHITEW6432) {
    throw 'Misoku needs 64-bit Windows.'
}
$archive = 'misoku-windows.zip'

$version = $env:MISOKU_VERSION
$base = $env:MISOKU_BASE_URL
if (-not $base) {
    if ($version) { $base = "https://github.com/$repo/releases/download/$version" }
    else { $base = "https://github.com/$repo/releases/latest/download"; $version = 'latest' }
}

$tmp = Join-Path ([IO.Path]::GetTempPath()) ("misoku-" + [Guid]::NewGuid())
New-Item -ItemType Directory -Path $tmp | Out-Null
try {
    Write-Host "Downloading Misoku ($version) for Windows..."
    $zip = Join-Path $tmp $archive
    Invoke-WebRequest "$base/$archive" -OutFile $zip -UseBasicParsing

    try {
        $sums = (Invoke-WebRequest "$base/checksums.txt" -UseBasicParsing).Content
        $line = ($sums -split "`n") | Where-Object { $_ -match [regex]::Escape($archive) + '\s*$' } | Select-Object -First 1
        if ($line) {
            $want = ($line -split '\s+')[0].ToLower()
            $got = (Get-FileHash $zip -Algorithm SHA256).Hash.ToLower()
            if ($want -ne $got) { throw "Checksum mismatch for $archive" }
        }
    } catch [System.Net.WebException] { }

    Expand-Archive $zip -DestinationPath $tmp -Force

    $dir = $env:MISOKU_INSTALL_DIR
    if (-not $dir) { $dir = Join-Path $env:LOCALAPPDATA 'Programs\misoku' }
    New-Item -ItemType Directory -Path $dir -Force | Out-Null
    Copy-Item (Join-Path $tmp 'misoku.exe') (Join-Path $dir 'misoku.exe') -Force

    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    if (-not (($userPath -split ';') -contains $dir)) {
        [Environment]::SetEnvironmentVariable('Path', ($userPath.TrimEnd(';') + ';' + $dir).TrimStart(';'), 'User')
        $env:Path = "$env:Path;$dir"
        Write-Host "Added $dir to your PATH (open a new terminal to use it everywhere)."
    }

    $ver = & (Join-Path $dir 'misoku.exe') --version
    Write-Host "Installed $ver to $dir\misoku.exe"
    Write-Host "Run 'misoku' in a folder of notes. Windows Terminal with a Nerd Font is recommended."
} finally {
    Remove-Item $tmp -Recurse -Force -ErrorAction SilentlyContinue
}
