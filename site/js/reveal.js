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
      let typerVisible = false;
      const typeIo = new IntersectionObserver((entries) => {
        for (const e of entries) typerVisible = e.isIntersecting;
      }, { threshold: 0.1 });
      typeIo.observe(typer);
      tick(); // loop self-schedules; the gate below holds it until visible

      function tick() {
        // Off-screen the loop is pure waste: every keystroke rewrites
        // innerHTML = page-wide style recalc even though nothing is
        // visible. Gate it on the IntersectionObserver flag instead of
        // disconnecting — so it resumes when the user scrolls back.
        if (!typerVisible) { setTimeout(tick, 250); return; }
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

  // ── screenshot filmstrip: transform-driven coverflow carousel ──
  // No scroll container: the JS auto-loop drives translateX on the track,
  // which is compositor-only (scrollLeft on a big track was the iGPU jank
  // source). The loop pauses while the pointer or keyboard focus is on the
  // track, after a button click, and whenever the tab is hidden. All
  // centering goes through a self-written rAF tween — the browser's
  // smooth scrollTo drops frames on integrated GPUs, which made the
  // prev/next buttons land off-center on real hardware.
  const track = document.getElementById("shots-track");
  const prevBtn = document.getElementById("shots-prev");
  const nextBtn = document.getElementById("shots-next");
  const carousel = track ? track.parentElement : null;
  if (track && prevBtn && nextBtn && carousel) {
    const gap = 22;
    const SPEED = 90; // px per second (70 felt too slow to idle on a card)
    const HOLD_MS = 3500; // pause after a manual button click
    const TWEEN_MS = 250; // centering tween duration
    const SET_N = 8; // original set size

    // Seamless loop: mirror the 8 cards on both sides (clone A + originals
    // + clone B = 24 cards). Wrap subtracts one set width; the swap is
    // invisible because the clones are pixel-identical, and both tween
    // directions always find a real card.
    const cards = [...track.children];
    const clonesA = cards.map((c) => c.cloneNode(true));
    const clonesB = cards.map((c) => c.cloneNode(true));
    // prepend() puts each node FIRST, so iterating forward would REVERSE
    // the clone-A block (Terminal…Services) and break the index%SET_N
    // mapping used by the counter/announcer. Prepend in reverse order so
    // the mirror reads Services…Terminal like the originals.
    [...clonesA].reverse().forEach((c) => track.prepend(c));
    clonesB.forEach((c) => track.appendChild(c));
    const cardsAll = [...track.children]; // 24

    let cardW = 620, trackLeft = 4, step = 642, SET = 5136, maxOffset = 0;
    const measure = () => {
      const card = track.querySelector(".shot-card");
      cardW = card ? card.offsetWidth : 620; // layout width — NOT the rect width (already scaled by coverflow)
      step = cardW + gap;
      SET = step * SET_N;
      trackLeft = track.offsetLeft + 4; // 4px track padding
      maxOffset = Math.max(0, track.scrollWidth - carousel.clientWidth);
    };
    // (i+0.5)*step counts half the gap twice (11px systematic error) —
    // this form is exact. NOTE: positions are relative to the carousel
    // (trackLeft is track.offsetLeft-based), so the target is
    // clientWidth/2 — NOT the absolute viewCenter(), which would add the
    // carousel's own offset (margin:auto) to every measurement.
    const centerOffset = (i) => trackLeft + i * step + cardW / 2 - carousel.clientWidth / 2;

    let offset = 0, raf = null, last = 0, lastOffset = NaN;
    let paused = false, holdUntil = 0, tween = null, drag = null;
    let visible = false; // carousel inside the viewport (IntersectionObserver)

    const render = () => {
      track.style.transform = "translate3d(" + (-offset).toFixed(2) + "px,0,0)";
    };

    const wrapOffset = () => {
      while (offset >= SET * 2) offset -= SET;
      while (offset < 0) offset += SET;
      offset = Math.max(0, Math.min(offset, maxOffset));
    };

    // Coverflow depth: the centered card scales to 1.0, stays fully
    // opaque and casts the strongest shadow; cards further out shrink,
    // dim and tilt slightly toward the viewer (rotateY), reading as a
    // carousel seen from straight ahead. Distances come from the offset
    // math (no getBoundingClientRect reads per card).
    // DEPTH IS SMOOTH, not quantized: step-quantized values (scale steps
    // of 0.01 ≈ 6.2px on a 620px card, rotateY steps of 0.5°) make the
    // coverflow ratchet past discrete steps while the track itself moves
    // smoothly — the visible "jitter" on real hardware. Transform and
    // opacity are compositor-only, so per-frame writes cost a cheap style
    // recalc, never a paint; the writes are still cache-compared so
    // unchanged cards (paused, off-screen, tween finished) write nothing.
    const applyDepth = () => {
      const vc = carousel.clientWidth / 2;
      const half = cardW / 2;
      for (let i = 0; i < cardsAll.length; i++) {
        const pos = trackLeft + i * step + half - offset;
        if (Math.abs(pos - vc) > 700) continue; // only cards near the viewport
        const dist = (pos - vc) / half;
        const abs = Math.min(Math.abs(dist), 3);
        // Real 3D depth via translateZ (the track has preserve-3d, so the
        // carousel's perspective reaches the cards): the browser orders
        // the cards by their Z position, smoothly, every frame — no
        // z-index stepping, no stacking-context pop. With perspective
        // 1400px, z=-150 renders ≈ scale(0.90): same feel as the old
        // scale, but the side card can never jump "onto" the focused one.
        const z = -abs * 150;
        const opacity = Math.max(0.4, 1 - abs * 0.18);
        // rotateY bends the coverflow into a cylinder (7°/unit, clamp
        // ±20°): the side cards visibly turn away, the centered card pops.
        const rot = reduced ? 0 : Math.max(-20, Math.min(20, -dist * 7));
        const card = cardsAll[i];
        const t = "translateZ(" + z.toFixed(1) + "px) rotateY(" + rot.toFixed(2) + "deg)";
        if (card.style.transform !== t) card.style.transform = t;
        const o = opacity.toFixed(3);
        if (card.style.opacity !== o) card.style.opacity = o;
        const glow = abs < 0.55 ? "1" : "0";
        if (card.style.getPropertyValue("--glow") !== glow) card.style.setProperty("--glow", glow);
      }
      paintCounter();
    };

    // rAF tween with mirror-aware target picking: the tween always goes to
    // the target's clone nearest the current offset, so stepping past the
    // end of the track (card 23 → 0) looks like a normal single step.
    const startTween = (target, ms) => {
      let best = target, bd = Infinity;
      for (let k = -1; k <= 1; k++) {
        const c = target + k * SET;
        const d = Math.abs(c - offset);
        if (d < bd) { bd = d; best = c; }
      }
      const from = offset;
      const dur = reduced ? 0 : (ms || TWEEN_MS);
      paused = true;
      holdUntil = performance.now() + HOLD_MS;
      if (dur === 0) { offset = best; render(); applyDepth(); return; }
      // A tween requested while the tick is parked (carousel scrolled out,
      // IO not re-fired yet) must still run: restart the loop here instead
      // of waiting for the observer.
      if (!raf) raf = requestAnimationFrame(tick);
      tween = { from, to: best, t0: performance.now(), dur };
    };

    function tick(ts) {
      // Carousel off-screen: stop scheduling entirely. (The paused flag is
      // for USER interaction — hover/focus/touch — and must not be touched
      // here, or a scroll-into-view would override an active hover pause.)
      if (!visible) { raf = null; return; }
      const dt = last ? Math.min((ts - last) / 1000, 0.05) : 0;
      last = ts;
      if (tween) {
        const p = Math.min(1, (ts - tween.t0) / tween.dur);
        const e = 1 - Math.pow(1 - p, 3); // easeOutCubic
        offset = tween.from + (tween.to - tween.from) * e;
        if (p >= 1) { offset = tween.to; tween = null; }
      } else if (!reduced && !paused && ts >= holdUntil && document.visibilityState === "visible") {
        offset = Math.min(offset + SPEED * dt, maxOffset);
        if (offset >= SET * 1.5) offset -= SET; // invisible mirror swap
      }
      render();
      // Depth is a function of the offset: skip the 24-card pass entirely
      // when nothing moved (paused / tween finished).
      if (offset !== lastOffset) { lastOffset = offset; applyDepth(); }
      raf = requestAnimationFrame(tick);
    }

    const nearestCardIndex = () => {
      const vc = carousel.clientWidth / 2;
      let best = 0, bd = Infinity;
      for (let i = 0; i < cardsAll.length; i++) {
        const pos = trackLeft + i * step + cardW / 2 - offset;
        const d = Math.abs(pos - vc);
        if (d < bd) { bd = d; best = i; }
      }
      return best;
    };

    // Prev/next: snap to the card nearest center FIRST, then step one
    // card. Stepping from the raw current position drifts when the loop
    // is mid-card; snap-first always lands exactly on a card.
    const clickStep = (dir) => {
      const next = (nearestCardIndex() + dir + cardsAll.length) % cardsAll.length;
      startTween(centerOffset(next));
      return next;
    };
    prevBtn.addEventListener("click", () => clickStep(-1));
    nextBtn.addEventListener("click", () => clickStep(1));

    const pause = () => { paused = true; };
    const resume = () => { paused = false; };

    let hoverCard = null, hx = 0, hy = 0;
    carousel.addEventListener("mousemove", (e) => {
      if (!paused) return;
      // Jitter guard: a stationary pointer (or sub-pixel sensor noise)
      // must never re-trigger. Without it, the track moving during a
      // tween slides a DIFFERENT card under the resting pointer and the
      // next jitter re-centers it — the endless focus flip at the edge.
      if (Math.abs(e.clientX - hx) + Math.abs(e.clientY - hy) < 8) return;
      hx = e.clientX; hy = e.clientY;
      const card = e.target.closest ? e.target.closest(".shot-card") : null;
      if (card && card !== hoverCard) {
        hoverCard = card;
        startTween(centerOffset(cardsAll.indexOf(card)));
      }
    });
    track.addEventListener("mouseenter", pause);
    track.addEventListener("mouseleave", () => { hoverCard = null; resume(); });
    track.addEventListener("focusin", () => { pause(); startTween(centerOffset(nearestCardIndex())); });
    track.addEventListener("focusout", resume);
    track.addEventListener("touchstart", () => { pause(); startTween(centerOffset(nearestCardIndex())); }, { passive: true });
    track.addEventListener("touchend", resume, { passive: true });
    document.addEventListener("visibilitychange", () => {
      if (document.hidden) pause();
      else resume();
    });

    // ── keyboard: ←/→ step · Space pause/resume · Home/End jump ──
    // Keyboard-first (hallmark): every pointer affordance needs a keyboard
    // equivalent. Handled ONLY while focus lives inside the carousel (the
    // user Tabbed there) — Space must not steal page scroll, and the
    // arrows must not fight inputs elsewhere on the page. Hold-to-repeat
    // is throttled to the 80–120ms instant-feedback window.
    const live = document.getElementById("shots-live");
    const counter = document.getElementById("shots-count");
    const cardTitle = (i) => {
      const fig = cardsAll[i] && cardsAll[i].querySelector("figcaption b");
      return fig ? fig.textContent : "";
    };
    const announceIndex = (i) => { if (live) live.textContent = "Showing " + cardTitle(i % SET_N); };
    let lastCount = -1;
    const paintCounter = () => {
      const n = (nearestCardIndex() % SET_N) + 1;
      if (counter && n !== lastCount) { lastCount = n; counter.textContent = n + " / " + SET_N; }
    };
    let lastKeyAt = 0;
    const KEY_REPEAT = 120;
    document.addEventListener("keydown", (e) => {
      if (!carousel.contains(document.activeElement)) return;
      if (e.key === " " && document.activeElement.tagName === "BUTTON") return; // let the focused nav button handle Space
      const now = performance.now();
      if (e.repeat && now - lastKeyAt < KEY_REPEAT) return;
      lastKeyAt = now;
      switch (e.key) {
        case "ArrowLeft": e.preventDefault(); announceIndex(clickStep(-1)); break;
        case "ArrowRight": e.preventDefault(); announceIndex(clickStep(1)); break;
        case "Home": e.preventDefault(); startTween(centerOffset(SET_N)); announceIndex(SET_N); break;
        case "End": e.preventDefault(); startTween(centerOffset(2 * SET_N - 1)); announceIndex(2 * SET_N - 1); break;
        case " ": e.preventDefault(); paused = !paused; if (!paused) holdUntil = 0; break; // resume must not wait out a stale holdUntil
      }
    });

    // Drag to scrub: pointerdown starts a drag (cancelling any in-flight
    // centering tween — otherwise tick's tween branch overwrites the
    // dragged offset every frame). Move/up listen on document because
    // pointer capture is unreliable; the drag gate replaces it.
    track.addEventListener("pointerdown", (e) => {
      pause();
      tween = null;
      drag = { x: e.clientX, from: offset };
      try { track.setPointerCapture(e.pointerId); } catch (err) { /* best-effort */ }
    }, { passive: true });
    document.addEventListener("pointermove", (e) => {
      if (!drag) return;
      offset = Math.max(0, Math.min(maxOffset, drag.from + (drag.x - e.clientX)));
      render();
      if (offset !== lastOffset) { lastOffset = offset; applyDepth(); }
    }, { passive: true });
    const endDrag = () => {
      if (!drag) return;
      drag = null;
      startTween(centerOffset(nearestCardIndex()));
    };
    document.addEventListener("pointerup", endDrag);
    document.addEventListener("pointercancel", endDrag);

    // Observe the CAROUSEL, never the track: the translated track sits
    // outside the viewport, so observing it reports "off-screen" and the
    // callback would cancel the tick — auto-loop and tweens die silently.
    const io = new IntersectionObserver((entries) => {
      for (const e of entries) {
        visible = e.isIntersecting;
        if (visible && !reduced && !raf) raf = requestAnimationFrame(tick);
      }
    }, { threshold: 0.1 });
    io.observe(carousel);

    // Init: measure after layout settles (double rAF), then again once
    // images finish so cardW is authoritative. Start with card 8 — the
    // first card of the original set — centered.
    const boot = () => {
      measure();
      offset = centerOffset(8);
      render();
      applyDepth();
    };
    requestAnimationFrame(() => requestAnimationFrame(boot));
    window.addEventListener("load", boot);
    window.addEventListener("resize", () => { measure(); wrapOffset(); render(); applyDepth(); });

    if (!reduced) raf = requestAnimationFrame(tick);
    else applyDepth();
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
