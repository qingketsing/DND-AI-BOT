import http from 'k6/http';
import { check, sleep } from 'k6';
import exec from 'k6/execution';
import { Rate, Trend } from 'k6/metrics';

const baseURL = (__ENV.BASE_URL || 'http://localhost:18080').replace(/\/+$/, '');
const email = __ENV.EMAIL || '';
const password = __ENV.PASSWORD || '';
const channel = __ENV.CHANNEL || 'web';
const authCookieName = 'dnd_auth_session';
const requestTimeout = __ENV.REQUEST_TIMEOUT || '30s';
const thinkTimeSeconds = Number(__ENV.THINK_TIME_SECONDS || '0');
const stagesEnv = __ENV.STAGES || '30:30s,50:30s,100:30s';
const preAllocatedVUs = Number(__ENV.PREALLOCATED_VUS || '200');
const maxVUs = Number(__ENV.MAX_VUS || '400');
const sessionsPerVU = Number(__ENV.SESSIONS_PER_VU || '1');

const acceptedFailureRate = new Rate('accepted_failure_rate');
const acceptedDuration = new Trend('accepted_duration_ms', true);

export const options = {
  scenarios: buildScenarios(stagesEnv),
  thresholds: {
    http_req_failed: ['rate<0.05'],
    accepted_failure_rate: ['rate<0.05'],
  },
};

export function setup() {
  if (!email || !password) {
    throw new Error('EMAIL and PASSWORD are required');
  }

  const loginResponse = http.post(
    `${baseURL}/auth/login`,
    JSON.stringify({ email, password }),
    {
      headers: { 'Content-Type': 'application/json' },
      tags: { endpoint: 'login' },
      timeout: requestTimeout,
    },
  );

  const loginOK = check(loginResponse, {
    'login status is 200': (r) => r.status === 200,
  });
  if (!loginOK) {
    throw new Error(`login failed: status=${loginResponse.status} body=${loginResponse.body}`);
  }

  const authToken = extractAuthToken(loginResponse);
  if (!authToken) {
    throw new Error(`login response missing ${authCookieName} cookie`);
  }

  const totalSessions = maxVUs * sessionsPerVU;
  const sessionIDs = [];
  for (let index = 0; index < totalSessions; index += 1) {
    const createResponse = http.post(
      `${baseURL}/sessions`,
      JSON.stringify({ channel }),
      {
        headers: authHeaders(authToken),
        tags: { endpoint: 'create_session' },
        timeout: requestTimeout,
      },
    );
    if (createResponse.status !== 201) {
      throw new Error(`create session failed at index=${index}: status=${createResponse.status} body=${createResponse.body}`);
    }
    const data = JSON.parse(createResponse.body);
    if (!data.id) {
      throw new Error(`create session missing id at index=${index}: ${createResponse.body}`);
    }
    sessionIDs.push(data.id);
  }

  return {
    authToken,
    sessionIDs,
    sessionsPerVU,
  };
}

export default function (data) {
  return submitAcceptedMessage(data);
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

function buildScenarios(raw) {
  const stages = parseStages(raw);
  const scenarios = {};
  let offsetSeconds = 0;
  for (let index = 0; index < stages.length; index += 1) {
    const stage = stages[index];
    scenarios[`asyncAcceptedFixedRate${index + 1}`] = {
      executor: 'constant-arrival-rate',
      rate: stage.target,
      timeUnit: '1s',
      duration: stage.duration,
      preAllocatedVUs,
      maxVUs,
      exec: 'submitAcceptedMessage',
      startTime: `${offsetSeconds}s`,
    };
    offsetSeconds += durationToSeconds(stage.duration);
  }
  return scenarios;
}

function durationToSeconds(raw) {
  const value = String(raw).trim();
  if (/^\d+s$/.test(value)) {
    return Number(value.slice(0, -1));
  }
  if (/^\d+m$/.test(value)) {
    return Number(value.slice(0, -1)) * 60;
  }
  if (/^\d+h$/.test(value)) {
    return Number(value.slice(0, -1)) * 3600;
  }
  throw new Error(`unsupported duration for staged fixed rate: ${raw}`);
}

function authHeaders(authToken) {
  return {
    'Content-Type': 'application/json',
    Cookie: `${authCookieName}=${authToken}`,
  };
}

function extractAuthToken(response) {
  const cookies = response.cookies || {};
  const authCookies = cookies[authCookieName];
  if (Array.isArray(authCookies) && authCookies.length > 0 && authCookies[0].value) {
    return authCookies[0].value;
  }
  return '';
}

export function submitAcceptedMessage(data) {
  const vuIndex = exec.vu.idInTest - 1;
  const localIndex = exec.vu.iterationInScenario % data.sessionsPerVU;
  const sessionIndex = vuIndex * data.sessionsPerVU + localIndex;
  const sessionID = data.sessionIDs[sessionIndex];
  if (!sessionID) {
    throw new Error(`missing session for vu=${vuIndex} localIndex=${localIndex}`);
  }

  const content = `async accepted benchmark vu=${vuIndex} iter=${exec.vu.iterationInScenario}`;
  const response = http.post(
    `${baseURL}/sessions/${sessionID}/messages`,
    JSON.stringify({ content }),
    {
      headers: authHeaders(data.authToken),
      tags: {
        endpoint: 'messages_async_accepted',
      },
      timeout: requestTimeout,
    },
  );

  const ok = check(response, {
    'messages status is 202': (r) => r.status === 202,
    'messages response has queued status': (r) => {
      try {
        const payload = JSON.parse(r.body);
        return payload.status === 'queued' && typeof payload.message_id === 'string' && typeof payload.job_id === 'string';
      } catch (_) {
        return false;
      }
    },
  });

  acceptedDuration.add(response.timings.duration);
  acceptedFailureRate.add(!ok);

  if (thinkTimeSeconds > 0) {
    sleep(thinkTimeSeconds);
  }
}
