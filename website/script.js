(() => {
  document.body.classList.add("js-enabled");

  const tabs = Array.from(document.querySelectorAll('[role="tab"]'));
  const panels = new Map(
    tabs.map((tab) => [tab, document.getElementById(tab.getAttribute("aria-controls"))])
  );

  function selectTab(nextTab, moveFocus) {
    for (const tab of tabs) {
      const selected = tab === nextTab;
      tab.setAttribute("aria-selected", selected ? "true" : "false");
      tab.tabIndex = selected ? 0 : -1;
      const panel = panels.get(tab);
      if (panel) {
        panel.hidden = !selected;
      }
    }
    if (moveFocus) {
      nextTab.focus();
    }
  }

  if (tabs.length > 0) {
    selectTab(tabs[0], false);
  }

  tabs.forEach((tab, index) => {
    tab.addEventListener("click", () => selectTab(tab, false));

    tab.addEventListener("keydown", (event) => {
      let targetIndex = null;
      if (event.key === "ArrowRight") {
        targetIndex = (index + 1) % tabs.length;
      } else if (event.key === "ArrowLeft") {
        targetIndex = (index - 1 + tabs.length) % tabs.length;
      } else if (event.key === "Home") {
        targetIndex = 0;
      } else if (event.key === "End") {
        targetIndex = tabs.length - 1;
      }

      if (targetIndex !== null) {
        event.preventDefault();
        selectTab(tabs[targetIndex], true);
      }
    });
  });

  const statusEl = document.getElementById("copy-status");
  const resetTimers = new WeakMap();

  function selectElementText(el) {
    const selection = window.getSelection();
    if (!selection) {
      return;
    }
    const range = document.createRange();
    range.selectNodeContents(el);
    selection.removeAllRanges();
    selection.addRange(range);
  }

  function announce(message) {
    if (!statusEl) {
      return;
    }
    statusEl.textContent = "";
    window.setTimeout(() => {
      statusEl.textContent = message;
    }, 10);
  }

  function resetButtonLater(btn) {
    const existing = resetTimers.get(btn);
    if (existing) {
      window.clearTimeout(existing);
    }
    const timer = window.setTimeout(() => {
      btn.textContent = "Copy";
      btn.removeAttribute("data-copied");
    }, 2000);
    resetTimers.set(btn, timer);
  }

  document.querySelectorAll("[data-copy-target]").forEach((btn) => {
    btn.addEventListener("click", async () => {
      const targetId = btn.getAttribute("data-copy-target");
      const codeEl = targetId ? document.getElementById(targetId) : null;
      if (!codeEl) {
        return;
      }

      const text = codeEl.textContent.replace(/\n$/, "");

      try {
        if (!navigator.clipboard || typeof navigator.clipboard.writeText !== "function") {
          throw new Error("Clipboard API unavailable");
        }
        await navigator.clipboard.writeText(text);
        btn.textContent = "Copied";
        btn.setAttribute("data-copied", "true");
        announce("Copied command to clipboard.");
        resetButtonLater(btn);
      } catch {
        selectElementText(codeEl);
        btn.textContent = "Select & copy";
        btn.removeAttribute("data-copied");
        announce("Clipboard access unavailable. Command text selected; press Ctrl+C or Cmd+C to copy.");
        resetButtonLater(btn);
      }
    });
  });

  const demoVideo = document.querySelector(".demo-viewport video");
  if (demoVideo && window.matchMedia("(prefers-reduced-motion: reduce)").matches) {
    demoVideo.removeAttribute("autoplay");
    demoVideo.pause();
    demoVideo.controls = true;
  }
})();
