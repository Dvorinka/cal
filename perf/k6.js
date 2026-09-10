// k6 smoke/load: login → list entries → create → update → delete.
//   k6 run perf/k6.js -e BASE=http://localhost:8081 -e EMAIL=demo@cal.local -e PASS=password123
import http from "k6/http";
import { check, sleep } from "k6";

export const options = {
  vus: 10,
  duration: "30s",
  thresholds: {
    http_req_failed: ["rate<0.01"],
    http_req_duration: ["p(95)<400"],
  },
};

const BASE = __ENV.BASE || "http://localhost:8081";

export function setup() {
  const res = http.post(
    `${BASE}/api/auth/login`,
    JSON.stringify({ email: __ENV.EMAIL, password: __ENV.PASS }),
    { headers: { "Content-Type": "application/json" } },
  );
  check(res, { "login 200": (r) => r.status === 200 });
  const cookie = res.cookies.cal_session?.[0]?.value;
  return { headers: { "Content-Type": "application/json", Cookie: `cal_session=${cookie}` } };
}

export default function (ctx) {
  const list = http.get(`${BASE}/api/entries`, { headers: ctx.headers });
  check(list, { "entries 200": (r) => r.status === 200 });

  const today = new Date().toISOString().slice(0, 10);
  const created = http.post(
    `${BASE}/api/entries`,
    JSON.stringify({ title: `k6 ${__VU}-${__ITER}`, type: "task", date: today }),
    { headers: ctx.headers },
  );
  check(created, { "create 201": (r) => r.status === 201 });
  const id = created.json("id");
  if (!id) return;

  const patched = http.request(
    "PATCH",
    `${BASE}/api/entries/${id}`,
    JSON.stringify({ completed: true }),
    { headers: ctx.headers },
  );
  check(patched, { "patch 200": (r) => r.status === 200 });

  check(http.del(`${BASE}/api/entries/${id}`, null, { headers: ctx.headers }), {
    "delete 204": (r) => r.status === 204,
  });
  sleep(0.2);
}
