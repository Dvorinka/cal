// Setup screen poller: refreshes into the real app once the backend is up,
// shows the error + retry path when init fails.
var detail = document.getElementById("detail");
var spin = document.getElementById("spin");
var errorBox = document.getElementById("error");
var errMsg = document.getElementById("errmsg");
var logPath = document.getElementById("logpath");
var retryBtn = document.getElementById("retry");

function poll() {
  fetch("/__setup/status", { cache: "no-store" })
    .then(function (r) { return r.json(); })
    .then(function (s) {
      if (s.ready) {
        location.replace("/");
        return;
      }
      if (s.error) {
        showError(s);
        return;
      }
      detail.textContent = s.detail || "Starting…";
      setTimeout(poll, 700);
    })
    .catch(function () { setTimeout(poll, 1500); });
}

function showError(s) {
  spin.style.display = "none";
  detail.style.display = "none";
  errMsg.textContent = s.error;
  logPath.textContent = s.log || "";
  errorBox.style.display = "block";
}

retryBtn.onclick = function () {
  retryBtn.disabled = true;
  fetch("/__setup/retry", { method: "POST" }).finally(function () {
    retryBtn.disabled = false;
    errorBox.style.display = "none";
    spin.style.display = "";
    detail.style.display = "";
    poll();
  });
};

poll();
