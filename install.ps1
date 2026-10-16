# Misoku installer for Windows (PowerShell 5.1+ or PowerShell 7).
#
#   irm https://raw.githubusercontent.com/pwn27sbx/misoku/main/install.ps1 | iex
#
# It installs misoku and gh, GitHub's command-line tool (Misoku signs in to
# GitHub with it), side by side in %LOCALAPPDATA%\Programs\misoku, and adds
# that folder to your PATH. A gh that this installer did not put there (from
# winget, Scoop or GitHub's own installer) is never touched: Misoku uses it.
# When git is missing it is installed with winget, if there is winget; git is
# needed only to sync, so its absence never stops the install.
#
# Environment overrides:
#   MISOKU_VERSION      release tag to install (default: latest), e.g. v0.1.0
#   MISOKU_INSTALL_DIR  where to put misoku and gh
#   MISOKU_BASE_URL     where to download Misoku (default: GitHub releases)
#   MISOKU_GH_BASE_URL  where to download gh's archive (default: gh's GitHub
#                        release); tests point it at a local folder (file://)
#   MISOKU_GH_SHA256    the SHA-256 that gh's archive must have instead of the
#                        one written below; honoured only when
#                        MISOKU_GH_BASE_URL is a file:// URL to a folder on
#                        this computer (offline tests; never a network share),
#                        so a real download is always checked against the
#                        pinned hash
#
# Keep this file ASCII: Windows PowerShell 5.1 reads a file without a BOM in
# the system's legacy code page.

