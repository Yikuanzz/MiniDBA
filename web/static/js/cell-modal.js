(function () {
  var modal = document.getElementById("cell-modal");
  var contentEl = document.getElementById("cell-modal-content");
  var copyBtn = document.getElementById("cell-modal-copy");
  var hintEl = document.getElementById("cell-modal-hint");

  if (!modal || !contentEl) return;

  function openModal(text) {
    contentEl.textContent = text;
    modal.hidden = false;
    document.addEventListener("keydown", onKeyDown);
    document.body.style.overflow = "hidden";
  }

  function closeModal() {
    modal.hidden = true;
    document.removeEventListener("keydown", onKeyDown);
    document.body.style.overflow = "";
  }

  function onKeyDown(e) {
    if (e.key === "Escape") {
      e.preventDefault();
      closeModal();
    }
  }

  modal.addEventListener("click", function (e) {
    if (e.target.hasAttribute("data-modal-close")) {
      closeModal();
    }
  });

  if (copyBtn) {
    copyBtn.addEventListener("click", function () {
      var text = contentEl.textContent;
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(text).then(function () {
          showHint();
        });
      } else {
        var ta = document.createElement("textarea");
        ta.value = text;
        ta.style.position = "fixed";
        ta.style.opacity = "0";
        document.body.appendChild(ta);
        ta.select();
        try {
          document.execCommand("copy");
          showHint();
        } catch (err) {}
        document.body.removeChild(ta);
      }
    });
  }

  function showHint() {
    if (!hintEl) return;
    hintEl.style.opacity = "1";
    hintEl.style.transform = "translateX(-50%) translateY(0)";
    setTimeout(function () {
      hintEl.style.opacity = "0";
      hintEl.style.transform = "translateX(-50%) translateY(8px)";
    }, 1500);
  }

  function attachCellListeners(root) {
    var tables = root.querySelectorAll
      ? root.querySelectorAll("table.data-table")
      : [];
    tables.forEach(function (table) {
      table.addEventListener("click", function (e) {
        var td = e.target.closest("tbody td");
        if (!td) return;
        openModal(td.textContent || "");
      });
    });
  }

  // Attach to existing tables
  attachCellListeners(document);

  // Observe for dynamically added tables (e.g., after HTMX or JS updates)
  if (window.MutationObserver) {
    var observer = new MutationObserver(function (mutations) {
      mutations.forEach(function (m) {
        m.addedNodes.forEach(function (node) {
          if (node.nodeType === 1) {
            attachCellListeners(node);
          }
        });
      });
    });
    observer.observe(document.body, { childList: true, subtree: true });
  }
})();
