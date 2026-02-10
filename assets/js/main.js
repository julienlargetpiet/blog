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





