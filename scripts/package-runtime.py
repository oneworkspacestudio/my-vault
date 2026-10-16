"""Build deterministic additive branding/runtime payload; never alter crypto JS."""
import argparse
import hashlib
import json
import pathlib
import shutil
import zipfile

p=argparse.ArgumentParser()
p.add_argument('--engine',required=True)
p.add_argument('--web',required=True)
p.add_argument('--stage',required=True)
p.add_argument('--build-receipt',required=True)
a=p.parse_args()
root=pathlib.Path(__file__).resolve().parents[1]
lock=json.loads((root/'upstream.lock.json').read_text())
receipt=json.loads(pathlib.Path(a.build_receipt).read_text(encoding='utf-8-sig'))
engine=pathlib.Path(a.engine)
sha=lambda data:hashlib.sha256(data).hexdigest()
if receipt['commit']!=lock['engine']['commit'] or receipt['sha256']!=sha(engine.read_bytes()) or receipt['cargoLockSHA256']!=lock['engine']['cargoLockSHA256']:
    raise SystemExit('Engine does not match its reviewed native source-build receipt')
if engine.read_bytes()[:2]!=b'MZ':
    raise SystemExit('Expected native Windows PE engine')
stage=pathlib.Path(a.stage)
if stage.exists():
    raise SystemExit('Use a fresh, dedicated payload staging folder')
stage.mkdir(parents=True)
shutil.copy2(engine,stage/'vaultwarden.exe')
web=stage/'web-vault'
shutil.copytree(a.web,web)
before={f.relative_to(web).as_posix():sha(f.read_bytes()) for f in web.rglob('*.js')}
index=web/'index.html'
html=index.read_text(encoding='utf-8')
html=html.replace('<title page-title>Vaultwarden Web</title>','<title page-title>My Vault</title>')
html=html.replace('</head>','<link rel="stylesheet" href="my-vault.css"/><link rel="icon" type="image/svg+xml" href="my-vault.svg"/></head>')
html=html.replace('</body>','<a class="myvault-recovery-link" href="_myvault/recovery">Recovery &amp; Portability ↗</a></body>')
index.write_text(html,encoding='utf-8')
shutil.copy2(root/'web/my-vault.svg',web/'my-vault.svg')
css='''/* My Vault additive presentation. No client cryptography is modified. */
:root{--color-primary-500:101 80 204;--color-primary-600:87 65 184;--color-primary-700:74 54 162}
body{font-family:system-ui,-apple-system,"Segoe UI",sans-serif}
bit-svg[aria-label="Bitwarden logo"]{background:url("my-vault.svg") left center/48px 48px no-repeat!important;min-width:210px;min-height:56px;align-items:center;padding-left:60px}
bit-svg[aria-label="Bitwarden logo"] svg{display:none!important}
bit-svg[aria-label="Bitwarden logo"]::after{content:"My Vault";font:700 25px system-ui;color:#34266a;white-space:nowrap}
button[type="submit"]{background-color:#6550cc!important;border-color:#6550cc!important;color:white!important}
.myvault-recovery-link{position:fixed;right:20px;bottom:18px;z-index:100;background:#fff;border:1px solid #dfd9f5;border-radius:10px;padding:10px 16px;color:#6550cc;font:600 12px system-ui;text-decoration:none;box-shadow:0 3px 18px #2221}
.myvault-guide{max-width:850px;margin:45px auto;padding:30px;color:#17233b;line-height:1.8}.myvault-guide h1{font-size:36px;letter-spacing:-1.3px}.myvault-guide h2{margin-top:35px}.myvault-guide code{background:#f1eefb;padding:3px 5px;border-radius:4px}.myvault-guide p{color:#64718a}
'''
(web/'my-vault.css').write_text(css,encoding='utf-8')
after={f.relative_to(web).as_posix():sha(f.read_bytes()) for f in web.rglob('*.js')}
if before!=after:raise SystemExit('Cryptographic/client JavaScript changed unexpectedly')
files={}
for file in sorted(stage.rglob('*')):
    if file.is_file():
        data=file.read_bytes()
        files[file.relative_to(stage).as_posix()]={'size':len(data),'sha256':sha(data)}
payload=root/'internal/payload'
with zipfile.ZipFile(payload/'runtime.zip','w',compression=zipfile.ZIP_DEFLATED,compresslevel=9) as z:
    for name in files:
        info=zipfile.ZipInfo(name,date_time=(2026,10,16,0,0,0));info.compress_type=zipfile.ZIP_DEFLATED
        z.writestr(info,(stage/name).read_bytes())
manifest={'schema':1,'engine':lock['engine']['tag'],'web':lock['web']['tag'],'files':files}
(payload/'manifest.json').write_text(json.dumps(manifest,sort_keys=True,separators=(',',':')),encoding='utf-8')
print(json.dumps({'files':len(files),'engineSha256':files['vaultwarden.exe']['sha256'],'clientJavaScriptUnchanged':True}))
