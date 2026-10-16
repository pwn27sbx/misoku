# Tests install.ps1 without a network: a fake Misoku release and a fake gh
# archive in a temporary folder, served as file:// URLs (MISOKU_BASE_URL,
# MISOKU_GH_BASE_URL), and a PATH of only fakes (gh, winget), so the
# runner's own git and gh are out of sight. Each case installs into its own
# folder, and the user PATH install.ps1 edits is put back at the end.
#
#   pwsh -File scripts/test-install.ps1                  # PowerShell 7
#   pwsh -File scripts/test-install.ps1 -Shell powershell # Windows PowerShell 5.1
#
# -Shell is the PowerShell that runs install.ps1. Needs Windows (the fakes
# are .exe and .cmd programs), Go (to build the fake misoku.exe and gh.exe)
# and Python (a loopback server for the one case that needs a non-file URL).
#
# Keep this file ASCII, like install.ps1.
param([string]$Shell = 'pwsh')

$ErrorActionPreference = 'Stop'
if ($env:OS -ne 'Windows_NT') { throw 'test-install.ps1 runs on Windows only' }

$root = Split-Path -Parent $PSScriptRoot
$installer = Join-Path $root 'install.ps1'
$shellExe = (Get-Command $Shell -CommandType Application | Select-Object -First 1).Source
$ghv = ([regex]::Match((Get-Content -Raw $installer), "\`$ghVersion = '([^']+)'")).Groups[1].Value
if (-not $ghv) { throw 'ghVersion not found in install.ps1' }

$work = Join-Path ([IO.Path]::GetTempPath()) ('misoku-install-test-' + [Guid]::NewGuid())
New-Item -ItemType Directory -Path $work | Out-Null
$savedUserPath = [Environment]::GetEnvironmentVariable('Path', 'User')
$server = $null
Add-Type -AssemblyName System.IO.Compression.FileSystem

function Sum([string]$file) { (Get-FileHash -LiteralPath $file -Algorithm SHA256).Hash.ToLower() }
function FileUrl([string]$dir) { ([Uri]$dir).AbsoluteUri }
function Text([string]$file) { [IO.File]::ReadAllText($file).Trim() }

