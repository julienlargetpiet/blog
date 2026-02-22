function initThemeToggle() {
  const root = document.documentElement;
  const btn = document.getElementById("theme-toggle");
  if (!btn) return;

  btn.textContent = root.dataset.theme === "dark" ? "☀︎" : "⏾";

  btn.addEventListener("click", () => {
    const isDark = root.dataset.theme === "dark";
    root.dataset.theme = isDark ? "light" : "dark";
    localStorage.setItem("theme", root.dataset.theme);
    btn.textContent = isDark ? "⏾" : "☀︎";
  });
}

document.addEventListener("DOMContentLoaded", () => {

  document.addEventListener("click", async (event) => {
    const btn = event.target.closest(".copy-btn");
    if (!btn) return;
  
    const pre = btn.closest("pre");
    if (!pre) return;
  
    const code = pre.querySelector("code");
    if (!code) return;
  
    const text = code.innerText.trim();
  
    try {
      await navigator.clipboard.writeText(text);
  
      // Visual feedback
      const original = btn.textContent;
      btn.textContent = "✅ Copied";
      btn.disabled = true;
  
      setTimeout(() => {
        btn.textContent = original;
        btn.disabled = false;
      }, 1500);
    } catch (err) {
      console.error("Copy failed", err);
      btn.textContent = "❌ Error";
    }
  });

  initThemeToggle();

});


(function () {
  const top = document.querySelector('.admin-table-scroll-top');
  const topInner = document.querySelector('.admin-table-scroll-inner');
  const wrapper = document.querySelector('.admin-table-wrapper');
  const table = wrapper?.querySelector('.admin-table');

  if (!top || !topInner || !wrapper || !table) return;

  function updateWidth() {
    topInner.style.width = table.scrollWidth + 'px';
  }

  let syncing = false;

  top.addEventListener('scroll', () => {
    if (syncing) return;
    syncing = true;
    wrapper.scrollLeft = top.scrollLeft;
    syncing = false;
  });

  wrapper.addEventListener('scroll', () => {
    if (syncing) return;
    syncing = true;
    top.scrollLeft = wrapper.scrollLeft;
    syncing = false;
  });

  updateWidth();
  window.addEventListener('resize', updateWidth);

  if ('ResizeObserver' in window) {
    new ResizeObserver(updateWidth).observe(table);
  }
})();


document.addEventListener("DOMContentLoaded", function () {
  const summary = document.getElementById("article-summary");
  const toggle = document.getElementById("summary-toggle");

  if (summary && toggle) {
    toggle.addEventListener("click", function () {
      summary.classList.toggle("open");
    });
  }
});

document.addEventListener("DOMContentLoaded", function () {
  initSummary();
});

function initSummary() {
  const content = document.querySelector(".article-content");
  const summaryNav = document.getElementById("summary-content");
  const summaryPanel = document.getElementById("article-summary");

  if (!content || !summaryNav || !summaryPanel) return;

  const headings = content.querySelectorAll("h2, h3");

  if (headings.length < 2) return;

  headings.forEach((heading) => {
    const id = heading.id || slugify(heading.textContent);
    heading.id = id;

    const link = document.createElement("a");
    link.href = "#" + id;
    link.textContent = heading.textContent;

    if (heading.tagName === "H3") {
      link.style.paddingLeft = "1rem";
    }

    link.addEventListener("click", () => {
      if (window.innerWidth <= 1100) {
        summaryPanel.classList.remove("open");
      }
    });

    summaryNav.appendChild(link);
  });

  // 🔥 Auto-open on large screens
  if (window.innerWidth >= 1100) {
    summaryPanel.classList.add("open");
  }
}

function slugify(text) {
  return text
    .toLowerCase()
    .trim()
    .replace(/[^\w\s-]/g, "")
    .replace(/\s+/g, "")
    .replace(/\s+/g, "-");
}

document.addEventListener("DOMContentLoaded", function () {
  initReslugFlash();
});

function initReslugFlash() {
  const form = document.querySelector('form[action="/admin/reslug"]');
  const flash = document.getElementById("flash-message");

  if (!form || !flash) return;

  form.addEventListener("submit", function (e) {

      e.preventDefault(); // ⬅️ stop immediate submission

      showFlashMessage("Build All to see changes");

      setTimeout(() => {
        form.submit(); // submit after delay
      }, 3000);

  });
}

function showFlashMessage(message) {
  const flash = document.getElementById("flash-message");
  if (!flash) return;

  flash.textContent = message;
  flash.classList.remove("hidden");

  setTimeout(() => {
    flash.classList.add("hidden");
  }, 3000);
}

document.addEventListener("DOMContentLoaded", function () {
  const progressBar = document.querySelector(".scroll-progress-bar");
  if (!progressBar) return;

  let ticking = false;

  function updateScrollProgress() {
    const scrollTop = window.scrollY;
    const docHeight =
      document.documentElement.scrollHeight - window.innerHeight;

    const progress = docHeight > 0
      ? scrollTop / docHeight
      : 0;

    progressBar.style.transform = `scaleX(${progress})`;
    ticking = false;
  }

  function onScroll() {
    if (!ticking) {
      requestAnimationFrame(updateScrollProgress);
      ticking = true;
    }
  }

  window.addEventListener("scroll", onScroll, { passive: true });
  window.addEventListener("resize", onScroll);

  updateScrollProgress();
});