# The whole install is one function, called on the last line: piped into
# Invoke-Expression, its settings and variables stay inside it instead of
# changing the PowerShell window the user pasted the command into.
function Install-Misoku {
    $ErrorActionPreference = 'Stop'
    # Windows PowerShell 5.1 downloads many times slower while it draws a
    # progress bar.
    $ProgressPreference = 'SilentlyContinue'
    [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12

    $repo = 'pwn27sbx/misoku'

    # gh is pinned: its version and the SHA-256 of its archive are written
    # here rather than read from the release's checksums.txt, which would
    # only catch a damaged download, not a tampered release. The hash is the
    # one for gh_2.102.0_windows_amd64.zip in
    # https://github.com/cli/cli/releases/download/v2.102.0/gh_2.102.0_checksums.txt
    # To update: change the version and the hash here and in install.sh in a
    # pull request.
    $ghVersion = '2.102.0'
    $ghSha256 = 'ae64e556ecc240b200f7eba60d550e4bb60d78e860e69dd88c449405b86067f4'

    # This file, next to a gh this installer put there, holds its version:
    # only that gh is ever updated or replaced.
    $ghMarkName = '.misoku-gh-version'

    function Fail([string]$why) { throw "misoku install: $why" }

    # LocalFile: url is a file:// URL to a drive on this computer.
    # file://host/share is a network share (UNC), which is a download like
    # any other, and so is file:///%5Chost/share: .NET decides IsUnc before
    # decoding it, so the decoded path that is used must start with a drive.
    function LocalFile([string]$url) {
        $u = $null
        if (-not [Uri]::TryCreate($url, [UriKind]::Absolute, [ref]$u)) { return $false }
        return ($u.IsFile -and -not $u.IsUnc -and $u.LocalPath -match '^[A-Za-z]:[\\/]')
    }

    # Fetch copies url to file. file:// URLs are read directly: PowerShell 7
    # cannot download them, and the offline tests use them.
    function Fetch([string]$url, [string]$file) {
        if ($url -like 'file:*' -and -not (LocalFile $url)) {
            Fail "refusing ${url}: file:// URLs must point to a folder on this computer, not a network share; nothing was installed"
        }
        try {
            if (LocalFile $url) {
                Copy-Item -LiteralPath ([Uri]$url).LocalPath -Destination $file
            } else {
                Invoke-WebRequest -Uri $url -OutFile $file -UseBasicParsing
            }
        } catch {
            Fail "download failed: $url ($($_.Exception.Message)); nothing was installed"
        }
    }

    # Check stops unless file's SHA-256 is want.
    function Check([string]$file, [string]$want) {
        $name = Split-Path $file -Leaf
        if (-not $want) { Fail "no checksum for ${name}: stopping; nothing was installed" }
        $got = (Get-FileHash -LiteralPath $file -Algorithm SHA256).Hash.ToLower()
        if ($got -ne $want.ToLower()) {
            Fail "checksum mismatch for ${name}: the download is damaged or not the official one; nothing was installed"
        }
    }

    # Place copies a new program over to; a running copy cannot be replaced
    # on Windows, which is the usual reason this fails.
    function Place([string]$from, [string]$to) {
        try {
            Copy-Item -LiteralPath $from -Destination $to -Force
        } catch {
            Fail "could not write $to ($($_.Exception.Message)). If Misoku or gh is open, close it and run the install command again."
        }
    }

    # One x64 build: it also runs on Windows on ARM through emulation, and
    # so does the x64 gh.
    if ($env:PROCESSOR_ARCHITECTURE -eq 'x86' -and -not $env:PROCESSOR_ARCHITEW6432) {
        Fail 'Misoku needs 64-bit Windows.'
    }
    $archive = 'misoku-windows.zip'
    $ghArchive = "gh_${ghVersion}_windows_amd64.zip"

    $version = $env:MISOKU_VERSION
    $base = $env:MISOKU_BASE_URL
    if (-not $version) { $version = 'latest' }
    if (-not $base) {
        if ($version -eq 'latest') { $base = "https://github.com/$repo/releases/latest/download" }
        else { $base = "https://github.com/$repo/releases/download/$version" }
    }
    $ghBase = $env:MISOKU_GH_BASE_URL
    if (-not $ghBase) { $ghBase = "https://github.com/cli/cli/releases/download/v$ghVersion" }
    # Pinning is the point: a command someone pastes must not be able to swap
    # the hash for a download from the network, and that includes a network
    # share (file://host/share). Only a folder on this computer qualifies.
    if ((LocalFile $ghBase) -and $env:MISOKU_GH_SHA256) { $ghSha256 = $env:MISOKU_GH_SHA256 }

    $dir = $env:MISOKU_INSTALL_DIR
    if (-not $dir) { $dir = Join-Path $env:LOCALAPPDATA 'Programs\misoku' }
    $ghPath = Join-Path $dir 'gh.exe'
    $ghMark = Join-Path $dir $ghMarkName

    # What to do with gh: install it when there is none, update the one this
    # installer put here, and leave any other alone.
    $ghDo = 'install'
    $ghOther = $null
    if (Test-Path -LiteralPath $ghPath) {
        if (-not (Test-Path -LiteralPath $ghMark)) {
            $ghDo = 'keep'
            $ghOther = $ghPath
        } elseif ([IO.File]::ReadAllText($ghMark).Trim() -eq $ghVersion) {
            $ghDo = 'current'
        } else {
            $ghDo = 'update'
        }
    } else {
        $found = Get-Command gh -CommandType Application -ErrorAction SilentlyContinue | Select-Object -First 1
        if ($found) {
            $ghDo = 'keep'
            $ghOther = $found.Source
        }
    }

    $tmp = Join-Path ([IO.Path]::GetTempPath()) ('misoku-' + [Guid]::NewGuid())
    New-Item -ItemType Directory -Path $tmp | Out-Null
    try {
        # Everything is downloaded and checked before anything is installed.
        Write-Host "Downloading Misoku ($version) for Windows..."
        $zip = Join-Path $tmp $archive
        $sums = Join-Path $tmp 'checksums.txt'
        Fetch "$base/$archive" $zip
        Fetch "$base/checksums.txt" $sums
        $want = $null
        foreach ($line in [IO.File]::ReadAllLines($sums)) {
            $f = $line.Trim() -split '\s+'
            if ($f.Count -eq 2 -and $f[1] -eq $archive) { $want = $f[0]; break }
        }
        Check $zip $want
        $unpacked = Join-Path $tmp 'misoku'
        Expand-Archive -LiteralPath $zip -DestinationPath $unpacked -Force
        $newMisoku = Join-Path $unpacked 'misoku.exe'
        if (-not (Test-Path -LiteralPath $newMisoku)) { Fail "no misoku.exe in $archive; nothing was installed" }

        if ($ghDo -eq 'install' -or $ghDo -eq 'update') {
            Write-Host "Also installing gh $ghVersion, GitHub's command-line tool: Misoku signs in to GitHub with it."
            $ghZip = Join-Path $tmp $ghArchive
            Fetch "$ghBase/$ghArchive" $ghZip
            Check $ghZip $ghSha256
            Expand-Archive -LiteralPath $ghZip -DestinationPath (Join-Path $tmp 'gh') -Force
            $newGh = Join-Path $tmp 'gh\bin\gh.exe'
            if (-not (Test-Path -LiteralPath $newGh)) { Fail "no bin\gh.exe in $ghArchive; nothing was installed" }
        }

        New-Item -ItemType Directory -Path $dir -Force | Out-Null
        Place $newMisoku (Join-Path $dir 'misoku.exe')

        $userPath = [string][Environment]::GetEnvironmentVariable('Path', 'User')
        if (-not (($userPath -split ';') -contains $dir)) {
            [Environment]::SetEnvironmentVariable('Path', ($userPath.TrimEnd(';') + ';' + $dir).TrimStart(';'), 'User')
            $env:Path = "$env:Path;$dir"
            Write-Host "Added $dir to your PATH (open a new terminal to use it everywhere)."
        }

        $ver = & (Join-Path $dir 'misoku.exe') --version
        Write-Host "Installed $ver to $dir\misoku.exe"

        switch ($ghDo) {
            { $_ -eq 'install' -or $_ -eq 'update' } {
                # The mark goes first, so a gh cut short halfway is still
                # known as this installer's and replaced next time. Written
                # without a BOM, as install.sh writes it.
                [IO.File]::WriteAllText($ghMark, "pending`n")
                Place $newGh $ghPath
                [IO.File]::WriteAllText($ghMark, "$ghVersion`n")
                Write-Host "Installed gh $ghVersion to $ghPath"
            }
            'current' { Write-Host "gh $ghVersion is already in $dir" }
            'keep' { Write-Host "Using your gh at $ghOther (left as it is)" }
        }

        # git is needed only to sync; its absence never stops the install.
        # Git for Windows' own copy in Program Files counts: Misoku finds it
        # there even when this window's PATH does not have it yet.
        $gitHint = 'Misoku works without it; it is needed only to sync your notes between computers.'
        $hasGit = Get-Command git -CommandType Application -ErrorAction SilentlyContinue
        if (-not $hasGit -and $env:ProgramFiles) {
            $hasGit = Test-Path -LiteralPath (Join-Path $env:ProgramFiles 'Git\cmd\git.exe')
        }
        if (-not $hasGit) {
            if (Get-Command winget -ErrorAction SilentlyContinue) {
                Write-Host "git is not installed: installing it with winget (Windows may ask for permission). $gitHint"
                $ok = $true
                try {
                    & winget install --id Git.Git -e --source winget --accept-package-agreements --accept-source-agreements
                    if ($LASTEXITCODE -ne 0) { $ok = $false }
                } catch { $ok = $false }
                if ($ok) { Write-Host 'git is installed.' }
                else { Write-Host "git could not be installed. $gitHint Download it from https://git-scm.com/download/win" }
            } else {
                Write-Host "git is not installed. $gitHint Download it from https://git-scm.com/download/win"
            }
        }

        Write-Host "Run 'misoku' to open your notes (Windows Terminal with a Nerd Font is recommended for icons)."
    } finally {
        Remove-Item -LiteralPath $tmp -Recurse -Force -ErrorAction SilentlyContinue
    }
}

Install-Misoku
