(function () {
  function cellText(td) {
    if (!td) return "";
    return String(td.textContent || "").trim();
  }

  function cmpStrings(a, b, asc) {
    var na = Number(a);
    var nb = Number(b);
    var useNum =
      a !== "" &&
      b !== "" &&
      !isNaN(na) &&
      !isNaN(nb) &&
      isFinite(na) &&
      isFinite(nb);
    var c = useNum ? na - nb : a.localeCompare(b, undefined, { numeric: true, sensitivity: "base" });
    return asc ? c : -c;
  }

  function rowMatches(tr, colIdx, op, val) {
    var cells = tr.querySelectorAll("td");
    if (colIdx < 0 || colIdx >= cells.length) return true;
    var raw = cellText(cells[colIdx]);
    if (op === "eq") return raw === val;
    if (op === "contains") return raw.indexOf(val) !== -1;
    if (op === "prefix") return raw.indexOf(val) === 0;
    if (op === "suffix") return val === "" || raw.lastIndexOf(val) === raw.length - val.length;
    return true;
  }

  function init(table) {
    if (!table || table.getAttribute("data-sql-enhanced") === "1") return;
    table.setAttribute("data-sql-enhanced", "1");
    var tbody = table.querySelector("tbody");
    if (!tbody) return;
    var ths = table.querySelectorAll("thead th");
    if (!ths.length) return;

    var selCol = document.getElementById("sql-filter-col");
    var selOp = document.getElementById("sql-filter-op");
    var inpVal = document.getElementById("sql-filter-val");
    var btnApply = document.getElementById("sql-filter-apply");
    var btnClear = document.getElementById("sql-filter-clear");

    if (selCol) {
      selCol.innerHTML = "";
      var opt0 = document.createElement("option");
      opt0.value = "";
      opt0.textContent = "（不筛选）";
      selCol.appendChild(opt0);
      for (var i = 0; i < ths.length; i++) {
        var o = document.createElement("option");
        o.value = String(i);
        o.textContent = cellText(ths[i]) || "col" + i;
        selCol.appendChild(o);
      }
      var tb = document.getElementById("sql-result-toolbar");
      if (window.MiniDBACustomSelect && tb) {
        window.MiniDBACustomSelect.refresh(tb);
      }
    }

    var sortState = { idx: -1, asc: true };

    function markSortHeaders() {
      ths.forEach(function (th, i) {
        th.removeAttribute("data-sort-dir");
        var mark = th.querySelector(".data-table__sort-mark");
        if (mark) mark.textContent = "";
        if (sortState.idx === i) {
          th.setAttribute("data-sort-dir", sortState.asc ? "asc" : "desc");
          if (mark) mark.textContent = sortState.asc ? " ↑" : " ↓";
        }
      });
    }

    function allRows() {
      return Array.prototype.slice.call(tbody.querySelectorAll("tr"));
    }

    function applyFilter() {
      var colIdx = selCol && selCol.value !== "" ? parseInt(selCol.value, 10) : -1;
      var op = selOp ? selOp.value : "contains";
      var val = inpVal ? inpVal.value.trim() : "";
      allRows().forEach(function (tr) {
        var ok = colIdx < 0 || val === "" || rowMatches(tr, colIdx, op, val);
        tr.style.display = ok ? "" : "none";
      });
    }

    function sortRows() {
      if (sortState.idx < 0) return;
      var idx = sortState.idx;
      var asc = sortState.asc;
      var rows = allRows();
      var vis = rows.filter(function (tr) {
        return tr.style.display !== "none";
      });
      var hid = rows.filter(function (tr) {
        return tr.style.display === "none";
      });
      vis.sort(function (ra, rb) {
        var a = cellText(ra.cells[idx]);
        var b = cellText(rb.cells[idx]);
        return cmpStrings(a, b, asc);
      });
      vis.concat(hid).forEach(function (tr) {
        tbody.appendChild(tr);
      });
    }

    for (var j = 0; j < ths.length; j++) {
      (function (colIndex) {
        ths[colIndex].classList.add("data-table__th-sort");
        var mark = document.createElement("span");
        mark.className = "data-table__sort-mark";
        ths[colIndex].appendChild(mark);
        ths[colIndex].addEventListener("click", function (e) {
          if (e.target.closest(".col-resize-handle")) return;
          e.preventDefault();
          if (sortState.idx === colIndex) sortState.asc = !sortState.asc;
          else {
            sortState.idx = colIndex;
            sortState.asc = true;
          }
          markSortHeaders();
          sortRows();
        });
      })(j);
    }

    markSortHeaders();

    if (btnApply) btnApply.addEventListener("click", applyFilter);
    if (btnClear) {
      btnClear.addEventListener("click", function () {
        if (selCol) {
          selCol.value = "";
          selCol.dispatchEvent(new Event("change", { bubbles: true }));
        }
        if (inpVal) inpVal.value = "";
        allRows().forEach(function (tr) {
          tr.style.display = "";
        });
      });
    }
  }

  function go() {
    var t = document.querySelector("table.sql-result-table");
    if (t) init(t);
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", go);
  } else {
    go();
  }
})();
