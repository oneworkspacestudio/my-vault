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
html=html.replace('<img class="new-logo-themed !tw-w-[200px]" alt="Vaultwarden"/>','<img src="my-vault.svg" width="64" height="64" alt="My Vault"/>')
css=(root/'web/client-light-tokens.css').read_bytes()+b'\n'+(root/'web/client-theme.css').read_bytes()
ui=(root/'web/client-theme.js').read_bytes()
css_name='my-vault.'+sha(css)[:16]+'.css'
ui_name='my-vault-ui.'+sha(ui)[:16]+'.js'
html=html.replace('</head>',f'<link rel="icon" type="image/svg+xml" href="my-vault.svg"/><script src="{ui_name}" defer></script></head>')
html=html.replace('</body>',f'<link rel="stylesheet" href="{css_name}"/></body>')
html=html.replace('</body>','<a class="myvault-recovery-link" href="_myvault/recovery">Recovery &amp; Portability ↗</a></body>')
index.write_text(html,encoding='utf-8')
connector=web/'sso-connector.html'
connector_html=connector.read_text(encoding='utf-8')
connector_html=connector_html.replace('<title>Bitwarden</title>','<title>My Vault · Signing in</title>')
connector_html=connector_html.replace('class="new-logo-themed tw-mb-4" alt="Bitwarden"','src="my-vault.svg" width="64" height="64" class="tw-mb-4" alt="My Vault"')
connector.write_text(connector_html,encoding='utf-8')
shutil.copy2(root/'web/my-vault.svg',web/'my-vault.svg')
(web/css_name).write_bytes(css)
after={f.relative_to(web).as_posix():sha(f.read_bytes()) for f in web.rglob('*.js')}
if before!=after:raise SystemExit('Cryptographic/client JavaScript changed unexpectedly')
(web/ui_name).write_bytes(ui)
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
