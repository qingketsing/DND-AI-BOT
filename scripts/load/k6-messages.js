import http from 'k6/http';
import { check, sleep } from 'k6';
import exec from 'k6/execution';
import { Rate, Trend } from 'k6/metrics';

const baseURL = (__ENV.BASE_URL || 'http://localhost:8080').replace(/\/+$/, '');
const email = __ENV.EMAIL || '';
const password = __ENV.PASSWORD || '';
const channel = __ENV.CHANNEL || 'web';
const messageMode = (__ENV.MESSAGE_MODE || 'L1').toUpperCase();
const thinkTimeSeconds = Number(__ENV.THINK_TIME_SECONDS || '1');
const stagesEnv = __ENV.STAGES || '1:3m,2:3m,5:3m,10:3m';

const messageFailureRate = new Rate('message_failure_rate');
const messageDuration = new Trend('message_duration_ms', true);

const payloads = {
  L1: [
    '我们说到哪了',
    '介绍一下当前设定',
    '当前的目标是什么',
  ],
  L2: [
    '我检查房间并观察周围',
    '创建一个高等精灵法师，使用标准数组',
    '我想检查一下门后有没有动静',
  ],
  L3: [
    '继续攻击并结算伤害',
    '根据当前 encounter 推进战斗并说明下一步',
    '我进行一次战斗动作，并按当前状态完整结算',
  ],
};

const selectedPayloads = payloads[messageMode] || payloads.L1;

export const options = {
  stages: parseStages(stagesEnv),
  thresholds: {
    http_req_failed: ['rate<0.01'],
    message_failure_rate: ['rate<0.01'],
    message_duration_ms: ['p(95)<120000', 'p(99)<180000'],
  },
};

let initialized = false;
let sessionID = '';

export default function () {
  ensureSession();

  const content = selectedPayloads[exec.scenario.iterationInTest % selectedPayloads.length];
  const body = JSON.stringify({ content });
  const response = http.post(`${baseURL}/sessions/${sessionID}/messages`, body, {
    headers: { 'Content-Type': 'application/json' },
    tags: {
      endpoint: 'messages',
      message_mode: messageMode,
      session_id: sessionID,
    },
    timeout: __ENV.REQUEST_TIMEOUT || '180s',
  });

  const ok = check(response, {
    'messages status is 200': (r) => r.status === 200,
    'messages returns session id': (r) => {
      try {
        const data = JSON.parse(r.body);
        return typeof data.id === 'string' && data.id.length > 0;
      } catch (_) {
        return false;
      }
    },
    'messages returns history': (r) => {
      try {
        const data = JSON.parse(r.body);
        return Array.isArray(data.history);
      } catch (_) {
        return false;
      }
    },
  });

  messageDuration.add(response.timings.duration, { message_mode: messageMode });
  messageFailureRate.add(!ok, { message_mode: messageMode });

  sleep(thinkTimeSeconds);
}

function ensureSession() {
  if (initialized) {
    return;
  }
  if (!email || !password) {
    throw new Error('EMAIL and PASSWORD are required');
  }

  const loginResponse = http.post(
    `${baseURL}/auth/login`,
    JSON.stringify({ email, password }),
    {
      headers: { 'Content-Type': 'application/json' },
      tags: { endpoint: 'login' },
      timeout: __ENV.REQUEST_TIMEOUT || '180s',
    },
  );

  const loginOK = check(loginResponse, {
    'login status is 200': (r) => r.status === 200,
  });
  if (!loginOK) {
    throw new Error(`login failed: status=${loginResponse.status} body=${loginResponse.body}`);
  }

  const createResponse = http.post(
    `${baseURL}/sessions`,
    JSON.stringify({ channel }),
    {
      headers: { 'Content-Type': 'application/json' },
      tags: { endpoint: 'create_session' },
      timeout: __ENV.REQUEST_TIMEOUT || '180s',
    },
  );

  const createOK = check(createResponse, {
    'create session status is 201': (r) => r.status === 201,
  });
  if (!createOK) {
    throw new Error(`create session failed: status=${createResponse.status} body=${createResponse.body}`);
  }

  const data = JSON.parse(createResponse.body);
  if (!data.id) {
    throw new Error(`create session response missing id: ${createResponse.body}`);
  }

  sessionID = data.id;
  initialized = true;
}

function parseStages(raw) {
  return String(raw)
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
    .map((item) => {
      const parts = item.split(':');
      if (parts.length !== 2) {
        throw new Error(`invalid STAGES item: ${item}`);
      }
      const target = Number(parts[0]);
      const duration = parts[1].trim();
      if (!Number.isFinite(target) || target <= 0 || !duration) {
        throw new Error(`invalid STAGES item: ${item}`);
      }
      return { target, duration };
    });
}
