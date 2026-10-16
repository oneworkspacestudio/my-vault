param(
 [Parameter(Mandatory)][string]$Engine,
 [Parameter(Mandatory)][string]$EngineReceipt,
 [Parameter(Mandatory)][string]$WebArchive,
 [Parameter(Mandatory)][string]$Cache,
 [string]$Version='0.2.0-rc.2',
 [string]$Go='go',
 [string]$Python='python'
)
$ErrorActionPreference='Stop'
$root=Split-Path $PSScriptRoot -Parent
$lock=Get-Content (Join-Path $root 'upstream.lock.json') -Raw|ConvertFrom-Json
if((Get-FileHash $WebArchive).Hash.ToLowerInvariant() -cne $lock.web.sha256){throw 'Compatible web vault archive does not match the pinned upstream hash'}
$build=Join-Path $Cache ('payload-'+[guid]::NewGuid().ToString('N'))
$web=Join-Path $build 'downloaded-web'
New-Item -ItemType Directory -Force $web|Out-Null
& tar -xf $WebArchive -C $web
if($LASTEXITCODE){throw 'Web extraction failed'}
$webRoot=Join-Path $web 'web-vault'
& $Python (Join-Path $PSScriptRoot 'package-runtime.py') --engine $Engine --web $webRoot --stage (Join-Path $build 'runtime') --build-receipt $EngineReceipt
if($LASTEXITCODE){throw 'Runtime verification/branding failed'}
Push-Location $root
try{
 New-Item -ItemType Directory -Force dist|Out-Null
 & $Go run github.com/akavel/rsrc@v0.10.2 -arch amd64 -ico web/my-vault.ico -o cmd/myvault/rsrc_windows_amd64.syso
 if($LASTEXITCODE){throw 'Windows icon resource generation failed'}
 & $Go build -trimpath -buildvcs=false -ldflags "-s -w -X main.version=$Version" -o dist/MyVault.exe ./cmd/myvault
 if($LASTEXITCODE){throw 'My Vault build failed'}
 Copy-Item $EngineReceipt dist/engine-build-receipt.json -Force
 Copy-Item upstream.lock.json,LICENSE,NOTICE,README.md,THIRD-PARTY-LICENSES.txt -Destination dist -Force
 Copy-Item docs -Destination dist -Recurse -Force
 Write-Output (Join-Path $root 'dist/MyVault.exe')
}finally{Pop-Location}
