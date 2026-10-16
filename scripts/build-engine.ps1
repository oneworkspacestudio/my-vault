param(
 [Parameter(Mandatory)][string]$Cache,
 [Parameter(Mandatory)][string]$Cargo,
 [Parameter(Mandatory)][string]$Perl,
 [string]$VCVars='C:/Program Files/Microsoft Visual Studio/2022/Community/VC/Auxiliary/Build/vcvars64.bat',
 [string]$TargetDir=''
)
$ErrorActionPreference='Stop'
$root=Split-Path $PSScriptRoot -Parent
$lock=Get-Content (Join-Path $root 'upstream.lock.json') -Raw|ConvertFrom-Json
if(!$TargetDir){$TargetDir=Join-Path $Cache 'target'}
New-Item -ItemType Directory -Force $Cache,$TargetDir|Out-Null
$source=Join-Path $Cache 'vaultwarden-source'
if(!(Test-Path $source)){
 & git clone --depth 1 --branch $lock.engine.tag $lock.engine.repository $source
 if($LASTEXITCODE){throw 'Upstream source clone failed'}
}
$revision=& git -c "safe.directory=$source" -C $source rev-parse HEAD
if($revision -cne $lock.engine.commit){throw 'Upstream commit mismatch'}
$dirty=@(& git -c "safe.directory=$source" -C $source diff --name-only)
if(@($dirty|Where-Object {$_ -ne 'Cargo.lock'}).Count){throw 'Upstream application source has local modifications'}
$securityLock=Join-Path $root 'upstream/Cargo.lock'
if((Get-FileHash $securityLock).Hash.ToLowerInvariant() -cne $lock.engine.cargoLockSHA256){throw 'Reviewed dependency lock mismatch'}
Copy-Item $securityLock (Join-Path $source 'Cargo.lock') -Force
if(!(Test-Path $Cargo) -or !(Test-Path $Perl) -or !(Test-Path $VCVars)){throw 'Native Rust/MSVC/Perl build dependencies are required'}
# Import the developer environment privately. Never print a captured environment.
$lines=& $env:ComSpec /d /s /c "`"`"$VCVars`" >nul && set`""
if($LASTEXITCODE){throw 'MSVC environment initialization failed'}
foreach($line in $lines){$at=$line.IndexOf('=');if($at -gt 0){[Environment]::SetEnvironmentVariable($line.Substring(0,$at),$line.Substring($at+1),'Process')}}
$env:OPENSSL_SRC_PERL=[IO.Path]::GetFullPath($Perl)
$env:PATH=(Split-Path $Perl)+';'+(Split-Path $Cargo)+';'+$env:PATH
$env:CARGO_TARGET_DIR=[IO.Path]::GetFullPath($TargetDir)
$env:RUSTFLAGS='-C target-feature=+crt-static'
$env:VW_VERSION=$lock.engine.tag
$env:CARGO_BUILD_JOBS='4'
$env:LC_ALL='C';$env:LANG='C'
Push-Location $source
try{
 & $Cargo build --locked --release --target x86_64-pc-windows-msvc --features sqlite,vendored_openssl
 if($LASTEXITCODE){throw 'Native Vaultwarden build failed'}
 $engine=Join-Path $TargetDir 'x86_64-pc-windows-msvc/release/vaultwarden.exe'
 $compiler=& (Join-Path (Split-Path $Cargo) 'rustc.exe') --version
 $receipt=@{commit=$revision;tag=$lock.engine.tag;sha256=(Get-FileHash $engine).Hash.ToLowerInvariant();cargoLockSHA256=(Get-FileHash (Join-Path $source 'Cargo.lock')).Hash.ToLowerInvariant();compiler=$compiler;target='x86_64-pc-windows-msvc';staticCRT=$true;features=@('sqlite','vendored_openssl')}
 $receipt|ConvertTo-Json -Depth 6|Set-Content (Join-Path $Cache 'engine-build-receipt.json') -Encoding utf8NoBOM
 Write-Output $engine
}finally{Pop-Location}
