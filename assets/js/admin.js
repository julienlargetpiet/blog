function updateFileName(input) {
  const label = document.getElementById("dataset-file-name");
  if (input.files && input.files.length > 0) {
    label.textContent = input.files[0].name;
  } else {
    label.textContent = "No file selected";
  }
}

function updatePreview() {
  if (!window.descriptionEditor) return;

  const preview = document.getElementById("description-preview");
  if (!preview) return;

  preview.innerHTML = window.descriptionEditor.state.doc.toString();
}

function togglePreview() {
  const preview = document.getElementById("description-preview");
  if (!preview) return;

  const isVisible = !preview.hasAttribute("hidden");

  if (isVisible) {
    preview.setAttribute("hidden", "");
  } else {
    updatePreview();
    preview.removeAttribute("hidden");
  }
}

window.addEventListener("DOMContentLoaded", () => {
  const textarea = document.getElementById("editor");
  if (!textarea || !window.CodeMirror6) return;

  const parent = textarea.parentElement;

  // Create CM6 editor
  const view = new window.CodeMirror6.EditorView({
    doc: textarea.value,
    extensions: [
      window.CodeMirror6.basicSetup,
      window.CodeMirror6.html()
    ],
    parent
  });

  // Hide textarea (but keep it for form submit)
  textarea.style.display = "none";

  // Sync CM → textarea on submit
  textarea.form.addEventListener("submit", () => {
    textarea.value = view.state.doc.toString();
  });

  // Expose globally for preview
  window.cmView = view;
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

