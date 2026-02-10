(() => {
  const seen = new Set();
  const queue = [];
  let qi = 0;
  let running = 0;

  const MAX_CONCURRENT = 4;

  function enqueue(url) {
    if (seen.has(url)) return;
    seen.add(url);
    queue.push(url);
    pump();
  }

  function pump() {
    while (running < MAX_CONCURRENT && qi < queue.length) {
      const url = queue[qi++];
      running++;

      fetch(url, {
        credentials: "same-origin",
        cache: "force-cache",
        priority: "low"
      })
        .catch(() => {})
        .finally(() => {
          running--;
          pump();
        });
    }
  }

  function collectLinks(root = document) {
    const links = root.querySelectorAll("a[href]");
    for (const a of links) {
      try {
        const url = new URL(a.href, location.href);

        if (url.origin !== location.origin) continue;
        if (url.pathname.startsWith("/admin/")) continue;

        if (/\.(js|css|png|jpg|jpeg|svg|webp|woff2?)$/i.test(url.pathname)) {
          continue;
        }

        enqueue(url.href);
      } catch {}
    }
  }

  function schedule(fn, timeout = 100) {
    if ("requestIdleCallback" in window) {
      requestIdleCallback(fn, { timeout });
    } else {
      setTimeout(fn, timeout);
    }
  }

  // Initial sweep
  collectLinks();

  // Batch DOM mutations (SPA-safe, CPU-safe)
  let dirty = false;
  const mo = new MutationObserver(() => {
    if (dirty) return;
    dirty = true;

    schedule(() => {
      dirty = false;
      collectLinks();
    });
  });

  mo.observe(document.documentElement, {
    childList: true,
    subtree: true
  });

  // Final idle sweep (best-effort)
  if ("requestIdleCallback" in window) {
    requestIdleCallback(() => collectLinks(), { timeout: 2000 });
  }
})();


