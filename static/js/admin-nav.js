// admin-nav.js — renders an admin navigation bar below the page banner when the
// visitor has an authenticated session. The menu is built entirely on the client so
// that cached, auth-agnostic page HTML (e.g. the home page) is never polluted.
(function () {
  async function getSession() {
    try {
      const res = await fetch('/api/auth/session', { credentials: 'same-origin' });
      if (!res.ok) return { authenticated: false };
      return await res.json();
    } catch (e) {
      return { authenticated: false };
    }
  }

  function buildNav(session) {
    const nav = document.createElement('nav');
    nav.className = 'admin-nav';

    const links = document.createElement('div');
    links.className = 'admin-nav-links';

    const items = [
      { label: 'Certificates', href: '/certificates' },
      { label: 'Add Certificate', href: '/certificates#new' },
      { label: 'Backup', href: '/backup' },
      { label: 'Restore', href: '/restore' },
      { label: 'Change Password', href: '/password' },
      { label: 'Admin', href: '/admin' },
    ];
    const current = window.location.pathname;
    items.forEach(function (it) {
      const a = document.createElement('a');
      a.href = it.href;
      a.textContent = it.label;
      a.className = 'admin-nav-link';
      if (it.href.split('#')[0] === current) a.classList.add('active');
      links.appendChild(a);
    });

    const right = document.createElement('div');
    right.className = 'admin-nav-user';

    const who = document.createElement('span');
    who.className = 'admin-nav-whoami';
    const username = session.user && session.user.username ? session.user.username : 'admin';
    who.textContent = 'Signed in as ' + username;
    right.appendChild(who);

    const logout = document.createElement('button');
    logout.type = 'button';
    logout.className = 'admin-nav-logout';
    logout.textContent = 'Log out';
    logout.addEventListener('click', async function () {
      try {
        await fetch('/api/auth/logout', { method: 'POST', credentials: 'same-origin' });
      } catch (e) {
        /* ignore network errors and redirect anyway */
      }
      window.location.href = '/';
    });
    right.appendChild(logout);

    nav.appendChild(links);
    nav.appendChild(right);
    return nav;
  }

  async function init() {
    const session = await getSession();
    if (!session || !session.authenticated) return;
    const header = document.querySelector('header');
    if (!header) return;
    header.insertAdjacentElement('afterend', buildNav(session));
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
