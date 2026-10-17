/* Presentation and upstream SSO navigation only. Never read vault storage,
 * account inputs, password fields, encryption keys, tokens or client services. */
(()=>{'use strict';
  const launch=new URLSearchParams(location.search).get('workspace')==='1';
  const independent=new URLSearchParams(location.search).get('login')==='independent';
  let settings=null,attempted=false,queued=false;
  function update(){
    queued=false;
    const route=location.hash.slice(2).split(/[?\/]/)[0]||'vault';
    if(document.documentElement.dataset.myvaultRoute!==route)document.documentElement.dataset.myvaultRoute=route;
    if(route==='sso'||route==='lock'||document.querySelector('#main-content'))attempted=true;
    const title=document.title.replace(/Vaultwarden Web|Bitwarden(?: Web)?/g,'My Vault');
    if(title!==document.title)document.title=title;
    const logos=new Set(document.querySelectorAll('bit-nav-logo a,a[aria-label="Bitwarden logo"]'));
    for(const svg of document.querySelectorAll('bit-svg[aria-label="Bitwarden logo"]')){const link=svg.closest('a');if(link)logos.add(link)}
    for(const link of logos){
      if(!link.classList.contains('myvault-logo'))link.classList.add('myvault-logo');
      if(link.getAttribute('aria-label')!=='My Vault')link.setAttribute('aria-label','My Vault');
      if(link.getAttribute('title')!=='My Vault')link.setAttribute('title','My Vault');
    }
    if(route!=='login'||!settings?.ssoEnabled||independent)return;
    const form=document.querySelector('form');
    if(!form||document.querySelector('.myvault-identity'))return;
    const box=document.createElement('section');box.className='myvault-identity';box.setAttribute('aria-label','MyAccounts sign-in');
    const link=document.createElement('a');link.href=settings.ssoPath;link.textContent='Continue with MyAccounts';
    const hint=document.createElement('p');hint.textContent='Your existing MyAccounts session signs you in. Unlock your vault when asked. On your first visit, choose a master password to protect your encrypted vault.';
    const direct=document.createElement('a');direct.className='myvault-independent';direct.href=settings.independentPath;direct.textContent='Use independent vault sign-in';
    box.append(link,hint,direct);form.before(box);
    // Independent login is an explicit recovery choice, not a second identity
    // form alongside MyAccounts. No account/password inputs are inspected.
    form.classList.add('myvault-independent-form');
    // The unmodified client first reuses its own authenticated/locked session.
    // Only a fresh Workspace launch reaching login starts the standard SSO route.
    // Never retry automatically after an error/callback or override direct login.
    if(launch&&!independent&&!attempted){attempted=true;link.textContent='Signing you in with MyAccounts…';box.setAttribute('role','status');location.hash=settings.ssoPath.slice(1)}
  }
  function schedule(){if(!queued){queued=true;requestAnimationFrame(update)}}
  const observer=new MutationObserver(schedule);
  observer.observe(document.documentElement,{childList:true,subtree:true});
  addEventListener('hashchange',schedule);schedule();
  fetch(new URL('_myvault/client-config',document.baseURI),{credentials:'omit',cache:'no-store',redirect:'error'})
    .then(r=>r.ok?r.json():null).then(value=>{if(value&&value.ssoPath==='#/sso?identifier=00000000-01DC-01DC-01DC-000000000000'&&value.independentPath==='?login=independent#/login'){settings=value;schedule()}}).catch(()=>{});
})();
