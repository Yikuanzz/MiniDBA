(function () {
  var doc = document;

  function clearListPosition(list) {
    if (!list) return;
    ["position", "left", "top", "bottom", "width", "maxHeight", "zIndex", "right"].forEach(function (k) {
      list.style[k] = "";
    });
  }

  function positionList(wrap) {
    var btn = wrap.querySelector(".custom-select__btn");
    var list = wrap.querySelector(".custom-select__list");
    if (!btn || !list || list.hidden) return;
    var r = btn.getBoundingClientRect();
    var margin = 6;
    var spaceBelow = window.innerHeight - r.bottom - margin - 8;
    var spaceAbove = r.top - margin - 8;
    var want = Math.min(280, window.innerHeight * 0.4);
    var openDown = spaceBelow >= 120 || spaceBelow >= spaceAbove;
    var maxH = Math.max(80, Math.min(want, openDown ? spaceBelow : spaceAbove));
    list.style.position = "fixed";
    list.style.left = r.left + "px";
    list.style.width = r.width + "px";
    list.style.zIndex = "10000";
    list.style.maxHeight = maxH + "px";
    if (openDown) {
      list.style.top = r.bottom + margin + "px";
      list.style.bottom = "auto";
    } else {
      list.style.top = "auto";
      list.style.bottom = window.innerHeight - r.top + margin + "px";
    }
  }

  function repositionOpenLists() {
    Array.prototype.forEach.call(doc.querySelectorAll(".custom-select.is-open"), positionList);
  }

  function closeWrap(wrap) {
    if (!wrap) return;
    wrap.classList.remove("is-open");
    var btn = wrap.querySelector(".custom-select__btn");
    var list = wrap.querySelector(".custom-select__list");
    if (btn) {
      btn.setAttribute("aria-expanded", "false");
    }
    if (list) {
      list.hidden = true;
      clearListPosition(list);
    }
  }

  function closeAllExcept(except) {
    Array.prototype.forEach.call(doc.querySelectorAll(".custom-select.is-open"), function (w) {
      if (w !== except) {
        closeWrap(w);
      }
    });
  }

  function openWrap(wrap) {
    closeAllExcept(wrap);
    wrap.classList.add("is-open");
    var btn = wrap.querySelector(".custom-select__btn");
    var list = wrap.querySelector(".custom-select__list");
    if (btn) {
      btn.setAttribute("aria-expanded", "true");
    }
    if (list) {
      list.hidden = false;
      requestAnimationFrame(function () {
        requestAnimationFrame(function () {
          positionList(wrap);
        });
      });
    }
  }

  function syncFromSelect(wrap) {
    var sel = wrap.querySelector("select");
    var btn = wrap.querySelector(".custom-select__btn");
    var list = wrap.querySelector(".custom-select__list");
    if (!sel || !btn) {
      return;
    }
    var opt = sel.options[sel.selectedIndex];
    btn.textContent = opt ? opt.textContent : "";
    if (list) {
      Array.prototype.forEach.call(list.querySelectorAll("[role='option']"), function (li) {
        li.setAttribute("aria-selected", li.dataset.value === sel.value ? "true" : "false");
      });
    }
  }

  function refreshList(wrap) {
    var sel = wrap.querySelector("select");
    var list = wrap.querySelector(".custom-select__list");
    var btn = wrap.querySelector(".custom-select__btn");
    if (!sel || !list || !btn) {
      return;
    }
    list.innerHTML = "";
    Array.from(sel.options).forEach(function (opt) {
      var li = doc.createElement("li");
      li.setAttribute("role", "option");
      li.dataset.value = opt.value;
      li.textContent = opt.textContent;
      li.className = "custom-select__option";
      li.addEventListener("mousedown", function (e) {
        e.preventDefault();
      });
      li.addEventListener("click", function (e) {
        e.preventDefault();
        e.stopPropagation();
        var changed = sel.value !== opt.value;
        if (changed) {
          sel.value = opt.value;
          sel.dispatchEvent(new Event("change", { bubbles: true }));
        }
        syncFromSelect(wrap);
        closeWrap(wrap);
        btn.focus();
        if (changed && wrap.hasAttribute("data-custom-select-submit")) {
          var form = sel.form || sel.closest("form");
          if (form) {
            form.submit();
          }
        }
      });
      list.appendChild(li);
    });
    syncFromSelect(wrap);
    if (wrap.classList.contains("is-open")) {
      requestAnimationFrame(function () {
        positionList(wrap);
      });
    }
  }

  function build(wrap) {
    if (wrap.dataset.customSelectBuilt === "1") {
      refreshList(wrap);
      return;
    }
    var sel = wrap.querySelector("select");
    if (!sel) {
      return;
    }
    wrap.dataset.customSelectBuilt = "1";
    sel.classList.add("custom-select__native");
    sel.setAttribute("tabindex", "-1");
    sel.setAttribute("aria-hidden", "true");

    var btn = doc.createElement("button");
    btn.type = "button";
    btn.className = "custom-select__btn";
    btn.setAttribute("aria-haspopup", "listbox");
    btn.setAttribute("aria-expanded", "false");
    btn.setAttribute("tabindex", "0");

    var list = doc.createElement("ul");
    list.className = "custom-select__list";
    list.setAttribute("role", "listbox");
    list.hidden = true;

    wrap.insertBefore(btn, sel);
    wrap.appendChild(list);

    refreshList(wrap);

    btn.addEventListener("click", function (e) {
      e.preventDefault();
      e.stopPropagation();
      if (wrap.classList.contains("is-open")) {
        closeWrap(wrap);
      } else {
        openWrap(wrap);
      }
    });

    btn.addEventListener("keydown", function (e) {
      if (e.key === "Escape") {
        closeWrap(wrap);
      } else if (e.key === "ArrowDown" && !wrap.classList.contains("is-open")) {
        e.preventDefault();
        openWrap(wrap);
      }
    });

    sel.addEventListener("change", function () {
      syncFromSelect(wrap);
    });
  }

  function queryCustomWraps(root) {
    if (!root || root === doc || root.nodeType === 9) {
      return doc.querySelectorAll("[data-custom-select]");
    }
    if (root.nodeType === 1 && root.hasAttribute("data-custom-select")) {
      return [root];
    }
    if (root.nodeType === 1 && root.querySelectorAll) {
      return root.querySelectorAll("[data-custom-select]");
    }
    return [];
  }

  function initAll(root) {
    var nodes = queryCustomWraps(root || doc);
    Array.prototype.forEach.call(nodes, build);
  }

  function refresh(root) {
    var nodes = queryCustomWraps(root || doc);
    Array.prototype.forEach.call(nodes, function (wrap) {
      if (wrap.dataset.customSelectBuilt === "1") {
        refreshList(wrap);
      } else {
        build(wrap);
      }
    });
  }

  doc.addEventListener("mousedown", function (e) {
    if (!e.target.closest("[data-custom-select]")) {
      Array.prototype.forEach.call(doc.querySelectorAll(".custom-select.is-open"), closeWrap);
    }
  });

  doc.addEventListener("keydown", function (e) {
    if (e.key === "Escape") {
      Array.prototype.forEach.call(doc.querySelectorAll(".custom-select.is-open"), closeWrap);
    }
  });

  window.addEventListener("scroll", repositionOpenLists, true);
  window.addEventListener("resize", repositionOpenLists);

  window.MiniDBACustomSelect = { refresh: refresh, init: initAll };

  function go() {
    initAll(doc);
  }

  if (doc.readyState === "loading") {
    doc.addEventListener("DOMContentLoaded", go);
  } else {
    go();
  }
})();