# Build writes an .exe that prints msg (and ignores its arguments).
function Build([string]$out, [string]$msg) {
    $src = Join-Path $work 'fake-src'
    if (-not (Test-Path $src)) {
        New-Item -ItemType Directory -Path $src | Out-Null
        [IO.File]::WriteAllText((Join-Path $src 'go.mod'), "module fake`n`ngo 1.21`n")
        [IO.File]::WriteAllText((Join-Path $src 'main.go'), "package main`n`nimport `"fmt`"`n`nvar msg string`n`nfunc main() { fmt.Println(msg) }`n")
    }
    New-Item -ItemType Directory -Path (Split-Path $out) -Force | Out-Null
    Push-Location $src
    try {
        & go build -o $out -ldflags "-X=main.msg=$msg" .
        if ($LASTEXITCODE -ne 0) { throw "go build failed for $out" }
    } finally { Pop-Location }
}

$script:fails = 0
function Expect([string]$name, [bool]$ok) {
    if ($ok) { Write-Host "ok   $name" }
    else {
        Write-Host "FAIL $name"
        $script:out -split "`n" | ForEach-Object { Write-Host "     | $_" }
        $script:fails++
    }
}
# Says: the output has text. Windows PowerShell may wrap a long error at
# the console width, so the output is also read with its line breaks taken
# out.
function Says([string]$text) { $script:out.Contains($text) -or ($script:out -replace "`r?`n", '').Contains($text) }

try {
    # The fake Misoku release.
    $rel = Join-Path $work 'release'
    Build (Join-Path $work 'build\misoku.exe') 'misoku-new'
    New-Item -ItemType Directory -Path $rel | Out-Null
    [IO.Compression.ZipFile]::CreateFromDirectory((Join-Path $work 'build'), (Join-Path $rel 'misoku-windows.zip'))
    [IO.File]::WriteAllText((Join-Path $rel 'checksums.txt'), "$(Sum (Join-Path $rel 'misoku-windows.zip'))  misoku-windows.zip`n")

    # The fake gh archive, laid out like the real one: bin\gh.exe at the root.
    $ghrel = Join-Path $work 'gh-release'
    $ghArchive = "gh_${ghv}_windows_amd64.zip"
    Build (Join-Path $work 'gh-build\bin\gh.exe') "gh-$ghv"
    New-Item -ItemType Directory -Path $ghrel | Out-Null
    [IO.Compression.ZipFile]::CreateFromDirectory((Join-Path $work 'gh-build'), (Join-Path $ghrel $ghArchive))
    $ghSum = Sum (Join-Path $ghrel $ghArchive)

    # Fakes: a winget that logs and succeeds, one that fails, and a gh that
    # is someone else's. An empty Program Files hides the runner's git.
    $fakes = Join-Path $work 'fakes'
    New-Item -ItemType Directory -Path "$fakes\winget-ok", "$fakes\winget-fail", "$fakes\their-gh", "$fakes\pf-empty", "$fakes\pf-git\Git\cmd" | Out-Null
    [IO.File]::WriteAllText("$fakes\winget-ok\winget.cmd", "@echo winget %*>>`"%LOG%`"`r`n@exit /b 0`r`n")
    [IO.File]::WriteAllText("$fakes\winget-fail\winget.cmd", "@echo winget %*>>`"%LOG%`"`r`n@exit /b 1`r`n")
    [IO.File]::WriteAllText("$fakes\their-gh\gh.exe", 'theirs')
    [IO.File]::WriteAllText("$fakes\pf-git\Git\cmd\git.exe", 'git')
    $sys = "$env:SystemRoot\System32;$env:SystemRoot\System32\WindowsPowerShell\v1.0"

    # Run sets the environment for one case, runs install.ps1 in a new
    # PowerShell and puts the environment back. Leaves the output in
    # $script:out and the exit code in $script:code.
    #
    # Windows rebuilds ProgramFiles from ProgramW6432 for every new 64-bit
    # process, and the runner has Git in the real Program Files, so the
    # child sets PATH, ProgramFiles and ProgramW6432 again itself, before
    # install.ps1 runs. -Probe prints what the child then sees instead of
    # installing.
    function Run([string]$case, [hashtable]$vars, [switch]$Piped, [switch]$Probe) {
        $env0 = @{
            MISOKU_INSTALL_DIR = (Join-Path $work "dir-$case")
            MISOKU_BASE_URL    = (FileUrl $rel)
            MISOKU_GH_BASE_URL = (FileUrl $ghrel)
            MISOKU_GH_SHA256   = $ghSum
            MISOKU_VERSION     = $null
            PATH               = $sys
            ProgramFiles       = "$fakes\pf-empty"
            ProgramW6432       = $null
            LOG                = (Join-Path $work "log-$case.txt")
        }
        foreach ($k in $vars.Keys) { $env0[$k] = $vars[$k] }
        $env0.ProgramW6432 = $env0.ProgramFiles
        $pre = ''
        foreach ($k in 'PATH', 'ProgramFiles', 'ProgramW6432') {
            $pre += "`$env:$k = '$($env0[$k] -replace "'", "''")'; "
        }
        $saved = @{}
        foreach ($k in $env0.Keys) {
            $saved[$k] = [Environment]::GetEnvironmentVariable($k)
            [Environment]::SetEnvironmentVariable($k, $env0[$k])
        }
        $old = $ErrorActionPreference
        $ErrorActionPreference = 'Continue'
        try {
            if ($Probe) {
                $cmd = $pre + '"git=" + (Get-Command git -CommandType Application -ErrorAction SilentlyContinue | Select-Object -First 1 | ForEach-Object Source); "pf=" + $env:ProgramFiles; "pfgit=" + (Test-Path (Join-Path $env:ProgramFiles ''Git\cmd\git.exe''))'
            } elseif ($Piped) {
                # Like `irm ... | iex`: the script arrives as one string.
                $cmd = $pre + "Get-Content -Raw -LiteralPath '$installer' | Invoke-Expression"
            } else {
                $cmd = $pre + "& '$installer'"
            }
            $script:out = (& $shellExe -NoProfile -NonInteractive -ExecutionPolicy Bypass -Command $cmd 2>&1 | ForEach-Object { "$_" }) -join "`n"
            $script:code = $LASTEXITCODE
        } finally {
            $ErrorActionPreference = $old
            foreach ($k in $saved.Keys) { [Environment]::SetEnvironmentVariable($k, $saved[$k]) }
        }
        return (Join-Path $work "dir-$case")
    }

    Write-Host "install.ps1 under $shellExe"

    # 0. The child really sees no git: none on PATH, none in Program Files.
    $null = Run 'probe' @{} -Probe
    Write-Host ($out -replace '(?m)^', '     probe: ')
    Expect 'isolation: no git on the child PATH' (Says "git=`n")
    Expect 'isolation: Program Files is the empty fake' (Says "pf=$fakes\pf-empty")
    Expect 'isolation: no git in that Program Files' (Says 'pfgit=False')

    # 1. Fresh install: misoku and gh side by side, gh marked as ours, the
    # folder added to the user PATH, and git's absence said without winget.
    $d = Run 'fresh' @{}
    Expect 'fresh: exits 0' ($code -eq 0)
    Expect 'fresh: misoku installed' ((Test-Path "$d\misoku.exe") -and ((& "$d\misoku.exe") -eq 'misoku-new'))
    Expect 'fresh: gh installed next to it' ((Test-Path "$d\gh.exe") -and ((& "$d\gh.exe") -eq "gh-$ghv"))
    Expect 'fresh: gh marked with its version' ((Test-Path "$d\.misoku-gh-version") -and ((Text "$d\.misoku-gh-version") -eq $ghv))
    Expect 'fresh: mark has no BOM' ([IO.File]::ReadAllBytes("$d\.misoku-gh-version")[0] -ne 0xEF)
    Expect 'fresh: says gh is installed (one line)' (Says "Also installing gh $ghv")
    Expect 'fresh: folder on the user PATH' (([Environment]::GetEnvironmentVariable('Path', 'User') -split ';') -contains $d)
    Expect 'fresh: no git, no winget: says where to get it' (Says 'git is not installed. Misoku works without it')
    Expect 'fresh: download link' (Says 'https://git-scm.com/download/win')

    # 2. The same install again: gh is already current.
    $d = Run 'fresh' @{}
    Expect 'rerun: exits 0' ($code -eq 0)
    Expect 'rerun: gh already current' (Says "gh $ghv is already in $d")
    Expect 'rerun: folder once on the user PATH' (@([Environment]::GetEnvironmentVariable('Path', 'User') -split ';' | Where-Object { $_ -eq $d }).Count -eq 1)

    # 3. Our older gh is updated.
    [IO.File]::WriteAllText("$d\gh.exe", 'old')
    [IO.File]::WriteAllText("$d\.misoku-gh-version", "2.0.0`n")
    $d = Run 'fresh' @{}
    Expect 'update: exits 0' ($code -eq 0)
    Expect 'update: our gh updated' ((& "$d\gh.exe") -eq "gh-$ghv")
    Expect 'update: mark updated' ((Text "$d\.misoku-gh-version") -eq $ghv)

    # 4. A gh we did not install, in the install folder, is left as it is.
    $d = Join-Path $work 'dir-theirs'
    New-Item -ItemType Directory -Path $d | Out-Null
    [IO.File]::WriteAllText("$d\gh.exe", 'theirs')
    $d = Run 'theirs' @{}
    Expect 'their gh in our folder: exits 0' ($code -eq 0)
    Expect 'their gh in our folder: untouched' ((Text "$d\gh.exe") -eq 'theirs')
    Expect 'their gh in our folder: not marked' (-not (Test-Path "$d\.misoku-gh-version"))
    Expect 'their gh in our folder: says so' (Says "Using your gh at $d\gh.exe")

    # 5. A gh elsewhere on the PATH (winget's, Scoop's) is used, not shadowed.
    $d = Run 'onpath' @{ PATH = "$fakes\their-gh;$sys" }
    Expect 'gh on PATH: exits 0' ($code -eq 0)
    Expect 'gh on PATH: none installed next to misoku' (-not (Test-Path "$d\gh.exe"))
    Expect 'gh on PATH: untouched' ((Text "$fakes\their-gh\gh.exe") -eq 'theirs')
    Expect 'gh on PATH: says so' (Says "Using your gh at $fakes\their-gh\gh.exe")

    # 6. A bad Misoku checksum stops before anything is installed.
    $bad = Join-Path $work 'bad-release'
    Copy-Item -Recurse $rel $bad
    [IO.File]::WriteAllText("$bad\checksums.txt", "0000  misoku-windows.zip`n")
    $d = Run 'badmisoku' @{ MISOKU_BASE_URL = (FileUrl $bad) }
    Expect 'bad misoku checksum: fails' ($code -ne 0)
    Expect 'bad misoku checksum: says so' (Says 'checksum mismatch for misoku-windows.zip')
    Expect 'bad misoku checksum: nothing installed' (-not (Test-Path $d))

    # 6a. A checksums.txt that does not list the archive stops too.
    [IO.File]::WriteAllText("$bad\checksums.txt", "$(Sum "$rel\misoku-windows.zip")  misoku-macos.tar.gz`n")
    $d = Run 'unlisted' @{ MISOKU_BASE_URL = (FileUrl $bad) }
    Expect 'unlisted archive: fails' ($code -ne 0)
    Expect 'unlisted archive: says so' (Says 'no checksum for misoku-windows.zip')
    Expect 'unlisted archive: nothing installed' (-not (Test-Path $d))

    # 6b. No checksums.txt at all stops too (it used to be skipped).
    Remove-Item "$bad\checksums.txt"
    $d = Run 'nosums' @{ MISOKU_BASE_URL = (FileUrl $bad) }
    Expect 'no checksums.txt: fails' ($code -ne 0)
    Expect 'no checksums.txt: says the download failed' (Says 'download failed')
    Expect 'no checksums.txt: nothing installed' (-not (Test-Path $d))

    # 7. A gh archive that is not the one pinned in install.ps1 stops the
    # install: without the test hash, the pinned one applies.
    $d = Run 'badgh' @{ MISOKU_GH_SHA256 = $null }
    Expect 'bad gh checksum: fails' ($code -ne 0)
    Expect 'bad gh checksum: says so' (Says "checksum mismatch for $ghArchive")
    Expect 'bad gh checksum: nothing installed' (-not (Test-Path $d))

    # 8. MISOKU_GH_SHA256 is ignored when gh comes from the network: the
    # pinned hash still applies, and the fake archive stops the install.
    $port = 8766
    $py = (Get-Command python -CommandType Application | Select-Object -First 1).Source
    $server = Start-Process $py -ArgumentList '-m', 'http.server', "$port", '--bind', '127.0.0.1', '--directory', "`"$ghrel`"" -PassThru -WindowStyle Hidden
    $up = $false
    foreach ($i in 1..30) {
        try { Invoke-WebRequest "http://127.0.0.1:$port/$ghArchive" -UseBasicParsing -TimeoutSec 2 | Out-Null; $up = $true; break } catch { Start-Sleep -Seconds 1 }
    }
    if (-not $up) { throw 'test server did not start' }
    $d = Run 'override' @{ MISOKU_GH_BASE_URL = "http://127.0.0.1:$port" }
    Expect 'hash override over the network: fails' ($code -ne 0)
    Expect 'hash override over the network: pinned hash used' (Says "checksum mismatch for $ghArchive")
    Expect 'hash override over the network: nothing installed' (-not (Test-Path $d))
    Stop-Process -Id $server.Id -ErrorAction SilentlyContinue
    $server = $null

    # 8a. A file:// URL to a network share (UNC) is a download from another
    # computer: refused, and the hash override never applies to it. The
    # runner's admin share C$ stands in for a remote one.
    $unc = 'file://127.0.0.1/' + ($ghrel -replace '^([A-Za-z]):', '$1$$' -replace '\\', '/')
    $d = Run 'unc' @{ MISOKU_GH_BASE_URL = $unc }
    Expect 'gh from a network share: fails' ($code -ne 0)
    Expect 'gh from a network share: refused or pinned hash used' ((Says 'not a network share') -or (Says "checksum mismatch for $ghArchive"))
    Expect 'gh from a network share: nothing installed' (-not (Test-Path $d))

    # 8b. The same share hidden behind a percent-encoded backslash, which
    # .NET does not see as UNC but decodes to one: refused too.
    $d = Run 'uncenc' @{ MISOKU_GH_BASE_URL = ('file:///%5C127.0.0.1/' + ($ghrel -replace '^([A-Za-z]):', '$1$$' -replace '\\', '/')) }
    Expect 'gh from an encoded network share: fails' ($code -ne 0)
    Expect 'gh from an encoded network share: refused' (Says 'not a network share')
    Expect 'gh from an encoded network share: nothing installed' (-not (Test-Path $d))

    # 9. No git, winget present: git is installed with winget, and the
    # install finishes either way.
    $d = Run 'winget' @{ PATH = "$fakes\winget-ok;$sys" }
    Expect 'winget: exits 0' ($code -eq 0)
    Expect 'winget: git installed with winget' ((Test-Path "$work\log-winget.txt") -and ((Text "$work\log-winget.txt") -eq 'winget install --id Git.Git -e --source winget --accept-package-agreements --accept-source-agreements'))
    Expect 'winget: says so' (Says 'installing it with winget')
    Expect 'winget: misoku installed' (Test-Path "$d\misoku.exe")
    $d = Run 'wingetfail' @{ PATH = "$fakes\winget-fail;$sys" }
    Expect 'winget fails: still exits 0' ($code -eq 0)
    Expect 'winget fails: says so' (Says 'git could not be installed')
    Expect 'winget fails: misoku installed' (Test-Path "$d\misoku.exe")

    # 10. git in Program Files but not on this window's PATH: nothing to do.
    $d = Run 'pfgit' @{ PATH = "$fakes\winget-ok;$sys"; ProgramFiles = "$fakes\pf-git" }
    Expect 'git in Program Files: exits 0' ($code -eq 0)
    Expect 'git in Program Files: winget not run' (-not (Test-Path "$work\log-pfgit.txt"))
    Expect 'git in Program Files: no notice' (-not (Says 'git is not installed'))

    # 11. Pasted as `irm ... | iex`.
    $d = Run 'piped' @{} -Piped
    Expect 'piped into iex: exits 0' ($code -eq 0)
    Expect 'piped into iex: misoku installed' ((& "$d\misoku.exe") -eq 'misoku-new')
    Expect 'piped into iex: gh installed' ((& "$d\gh.exe") -eq "gh-$ghv")
    $d = Run 'pipedbad' @{ MISOKU_GH_SHA256 = $null } -Piped
    Expect 'piped into iex, bad gh: says so' (Says "checksum mismatch for $ghArchive")
    Expect 'piped into iex, bad gh: nothing installed' (-not (Test-Path $d))

    # 12. The temporary folder is removed after a success and a failure.
    New-Item -ItemType Directory -Path "$work\tmp-ok", "$work\tmp-fail" | Out-Null
    $null = Run 'tmpok' @{ TMP = "$work\tmp-ok"; TEMP = "$work\tmp-ok" }
    # Only the installer's own folders count: PowerShell may leave its
    # __PSScriptPolicyTest_* files there.
    Expect 'tmp: success leaves nothing' (-not (Get-ChildItem -Force "$work\tmp-ok" -Filter 'misoku-*'))
    $null = Run 'tmpfail' @{ TMP = "$work\tmp-fail"; TEMP = "$work\tmp-fail"; MISOKU_GH_SHA256 = $null }
    Expect 'tmp: failure exits non-zero' ($code -ne 0)
    Expect 'tmp: failure leaves nothing' (-not (Get-ChildItem -Force "$work\tmp-fail" -Filter 'misoku-*'))
} finally {
    if ($server) { Stop-Process -Id $server.Id -ErrorAction SilentlyContinue }
    [Environment]::SetEnvironmentVariable('Path', $savedUserPath, 'User')
    Remove-Item -LiteralPath $work -Recurse -Force -ErrorAction SilentlyContinue
}

if ($script:fails -ne 0) { throw "$($script:fails) install check(s) failed" }
Write-Host "install.ps1 checks OK ($shellExe)"
# The last native command was an install that was meant to fail; without
# this its exit code would be the script's, and CI's pwsh step would fail.
exit 0
