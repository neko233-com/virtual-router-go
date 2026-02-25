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
    @{ os = "windows"; arch = "arm64" },
    @{ os = "linux";   arch = "amd64" },
    @{ os = "linux";   arch = "arm64" },
    @{ os = "darwin";  arch = "amd64" },
    @{ os = "darwin";  arch = "arm64" }
)

$serverBuild = @{ name = "virtual-router-server"; path = "./cmd/router-center" }

$serverConfigFile = Join-Path $root "neko233-router-server.json"
if (-not (Test-Path $serverConfigFile)) {
    throw "默认配置文件不存在: $serverConfigFile"
}

$devopsScriptFile = Join-Path $root "devops_for_virtual_router_server.sh"
if (-not (Test-Path $devopsScriptFile)) {
    throw "运维脚本不存在: $devopsScriptFile"
}

$env:CGO_ENABLED = "0"

foreach ($t in $targets) {
    $os = $t.os
    $arch = $t.arch

    $outDir = Join-Path $dist "$os-$arch"
    New-Item -ItemType Directory -Path $outDir | Out-Null

    $binName = $serverBuild.name
    if ($os -eq "windows") {
        $binName = "$binName.exe"
    }

    $outFile = Join-Path $outDir $binName
    $env:GOOS = $os
    $env:GOARCH = $arch

    Write-Host "Building $($serverBuild.name) for $os/$arch ..."
    go build -trimpath -ldflags "-s -w" -o $outFile $serverBuild.path

    Copy-Item -Path $serverConfigFile -Destination (Join-Path $outDir "neko233-router-server.json") -Force
    if ($os -eq "linux" -or $os -eq "darwin") {
        Copy-Item -Path $devopsScriptFile -Destination (Join-Path $outDir "devops_for_virtual_router_server.sh") -Force
    }
}

Write-Host "Build complete. Output: $dist"
