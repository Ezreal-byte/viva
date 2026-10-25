"use strict";

(async () => {
  try {
    if (typeof Go !== "function") throw new Error("The Go WebAssembly loader is unavailable.");
    const runtime = new Go();
    const response = await fetch(document.documentElement.dataset.wasm);
    if (!response.ok) throw new Error(`Preview download failed (${response.status}).`);
    const result = await WebAssembly.instantiate(await response.arrayBuffer(), runtime.importObject);
    await runtime.run(result.instance);
  } catch (error) {
    const message = error instanceof Error ? error.message : "Preview could not start.";
    const status = document.getElementById("preview-status");
    status.hidden = false;
    status.textContent = message;
    parent.postMessage({ type: "viva-preview-error", message }, location.origin);
  }
})();
