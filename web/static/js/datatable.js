(function () {
  var STORAGE_PREFIX = "minidba.dt.cols.";
  var MIN_W = 48;
  var MAX_W = 300;
  var CHAR_W = 7;

  function debounce(fn, ms) {
    var t;
    return function () {
      clearTimeout(t);
      var args = arguments;
      t = setTimeout(function () {
        fn.apply(null, args);
      }, ms);
    };
  }

  function estimateWidth(ths, rows, colIdx) {
    var maxLen = 0;
    var headerText = ths[colIdx] ? (ths[colIdx].textContent || "").trim() : "";
    maxLen = Math.max(maxLen, headerText.length);
    for (var r = 0; r < Math.min(rows.length, 3); r++) {
      var cells = rows[r].querySelectorAll("td");
      if (colIdx < cells.length) {
        var text = (cells[colIdx].textContent || "").trim();
        maxLen = Math.max(maxLen, text.length);
      }
    }
    var w = maxLen * CHAR_W + 24;
    return Math.max(MIN_W, Math.min(MAX_W, w));
  }

  function initTable(table) {
    var key = table.getAttribute("data-dt-key");
    if (!key) return;
    var ths = table.querySelectorAll("thead th");
    if (!ths.length) return;
    if (table.getAttribute("data-dt-enhanced") === "1") return;
    table.setAttribute("data-dt-enhanced", "1");
    table.style.tableLayout = "fixed";

    var colgroup = table.querySelector("colgroup");
    var cols = colgroup ? colgroup.querySelectorAll("col") : [];
    if (!colgroup || cols.length !== ths.length) {
      colgroup = document.createElement("colgroup");
      cols = [];
      for (var i = 0; i < ths.length; i++) {
        var c = document.createElement("col");
        colgroup.appendChild(c);
        cols.push(c);
      }
      var thead = table.querySelector("thead");
      table.insertBefore(colgroup, thead);
    }

    var storageKey = STORAGE_PREFIX + key;
    var widths = null;
    try {
      var raw = localStorage.getItem(storageKey);
      if (raw) widths = JSON.parse(raw);
    } catch (e) {}
    if (widths && widths.length === cols.length) {
      for (var j = 0; j < cols.length; j++) {
        if (widths[j] >= MIN_W) cols[j].style.width = widths[j] + "px";
      }
    } else {
      var rows = table.querySelectorAll("tbody tr");
      for (var cIdx = 0; cIdx < cols.length; cIdx++) {
        cols[cIdx].style.width = estimateWidth(ths, rows, cIdx) + "px";
      }
    }

    var save = debounce(function () {
      var arr = [];
      for (var k = 0; k < cols.length; k++) {
        var w = cols[k].offsetWidth || 0;
        arr.push(w);
      }
      try {
        localStorage.setItem(storageKey, JSON.stringify(arr));
      } catch (e) {}
    }, 150);

    for (var i = 0; i < ths.length; i++) {
      (function (idx) {
        var th = ths[idx];
        if (th.querySelector(".col-resize-handle")) return;
        var h = document.createElement("span");
        h.className = "col-resize-handle";
        h.setAttribute("aria-hidden", "true");
        th.style.position = "relative";
        th.appendChild(h);
        var startX;
        var startW;
        h.addEventListener("click", function (e) {
          e.stopPropagation();
        });
        h.addEventListener("mousedown", function (e) {
          e.preventDefault();
          e.stopPropagation();
          startX = e.pageX;
          startW = cols[idx].offsetWidth || th.offsetWidth || MIN_W;
          function mm(ev) {
            var nw = Math.max(MIN_W, startW + (ev.pageX - startX));
            cols[idx].style.width = nw + "px";
          }
          function mu() {
            document.removeEventListener("mousemove", mm);
            document.removeEventListener("mouseup", mu);
            save();
          }
          document.addEventListener("mousemove", mm);
          document.addEventListener("mouseup", mu);
        });
      })(i);
    }
  }

  function scan() {
    document.querySelectorAll("table.data-table[data-dt-key]").forEach(initTable);
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", scan);
  } else {
    scan();
  }
})();
