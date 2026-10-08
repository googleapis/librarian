/*
 * Copyright 2026 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Storage that tolerates sandboxed frames (embedded previews deny
// localStorage and throw on access). Preferences then last for the page.
var store = (function () {
  "use strict";
  var memory = {};
  return {
    get: function (k) {
      try { return localStorage.getItem(k); } catch (e) { return memory[k] === undefined ? null : memory[k]; }
    },
    set: function (k, v) {
      try { localStorage.setItem(k, v); } catch (e) { memory[k] = v; }
    }
  };
})();

// Steps are hidden by CSS only once scripting is known to work, so a page
// whose script fails still shows everything.
document.body.classList.add("js");

// Theme toggle: light, dark, or follow the system. The choice is stored in
// localStorage and applied as early as possible by an inline script in the
// page head; this only wires the button.
(function () {
  "use strict";
  var key = "walkthrough.theme";
  var button = document.getElementById("theme");
  if (!button) {
    return;
  }
  function effective() {
    var t = document.documentElement.getAttribute("data-theme");
    if (t) {
      return t;
    }
    return window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
  }
  function label() {
    button.textContent = effective() === "dark" ? "☾" : "☀";
    button.title = "Switch to " + (effective() === "dark" ? "light" : "dark") + " theme";
  }
  button.addEventListener("click", function () {
    var next = effective() === "dark" ? "light" : "dark";
    document.documentElement.setAttribute("data-theme", next);
    store.set(key, next);
    label();
    window.dispatchEvent(new Event("resize"));
  });
  label();
})();

// Stepper for walkthrough pages: one step visible at a time, driven by the
// URL hash (#intro, #step-N), with keyboard navigation, a progress bar, an
// optional show-all mode, and best-effort mermaid rendering.
(function () {
  "use strict";
  var steps = Array.prototype.slice.call(document.querySelectorAll("section.step"));
  if (steps.length === 0) {
    return;
  }
  var links = Array.prototype.slice.call(document.querySelectorAll(".step-link"));
  var prev = document.getElementById("prev");
  var next = document.getElementById("next");
  var bar = document.getElementById("progress");
  var showAll = document.getElementById("show-all");
  var last = steps.length - 1;
  var current = 0;

  function indexFromHash() {
    var h = location.hash || "#intro";
    if (h === "#intro") {
      return 0;
    }
    var m = /^#step-(\d+)$/.exec(h);
    if (m) {
      return Math.min(last, Math.max(0, parseInt(m[1], 10)));
    }
    // A heading anchor inside a step: show the step that contains it.
    var el = document.getElementById(h.slice(1));
    while (el && el !== document.body) {
      if (el.classList && el.classList.contains("step")) {
        return parseInt(el.getAttribute("data-step"), 10) || 0;
      }
      el = el.parentNode;
    }
    return 0;
  }

  function hashFor(i) {
    return i === 0 ? "#intro" : "#step-" + i;
  }

  function show(i, scroll) {
    current = i;
    steps.forEach(function (s, j) {
      s.classList.toggle("active", j === i);
    });
    links.forEach(function (a, j) {
      a.classList.toggle("active", j === i);
      a.classList.toggle("done", j < i);
    });
    if (prev) {
      prev.href = hashFor(Math.max(0, i - 1));
      prev.classList.toggle("disabled", i === 0);
    }
    if (next) {
      next.href = hashFor(Math.min(last, i + 1));
      next.classList.toggle("disabled", i === last);
      next.textContent = i === last ? "Done" : "Next →";
    }
    if (bar) {
      bar.style.width = (last === 0 ? 100 : (i / last) * 100) + "%";
    }
    if (scroll) {
      window.scrollTo(0, 0);
    }
    document.title = document.title.replace(/^(\d+\. )?/, i === 0 ? "" : i + ". ");
  }

  function go(i) {
    i = Math.min(last, Math.max(0, i));
    if (hashFor(i) !== location.hash) {
      location.hash = hashFor(i);
    } else {
      show(i, true);
    }
  }

  window.addEventListener("hashchange", function () {
    var i = indexFromHash();
    show(i, !document.body.classList.contains("show-all"));
    var target = location.hash && document.getElementById(location.hash.slice(1));
    if (target && !/^#(intro|step-\d+)$/.test(location.hash)) {
      target.scrollIntoView();
    }
  });

  document.addEventListener("keydown", function (e) {
    if (e.target && /^(INPUT|TEXTAREA|SELECT)$/.test(e.target.tagName)) {
      return;
    }
    if (e.key === "ArrowRight" || e.key === "j") {
      go(current + 1);
    } else if (e.key === "ArrowLeft" || e.key === "k") {
      go(current - 1);
    }
  });

  if (showAll) {
    var key = "walkthrough.showAll";
    showAll.checked = store.get(key) === "1";
    document.body.classList.toggle("show-all", showAll.checked);
    showAll.addEventListener("change", function () {
      store.set(key, showAll.checked ? "1" : "0");
      document.body.classList.toggle("show-all", showAll.checked);
      show(current, false);
    });
  }

  // Browsers jump to the hash target on load; the stepper controls the
  // scroll position instead.
  if ("scrollRestoration" in history) {
    history.scrollRestoration = "manual";
  }
  show(indexFromHash(), /^#(step-\d+)$/.test(location.hash));

  // Mermaid blocks are optional: when the CDN is reachable they render as
  // diagrams, otherwise the source stays visible as text.
  if (document.querySelector("pre.mermaid")) {
    var s = document.createElement("script");
    s.src = "https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.min.js";
    s.onload = function () {
      var theme = document.documentElement.getAttribute("data-theme");
      var dark = theme ? theme === "dark" : window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches;
      window.mermaid.initialize({ startOnLoad: false, theme: dark ? "dark" : "default" });
      window.mermaid.run({ querySelector: "pre.mermaid" });
    };
    document.head.appendChild(s);
  }
})();

// Interactive package map (the `imports` generated block): click a package
// to highlight and wire what it imports and what imports it. The data is
// embedded in the page as JSON at build time.
(function () {
  "use strict";
  Array.prototype.slice.call(document.querySelectorAll(".pkgmap")).forEach(function (map) {
    var data = JSON.parse(map.querySelector(".pkgmap-data").textContent);
    var body = map.querySelector(".pkgmap-body");
    var layers = map.querySelector(".pkgmap-layers");
    var panel = map.querySelector(".pkgmap-panel");
    var svg = map.querySelector(".pkgmap-wires");
    var byId = {};
    var dependents = {};
    data.layers.forEach(function (l) {
      l.packages.forEach(function (p) {
        byId[p.id] = p;
        p.imports = p.imports || [];
        p.imports.forEach(function (d) {
          (dependents[d] = dependents[d] || []).push(p.id);
        });
      });
    });
    var selected = null;
    function node(id) {
      return layers.querySelector('.pkg[data-id="' + id.replace(/"/g, '\\"') + '"]');
    }
    function short(id) {
      return id.replace(/^internal\//, "");
    }
    function chips(ids) {
      if (!ids || ids.length === 0) {
        return '<span class="muted">none</span>';
      }
      return ids.map(function (id) {
        return '<button type="button" class="chip" data-go="' + id + '">' + short(id) + "</button>";
      }).join("");
    }
    function draw() {
      while (svg.firstChild) {
        svg.removeChild(svg.firstChild);
      }
      if (!selected || layers.offsetWidth === 0) {
        return;
      }
      var box = layers.getBoundingClientRect();
      svg.setAttribute("width", box.width);
      svg.setAttribute("height", box.height);
      var ns = "http://www.w3.org/2000/svg";
      var defs = document.createElementNS(ns, "defs");
      defs.innerHTML = '<marker id="pm-imp" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="6" markerHeight="6" orient="auto"><path d="M0,0 L10,5 L0,10 z" class="imp"/></marker>' +
        '<marker id="pm-dep" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="6" markerHeight="6" orient="auto"><path d="M0,0 L10,5 L0,10 z" class="dep"/></marker>';
      svg.appendChild(defs);
      function rect(el) {
        var r = el.getBoundingClientRect();
        return { l: r.left - box.left, r: r.right - box.left, t: r.top - box.top, b: r.bottom - box.top, x: (r.left + r.right) / 2 - box.left, y: (r.top + r.bottom) / 2 - box.top };
      }
      function wire(fromId, toId, cls) {
        var a = node(fromId), c = node(toId);
        if (!a || !c) {
          return;
        }
        var p = rect(a), q = rect(c);
        var d;
        if (Math.abs(p.y - q.y) < 4) {
          var y = p.y, x1 = p.x < q.x ? p.r : p.l, x2 = p.x < q.x ? q.l : q.r;
          d = "M" + x1 + "," + y + " C" + (x1 + x2) / 2 + "," + (y - 24) + " " + (x1 + x2) / 2 + "," + (y - 24) + " " + x2 + "," + y;
        } else {
          var y1 = p.y < q.y ? p.b : p.t, y2 = p.y < q.y ? q.t : q.b;
          d = "M" + p.x + "," + y1 + " C" + p.x + "," + (y1 + y2) / 2 + " " + q.x + "," + (y1 + y2) / 2 + " " + q.x + "," + y2;
        }
        var path = document.createElementNS(ns, "path");
        path.setAttribute("d", d);
        path.setAttribute("class", cls);
        path.setAttribute("marker-end", "url(#pm-" + cls + ")");
        svg.appendChild(path);
      }
      byId[selected].imports.forEach(function (id) { wire(selected, id, "imp"); });
      (dependents[selected] || []).forEach(function (id) { wire(id, selected, "dep"); });
    }
    function select(id) {
      selected = id;
      var p = byId[id];
      var imps = p.imports, deps = dependents[id] || [];
      Array.prototype.slice.call(layers.querySelectorAll(".pkg")).forEach(function (el) {
        var pid = el.getAttribute("data-id");
        el.classList.toggle("sel", pid === id);
        el.classList.toggle("imp", imps.indexOf(pid) >= 0);
        el.classList.toggle("dep", deps.indexOf(pid) >= 0);
      });
      panel.innerHTML = '<h4><code>' + id + "</code></h4>" +
        '<p class="muted">' + (p.desc || "No package comment.") + "</p>" +
        '<p class="pkgmap-sub">Imports <span>' + imps.length + "</span></p>" + chips(imps) +
        '<p class="pkgmap-sub">Imported by <span>' + deps.length + "</span></p>" + chips(deps);
      draw();
    }
    layers.addEventListener("click", function (e) {
      var el = e.target.closest(".pkg");
      if (el) {
        select(el.getAttribute("data-id"));
      }
    });
    panel.addEventListener("click", function (e) {
      var el = e.target.closest(".chip");
      if (el) {
        select(el.getAttribute("data-go"));
        node(el.getAttribute("data-go")).scrollIntoView({ block: "nearest" });
      }
    });
    window.addEventListener("resize", draw);
    window.addEventListener("hashchange", function () { setTimeout(draw, 0); });
    body.classList.add("ready");
  });
})();

// Ask panel under each step. The prompt bundles the page, the step and the
// files it excerpts. With the local server started with -ask, the question
// is POSTed to /ask and the answer shown inline; otherwise the prompt is
// copied to the clipboard for any agent.
(function () {
  "use strict";
  var meta = function (name) {
    var m = document.querySelector('meta[name="' + name + '"]');
    return m ? m.getAttribute("content") : "";
  };
  var token = meta("walkthrough-token");
  var live = meta("walkthrough-ask") === "1" && token;
  var sha = (document.querySelector(".sha a") || {}).textContent || "";
  Array.prototype.slice.call(document.querySelectorAll("details.ask")).forEach(function (box) {
    var step = box.parentNode;
    var input = box.querySelector("textarea");
    var send = box.querySelector(".ask-send");
    var status = box.querySelector(".ask-status");
    var answer = box.querySelector(".ask-answer");
    function prompt() {
      var files = Array.prototype.slice.call(step.querySelectorAll("figure.excerpt .path")).map(function (el) {
        return el.textContent;
      });
      var lines = [
        "Context: walkthrough page \"" + box.getAttribute("data-page") + "\", step \"" + box.getAttribute("data-step") + "\"" + (sha ? " (commit " + sha.trim() + ")" : "") + ".",
        files.length ? "Files excerpted on this step: " + files.join(", ") + "." : "",
        "Step text:",
        step.querySelector(".body").innerText.trim().slice(0, 4000),
        "",
        "Question: " + input.value.trim()
      ];
      return lines.filter(function (l) { return l !== null; }).join("\n");
    }
    send.addEventListener("click", function () {
      if (!input.value.trim()) {
        input.focus();
        return;
      }
      var p = prompt();
      if (!live) {
        var done = function () { status.textContent = "Copied. Paste it into your agent."; };
        if (navigator.clipboard && navigator.clipboard.writeText) {
          navigator.clipboard.writeText(p).then(done, function () { answer.textContent = p; answer.hidden = false; });
        } else {
          answer.textContent = p;
          answer.hidden = false;
        }
        return;
      }
      send.disabled = true;
      status.textContent = "Asking…";
      answer.hidden = true;
      fetch("ask", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ token: token, prompt: p }) })
        .then(function (r) { return r.text().then(function (t) { return { ok: r.ok, text: t }; }); })
        .then(function (r) {
          status.textContent = r.ok ? "" : "The agent command failed.";
          answer.textContent = r.text;
          answer.hidden = false;
        }, function (e) {
          status.textContent = "Request failed: " + e;
        })
        .then(function () { send.disabled = false; });
    });
  });
})();
