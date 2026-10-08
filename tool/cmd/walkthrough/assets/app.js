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
    showAll.checked = localStorage.getItem(key) === "1";
    document.body.classList.toggle("show-all", showAll.checked);
    showAll.addEventListener("change", function () {
      localStorage.setItem(key, showAll.checked ? "1" : "0");
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
      var dark = window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches;
      window.mermaid.initialize({ startOnLoad: false, theme: dark ? "dark" : "default" });
      window.mermaid.run({ querySelector: "pre.mermaid" });
    };
    document.head.appendChild(s);
  }
})();
