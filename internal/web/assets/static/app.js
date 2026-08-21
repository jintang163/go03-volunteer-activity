const TOKEN_KEY = "va_token";

function token() { return localStorage.getItem(TOKEN_KEY) || ""; }
function setToken(t) { localStorage.setItem(TOKEN_KEY, t); }
function clearToken() { localStorage.removeItem(TOKEN_KEY); }

async function api(path, opts = {}) {
  const headers = Object.assign({ "Content-Type": "application/json" }, opts.headers || {});
  const t = token();
  if (t) headers.Authorization = "Bearer " + t;
  const res = await fetch(path, Object.assign({}, opts, { headers }));
  if (res.status === 204) return null;
  const text = await res.text();
  let data = null;
  try { data = text ? JSON.parse(text) : null; } catch { data = { message: text }; }
  if (!res.ok) {
    const msg = (data && (data.message || data.code)) || ("HTTP " + res.status);
    throw new Error(msg);
  }
  return data;
}

function requireLogin() {
  if (!token()) { location.href = "/login"; return false; }
  return true;
}

async function currentMe() { return api("/api/auth/me"); }

function logout() {
  api("/api/auth/logout", { method: "POST" }).catch(() => {});
  clearToken();
  location.href = "/login";
}

function $(id) { return document.getElementById(id); }

function fillUserBar(me) {
  const el = $("userbar");
  if (!el || !me) return;
  const u = me.user || me;
  el.innerHTML = `<span>${u.display_name || u.username} · ${u.role} · ${u.total_minutes||0} 分钟 · ${u.points||0} 分</span>
    <a href="/app">广场</a> <a href="/me">我的</a>
    ${u.role === "organizer" || u.role === "admin" ? '<a href="/organizer">组织</a>' : ""}
    ${u.role === "admin" ? '<a href="/admin">后台</a>' : ""}
    <button class="btn btn-ghost" onclick="logout()">退出</button>`;
}

function catLabel(id) {
  const m = {environment:"环保",elderly:"助老",education:"支教",community:"社区服务",emergency:"应急",culture:"文化",sports:"体育",other:"其他"};
  return m[id] || id;
}

function statusLabel(id) {
  const m = {draft:"草稿",published:"已发布",registration_closed:"报名截止",in_progress:"进行中",completed:"已完成",cancelled:"已取消",
    pending:"待审",approved:"已录取",waitlisted:"候补",rejected:"已拒绝",no_show:"缺席"};
  return m[id] || id;
}

window.VA = { api, token, setToken, clearToken, requireLogin, currentMe, logout, $, fillUserBar, catLabel, statusLabel };
