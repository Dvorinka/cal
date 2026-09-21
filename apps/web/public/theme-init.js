// Apply the cached theme before first paint to avoid a flash.
try {
  var s = JSON.parse(localStorage.getItem("cal:settings") || "{}");
  var t = s.theme || "system";
  var dark = t === "dark" || (t === "system" && matchMedia("(prefers-color-scheme: dark)").matches);
  document.documentElement.dataset.theme = dark ? "dark" : "light";
} catch (e) {}
