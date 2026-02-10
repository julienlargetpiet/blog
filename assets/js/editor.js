import { EditorState } from "@codemirror/state";
import {
  EditorView,
  keymap,
  lineNumbers,
  highlightActiveLine,
  drawSelection
} from "@codemirror/view";
import { defaultKeymap } from "@codemirror/commands";
import { html } from "@codemirror/lang-html";
import { bracketMatching } from "@codemirror/language";
import { closeBrackets } from "@codemirror/autocomplete";
import { oneDark } from "@codemirror/theme-one-dark";

function createHtmlEditor({
  textareaId,
  previewId = null,
  fontSize = "16px",
  minHeight = null,
  maxHeight = null
}) {
  const textarea = document.getElementById(textareaId);
  if (!textarea) return null;

  const preview = previewId
    ? document.getElementById(previewId)
    : null;

  const sizeTheme = EditorView.theme({
    "&": {
      fontSize
    },
    ".cm-content": {
      fontFamily: "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace",
      lineHeight: "1.6",
      ...(minHeight ? { minHeight } : {})
    },
    ...(maxHeight
      ? {
          ".cm-scroller": {
            maxHeight,
            overflow: "auto"
          }
        }
      : {})
  });

  const state = EditorState.create({
    doc: textarea.value,
    extensions: [
      html(),
      lineNumbers(),
      highlightActiveLine(),
      drawSelection(),
      bracketMatching(),
      closeBrackets(),
      EditorView.lineWrapping,
      EditorState.tabSize.of(2),
      keymap.of(defaultKeymap),
      oneDark,
      sizeTheme,

      EditorView.updateListener.of(update => {
        if (!update.docChanged) return;
        const value = update.state.doc.toString();
        textarea.value = value;

        if (preview && !preview.hasAttribute("hidden")) {
          preview.innerHTML = value;
        }
      })
    ]
  });

  const view = new EditorView({
    state,
    parent: textarea.parentElement
  });

  textarea.style.display = "none";
  return view;
}

document.addEventListener("DOMContentLoaded", () => {
  // Main description editor (normal height + preview)
  window.descriptionEditor = createHtmlEditor({
    textareaId: "description-editor",
    previewId: "description-preview",
    fontSize: "16px",
    minHeight: "140px",
    maxHeight: "400px"
  });

  // Signature editor (small, compact, no preview)
  window.signatureEditor = createHtmlEditor({
    textareaId: "signature-editor",
    fontSize: "14px",
    minHeight: "48px",
    maxHeight: "80px"
  });
});



