/* ─── Landing micro-interactions ───
   KPI count-up, quickstart typing loop, nav shrink on scroll.
   No scroll-triggered reveal: sections render statically (anti-slop
   rule — the page settles after the initial hero entrance, which is
   a pure CSS animation in style.css). */

(function () {
  "use strict";
  if (window.__landingInit) return;
  window.__landingInit = true;

  const reduced = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  const opts = { threshold: 0.4 };

  // ── nav shrink on scroll ──
  const nav = document.getElementById("nav");
  if (nav) {
    const onScroll = () => nav.classList.toggle("is-scrolled", window.scrollY > 24);
    onScroll();
    window.addEventListener("scroll", onScroll, { passive: true });
  }

  // ── mobile nav drawer (hamburger → slide-in) ──
  const burger = document.getElementById("nav-burger");
  const drawer = document.getElementById("nav-drawer");
  if (burger && drawer) {
    const panel = drawer.querySelector(".nav-drawer-panel");
    const focusables = () => [...panel.querySelectorAll("a, button")].filter((el) => el.offsetParent !== null);
    const setOpen = (open) => {
      drawer.classList.toggle("is-open", open);
      drawer.setAttribute("aria-hidden", String(!open));
      burger.setAttribute("aria-expanded", String(open));
      burger.setAttribute("aria-label", open ? "Close menu" : "Open menu");
      if (open) {
        const first = focusables()[0];
        if (first) first.focus();
      } else {
        burger.focus();
      }
    };
    burger.addEventListener("click", () => setOpen(!drawer.classList.contains("is-open")));
    drawer.addEventListener("click", (e) => {
      if (e.target.closest("[data-drawer-close]")) setOpen(false);
    });
    drawer.querySelectorAll("a").forEach((a) => a.addEventListener("click", () => setOpen(false)));
    document.addEventListener("keydown", (e) => {
      if (e.key === "Escape" && drawer.classList.contains("is-open")) setOpen(false);
      if (e.key === "Tab" && drawer.classList.contains("is-open")) {
        const f = focusables();
        if (!f.length) return;
        const first = f[0], last = f[f.length - 1];
        if (e.shiftKey && document.activeElement === first) { e.preventDefault(); last.focus(); }
        else if (!e.shiftKey && document.activeElement === last) { e.preventDefault(); first.focus(); }
      }
    });
  }

  // ── KPI count-up (one-shot when visible) ──
  const kpiCells = document.querySelectorAll(".kpi-num[data-count]");
  if (kpiCells.length) {
    const io = new IntersectionObserver((entries) => {
      for (const e of entries) {
        if (!e.isIntersecting) continue;
        const el = e.target;
        io.unobserve(el);
        const target = parseInt(el.dataset.count, 10);
        const suffix = el.dataset.suffix || "";
        if (reduced || target === 0) {
          el.textContent = target + suffix;
          continue;
        }
        const t0 = performance.now();
        const dur = 900;
        const step = (now) => {
          const p = Math.min(1, (now - t0) / dur);
          const eased = 1 - Math.pow(1 - p, 3);
          el.textContent = Math.round(target * eased) + suffix;
          if (p < 1) requestAnimationFrame(step);
        };
        requestAnimationFrame(step);
      }
    }, opts);
    kpiCells.forEach((c) => io.observe(c));
  }

  // ── quickstart typing loop ──
  const typer = document.getElementById("typer");
  if (typer) {
    const lines = [
      { p: "$", t: "curl -fsSL https://github.com/bnhminh1010/homelab-dashboard/releases/latest/download/install.sh | bash" },
      { p: "$", t: "podman compose up -d" },
      { p: "→", t: "https://homelab-dashboard.tailnet.ts.net" },
    ];
    // Reduced motion: render the three commands statically, no typing.
    if (reduced) {
      typer.innerHTML = lines
        .map((l) => '<span class="c-prompt">' + l.p + "</span> <span class=\"c-cmd\">" + esc(l.t) + "</span>")
        .join("<br>");
    } else {
      let li = 0, ci = 0, phase = "type";
      const typeIo = new IntersectionObserver((entries) => {
        for (const e of entries) {
          if (e.isIntersecting) {
            typeIo.disconnect();
            tick();
          }
        }
      }, { threshold: 0.3 });
      typeIo.observe(typer);

      function tick() {
        if (phase === "type") {
          const line = lines[li];
          const cur = line.t.slice(0, ++ci);
          typer.innerHTML =
            '<span class="c-prompt">' + line.p + "</span> " +
            '<span class="c-cmd">' + esc(cur) + "</span><span class=\"c-cursor\"></span>";
          if (ci >= line.t.length) { phase = "pause"; setTimeout(tick, 650); }
          else setTimeout(tick, 18 + Math.random() * 30);
        } else if (phase === "pause") {
          phase = "erase";
          setTimeout(tick, 60);
        } else if (phase === "erase") {
          const line = lines[li];
          ci = Math.max(0, ci - 3);
          const cur = line.t.slice(0, ci);
          typer.innerHTML =
            '<span class="c-prompt">' + line.p + "</span> " +
            '<span class="c-cmd">' + esc(cur) + "</span><span class=\"c-cursor\"></span>";
          if (ci <= 0) {
            li = (li + 1) % lines.length;
            phase = "type";
          }
          setTimeout(tick, 16);
        }
      }
    }

    function esc(s) {
      return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
    }
  }

  // ── quickstart copy button (copies the real installer command) ──
  const copyBtn = document.getElementById("qs-copy");
  if (copyBtn) {
    const INSTALL_CMD =
      "curl -fsSL https://github.com/bnhminh1010/homelab-dashboard/releases/latest/download/install.sh | bash";
    const copy = async () => {
      try {
        await navigator.clipboard.writeText(INSTALL_CMD);
      } catch (e) {
        const ta = document.createElement("textarea");
        ta.value = INSTALL_CMD;
        ta.style.position = "fixed";
        ta.style.opacity = "0";
        document.body.appendChild(ta);
        ta.select();
        document.execCommand("copy");
        ta.remove();
      }
      copyBtn.classList.add("is-copied");
      copyBtn.textContent = "Copied ✓";
      setTimeout(() => {
        copyBtn.classList.remove("is-copied");
        copyBtn.textContent = "Copy";
      }, 1800);
    };
    copyBtn.addEventListener("click", copy);
  }

  // ── screenshot filmstrip: auto-advance loop, pause on hover/focus ──
  // Scrolls left→right continuously. The loop pauses while the pointer or
  // keyboard focus is on the track, after a button click, and whenever the
  // tab is hidden. Respects prefers-reduced-motion (static, no auto-scroll).
  const track = document.getElementById("shots-track");
  const prevBtn = document.getElementById("shots-prev");
  const nextBtn = document.getElementById("shots-next");
  if (track && prevBtn && nextBtn) {
    // Seamless loop: mirror the 8 cards on both sides of the track and
    // start at the original set. The loop wraps by subtracting one set
    // width whenever scrollLeft crosses it — visually identical because
    // the mirrored cards are exact clones, so there is no jump.
    const cards = [...track.children];
    const gap = 22;
    const step = () => {
      const card = track.querySelector(".shot-card");
      return card ? card.offsetWidth + gap : track.clientWidth * 0.8;
    };
    const LOOP = step() * 8; // width of one 8-card set
    const clonesA = cards.map((c) => c.cloneNode(true));
    const clonesB = cards.map((c) => c.cloneNode(true));
    clonesA.forEach((c) => track.prepend(c));
    clonesB.forEach((c) => track.appendChild(c));
    track.scrollLeft = LOOP; // start at the original set
    const norm = () => {
      // Keep scrollLeft inside [0, 2*LOOP) so there is always a full
      // mirrored set on either side. Wrapping subtracts one set width;
      // the mirrored cards are pixel-identical clones so the swap is
      // invisible. IMPORTANT: do NOT push scrollLeft back up to LOOP —
      // centering the FIRST card of the original set legitimately lands
      // below LOOP, and forcing it up would make the auto-loop visibly
      // jump a full set on resume (the flicker we hunted for v3).
      let s = track.scrollLeft;
      while (s >= LOOP * 2) s -= LOOP;
      track.scrollLeft = Math.max(0, s);
    };
    const clickStep = (dir) => {
      track.scrollBy({ left: dir * step(), behavior: reduced ? "auto" : "smooth" });
      hold();
      window.setTimeout(norm, 700); // re-normalize once smooth scroll settles
    };
    prevBtn.addEventListener("click", () => clickStep(-1));
    nextBtn.addEventListener("click", () => clickStep(1));

    const SPEED = 70; // px per second
    const HOLD_MS = 3500; // pause after a manual button click
    let raf = null, last = 0, paused = false, holdUntil = 0;

    // Coverflow depth: the centered card scales to 1.0, stays fully
    // opaque and casts the strongest shadow; cards further out shrink,
    // dim and tilt slightly toward the viewer (rotateY), reading as a
    // carousel seen from straight ahead. Under reduced motion the tilt
    // is dropped; only the scroll-linked scale/fade remains.
    const applyDepth = () => {
      const tr = track.getBoundingClientRect();
      const center = tr.left + tr.width / 2;
      // Only style cards near the viewport (plus one on each side); the
      // rest are invisible and updating them every frame is wasted work.
      const lo = tr.left - 700, hi = tr.right + 700;
      for (const card of track.children) {
        const r = card.getBoundingClientRect();
        if (r.right < lo || r.left > hi) continue;
        // Use layout width (offsetWidth) as the distance denominator;
        // getBoundingClientRect().width is already scaled by the
        // coverflow transform and would skew the ratio.
        const half = card.offsetWidth / 2;
        const dist = (r.left + half - center) / half;
        const abs = Math.min(Math.abs(dist), 3);
        const scale = 1 - abs * 0.09;
        const opacity = Math.max(0.4, 1 - abs * 0.18);
        const rot = reduced ? 0 : Math.max(-10, Math.min(10, -dist * 3.5));
        card.style.transform = "scale(" + scale.toFixed(3) + ") rotateY(" + rot.toFixed(1) + "deg)";
        card.style.opacity = opacity.toFixed(3);
        // Quantize z-index so it only changes when a card crosses a
        // depth threshold — per-frame paint changes here cause visible
        // repaint flicker while the loop scrolls. The depth glow is a
        // pseudo-element whose opacity we drive via a CSS custom property
        // (compositor-only, no repaint).
        card.style.zIndex = String(Math.round(10 - Math.floor(abs * 2) * 2));
        card.style.setProperty("--glow", abs < 0.55 ? "1" : "0");
      }
    };

    function tick(ts) {
      if (!reduced && !paused && ts >= holdUntil && document.visibilityState === "visible") {
        const dt = last ? (ts - last) / 1000 : 0;
        track.scrollLeft = Math.min(track.scrollLeft + SPEED * dt, LOOP * 2);
        norm(); // seamless wrap at the set boundary — no visual jump
      }
      last = ts;
      if (!reduced) applyDepth();
      raf = requestAnimationFrame(tick);
    }

    // On pause (hover/focus/touch), smoothly scroll the nearest card to
    // center instead of letting scroll-snap jump instantly. While
    // auto-scrolling, snap stays off so the motion is smooth.
    const alignNearest = () => {
      const tr = track.getBoundingClientRect();
      const center = tr.left + tr.width / 2;
      let best = null, bestDist = Infinity;
      for (const card of track.children) {
        const r = card.getBoundingClientRect();
        const dist = Math.abs(r.left + r.width / 2 - center);
        if (dist < bestDist) { bestDist = dist; best = card; }
      }
      if (best) {
        const r = best.getBoundingClientRect();
        const target = track.scrollLeft + (r.left + r.width / 2 - center);
        track.scrollTo({ left: target, behavior: reduced ? "auto" : "smooth" });
      }
    };
    // Per-card hover: the card under the pointer eases to center. Because
    // the track is mirrored, the hovered card exists 3×; pick the mirror
    // whose centering scroll is closest to the current position so the
    // motion is always the shortest step (never a long rewind).
    const centerCard = (card) => {
      const tr = track.getBoundingClientRect();
      const r = card.getBoundingClientRect();
      const center = tr.left + tr.width / 2;
      const raw = track.scrollLeft + (r.left + r.width / 2 - center);
      const max = track.scrollWidth - track.clientWidth;
      let best = raw, bestDist = Math.abs(raw - track.scrollLeft);
      for (const cand of [raw - LOOP, raw + LOOP]) {
        if (cand >= 0 && cand <= max) {
          const d = Math.abs(cand - track.scrollLeft);
          if (d < bestDist) { bestDist = d; best = cand; }
        }
      }
      track.scrollTo({ left: best, behavior: reduced ? "auto" : "smooth" });
    };
    const pause = () => {
      paused = true;
      track.style.scrollSnapType = "none"; // prevent instant snap jump
    };
    const resume = () => {
      paused = false;
      track.style.scrollSnapType = "none";
    };
    const hold = () => { holdUntil = performance.now() + HOLD_MS; };

    let hoverCard = null;
    track.addEventListener("mouseenter", pause);
    track.addEventListener("mouseleave", () => { hoverCard = null; resume(); });
    // Per-card hover via mousemove, NOT mouseover: mouseover fires when the
    // scroll animation drags a different card under a stationary pointer,
    // which would cascade the centering endlessly. mousemove only fires on
    // real pointer travel, so the hovered card is always user-chosen. The
    // card !== hoverCard check already limits work to actual card changes,
    // so no extra throttle (it would drop fast pointer travel).
    track.addEventListener("mousemove", (e) => {
      if (!paused) return;
      const card = e.target.closest ? e.target.closest(".shot-card") : null;
      if (card && card !== hoverCard) {
        hoverCard = card;
        centerCard(card);
      }
    });
    track.addEventListener("focusin", () => { pause(); alignNearest(); });
    track.addEventListener("focusout", resume);
    track.addEventListener("touchstart", () => { pause(); alignNearest(); }, { passive: true });
    track.addEventListener("touchend", resume, { passive: true });
    // Under reduced motion there is no rAF-driven loop, so manual
    // scrolling still needs the depth pass via the scroll event.
    track.addEventListener("scroll", () => { if (reduced) applyDepth(); }, { passive: true });
    document.addEventListener("visibilitychange", () => {
      if (document.hidden) pause();
      else resume();
    });

    if (!reduced) {
      track.style.scrollSnapType = "none"; // auto-loop runs smooth from the start
      raf = requestAnimationFrame(tick);
    }
  }

  // ── demo iframe: hide the inner scrollbar (wheel scrolling still works) ──
  // The dashboard grid is a scroll container by design; inside the embedded
  // demo frame that shows a visible scrollbar which reads as sloppy. We
  // inject a small style into the same-origin frame document instead of
  // touching the app's own CSS.
  const demoFrame = document.querySelector('.demo-frame iframe');
  if (demoFrame) {
    const hideScroller = () => {
      try {
        const doc = demoFrame.contentDocument;
        // Lazy-loaded frames start as about:blank — only touch the real document.
        if (!doc || doc.URL === "about:blank" || !doc.head) return;
        const st = doc.createElement("style");
        st.textContent =
          ".dashboard-grid { scrollbar-width: none; -ms-overflow-style: none; }" +
          ".dashboard-grid::-webkit-scrollbar { display: none; }";
        doc.head.appendChild(st);
      } catch (e) { /* cross-origin or not ready — harmless */ }
    };
    // Always hook the load event: with loading="lazy" the contentDocument is
    // about:blank at init time and only becomes the real document on load.
    demoFrame.addEventListener("load", hideScroller);
    if (demoFrame.contentDocument && demoFrame.contentDocument.URL !== "about:blank") {
      hideScroller();
    }
  }
})();
