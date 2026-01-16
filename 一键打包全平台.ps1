$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $root

$dist = Join-Path $root "release"
if (Test-Path $dist) {
    Remove-Item -Recurse -Force $dist
}
New-Item -ItemType Directory -Path $dist | Out-Null

$targets = @(
    @{ os = "windows"; arch = "amd64" },
    @{ os = "windows"; arch = "386" },
    @{ os = "linux";   arch = "amd64" },
    @{ os = "linux";   arch = "386" },
    @{ os = "darwin";  arch = "amd64" },
    @{ os = "darwin";  arch = "arm64" }
)

$commands = @(
    @{ name = "router-center"; path = "./cmd/router-center" },
    @{ name = "router-client"; path = "./cmd/router-client" }
)

$env:CGO_ENABLED = "0"

foreach ($t in $targets) {
    $os = $t.os
    $arch = $t.arch

    $outDir = Join-Path $dist "$os-$arch"
    New-Item -ItemType Directory -Path $outDir | Out-Null

    foreach ($cmd in $commands) {
        $binName = $cmd.name
        if ($os -eq "windows") {
            $binName = "$binName.exe"
        }

        $outFile = Join-Path $outDir $binName
        $env:GOOS = $os
        $env:GOARCH = $arch

        Write-Host "Building $($cmd.name) for $os/$arch ..."
        go build -trimpath -ldflags "-s -w" -o $outFile $cmd.path
    }
}

Write-Host "Build complete. Output: $dist"
