// k6 load test for `girl serve`.
//
// Simulates coding agents calling GIRL in their edit loop: ask for the next
// fixes on a small change, occasionally rescan a whole monorepo. Run it
// against servers pinned to different CPU/memory budgets
// (scripts/k6-hardware.sh) to see what hardware an agent fleet needs.
//
//   k6 run -e GIRL_URL=http://127.0.0.1:7777 bench/k6/girl.js
//
// Environment:
//   GIRL_URL       server base URL                      (http://127.0.0.1:7777)
//   PROFILE        label for this hardware profile      (local)
//   SMALL_PATH     path for agent-sized requests        (small)
//   LARGE_PATH     path for repo-wide scans             (large)
//   VUS            concurrent simulated agents          (4)
//   DURATION       test length                          (30s)
//   THINK          seconds an agent thinks between calls (0.5)
//   P95_MS         p95 budget for agent requests, ms    (2000)
//   SUMMARY_DIR    where to write <profile>.json/.md    (unset: stdout only)
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend, Counter } from 'k6/metrics';

const BASE = __ENV.GIRL_URL || 'http://127.0.0.1:7777';
const PROFILE = __ENV.PROFILE || 'local';
const SMALL = __ENV.SMALL_PATH || 'small';
const LARGE = __ENV.LARGE_PATH || 'large';
const VUS = parseInt(__ENV.VUS || '4', 10);
const DURATION = __ENV.DURATION || '30s';
const THINK = parseFloat(__ENV.THINK || '0.5');
const P95 = parseInt(__ENV.P95_MS || '2000', 10);

const queueMs = new Trend('girl_queue_ms', true);
const analysisMs = new Trend('girl_analysis_ms', true);
const filesAnalyzed = new Counter('girl_files_analyzed');

export const options = {
  scenarios: {
    // Agents iterating on a change: girl next on a small target.
    agent_loop: {
      executor: 'constant-vus',
      exec: 'agentLoop',
      vus: VUS,
      duration: DURATION,
      tags: { scenario: 'agent_loop' },
    },
    // A CI job or a fresh session scanning the whole monorepo.
    repo_scan: {
      executor: 'constant-vus',
      exec: 'repoScan',
      vus: 1,
      duration: DURATION,
      tags: { scenario: 'repo_scan' },
    },
  },
  thresholds: {
    'http_req_failed': ['rate<0.01'],
    'http_req_duration{scenario:agent_loop}': [`p(95)<${P95}`],
    'checks': ['rate>0.99'],
    // Always-true thresholds make k6 report these per-scenario sub-metrics.
    'http_req_duration{scenario:repo_scan}': ['max>=0'],
    'http_reqs{scenario:agent_loop}': ['count>=0'],
    'http_reqs{scenario:repo_scan}': ['count>=0'],
  },
  summaryTrendStats: ['avg', 'med', 'p(90)', 'p(95)', 'max'],
};

const headers = { 'Content-Type': 'application/json' };

function recordTiming(res) {
  // Server-Timing: queue;dur=1.2, analysis;dur=34.5
  const timing = res.headers['Server-Timing'] || '';
  const q = /queue;dur=([\d.]+)/.exec(timing);
  const a = /analysis;dur=([\d.]+)/.exec(timing);
  if (q) queueMs.add(parseFloat(q[1]));
  if (a) analysisMs.add(parseFloat(a[1]));
  const files = parseInt(res.headers['X-Girl-Files'] || '0', 10);
  if (files > 0) filesAnalyzed.add(files);
}

export function setup() {
  const res = http.get(`${BASE}/healthz`);
  if (res.status !== 200) {
    throw new Error(`girl serve not healthy at ${BASE}: ${res.status}`);
  }
  return res.json();
}

export function agentLoop() {
  const res = http.post(`${BASE}/v1/next`,
    JSON.stringify({ path: SMALL, limit: 5, excerptLines: 12 }), { headers, tags: { scenario: 'agent_loop' } });
  check(res, {
    'next: 200': (r) => r.status === 200,
    'next: has items': (r) => r.status === 200 && r.json('items').length > 0,
    'next: items located': (r) => r.status === 200 && r.json('items').every((i) => i.line > 0),
  });
  recordTiming(res);
  sleep(THINK);
}

export function repoScan() {
  const res = http.post(`${BASE}/v1/benchmark`,
    JSON.stringify({ path: LARGE }), { headers, tags: { scenario: 'repo_scan' }, timeout: '300s' });
  check(res, {
    'benchmark: 200': (r) => r.status === 200,
    'benchmark: scanned files': (r) => r.status === 200 && r.json('summary.filesScanned') > 0,
  });
  recordTiming(res);
}

function metric(data, name, stat) {
  const m = data.metrics[name];
  return m && m.values[stat] !== undefined ? m.values[stat] : 0;
}

export function handleSummary(data) {
  const health = data.setup_data || {};
  const row = {
    profile: PROFILE,
    workers: health.workers,
    maxConcurrent: health.maxConcurrent,
    agentRps: metric(data, 'http_reqs{scenario:agent_loop}', 'rate') || metric(data, 'iterations', 'rate'),
    agentP50: metric(data, 'http_req_duration{scenario:agent_loop}', 'med'),
    agentP95: metric(data, 'http_req_duration{scenario:agent_loop}', 'p(95)'),
    scanAvg: metric(data, 'http_req_duration{scenario:repo_scan}', 'avg'),
    scans: metric(data, 'http_reqs{scenario:repo_scan}', 'count'),
    queueP95: metric(data, 'girl_queue_ms', 'p(95)'),
    analysisP95: metric(data, 'girl_analysis_ms', 'p(95)'),
    failed: metric(data, 'http_req_failed', 'rate'),
    thresholdsOk: Object.values(data.metrics).every((m) => !m.thresholds ||
      Object.values(m.thresholds).every((t) => t.ok)),
  };
  const fmt = (v) => (v || 0).toFixed(0);
  const md = `| ${row.profile} | ${row.workers ?? '?'} | ${fmt(row.agentP50)} | ${fmt(row.agentP95)} | ` +
    `${(row.agentRps || 0).toFixed(1)} | ${fmt(row.queueP95)} | ${(row.scanAvg / 1000).toFixed(2)}s | ` +
    `${(row.failed * 100).toFixed(1)}% | ${row.thresholdsOk ? 'pass' : 'FAIL'} |\n`;

  const text = `\ngirl k6 [${PROFILE}] agent p50 ${fmt(row.agentP50)}ms p95 ${fmt(row.agentP95)}ms, ` +
    `${(row.agentRps || 0).toFixed(1)} req/s, queue p95 ${fmt(row.queueP95)}ms, ` +
    `repo scan avg ${(row.scanAvg / 1000).toFixed(2)}s, thresholds ${row.thresholdsOk ? 'pass' : 'FAIL'}\n`;
  const out = { stdout: text };
  if (__ENV.SUMMARY_DIR) {
    out[`${__ENV.SUMMARY_DIR}/${PROFILE}.json`] = JSON.stringify({ row, metrics: data.metrics }, null, 2);
    out[`${__ENV.SUMMARY_DIR}/${PROFILE}.md`] = md;
  }
  return out;
}
