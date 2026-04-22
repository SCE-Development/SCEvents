import http from 'k6/http';
import { check, sleep } from 'k6';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8002';

export const options = {
  scenarios: {
    registration_flood: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 100 },   // ramp up to 100
        { duration: '1m',  target: 500 },   // ramp up to 500
        { duration: '30s', target: 1000 },  // peak at 1000
        { duration: '30s', target: 0 },     // ramp down
      ],
    },
  },
  thresholds: {
    // 95th percentile response time under 2 seconds
    http_req_duration: ['p(95)<2000'],
    // less than 5% of requests should fail (non-2xx or network error)
    http_req_failed: ['rate<0.05'],
  },
};

// setup() runs once before any VUs start.
// Creates a handful of events with large capacity to spread load across.
export function setup() {
  const adminToken = 'load-test-admin';
  const headers = {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${adminToken}`,
  };

  const eventIDs = [];
  for (let i = 0; i < 5; i++) {
    const res = http.post(
      `${BASE_URL}/events/`,
      JSON.stringify({
        name: `Load Test Event ${i + 1}`,
        date: '2099-12-31',
        time: '18:00',
        location: 'Load Test Venue',
        description: 'Created by k6 load test — safe to delete',
        max_attendees: 100000,
        status: 'published',
        visibility: 'public',
        registration_form: [],
      }),
      { headers }
    );

    if (res.status === 201 || res.status === 200) {
      const body = JSON.parse(res.body);
      if (body.id) {
        eventIDs.push(body.id);
      }
    }
  }

  if (eventIDs.length === 0) {
    throw new Error('setup failed: could not create any test events');
  }

  console.log(`setup: created ${eventIDs.length} events`);
  return { eventIDs };
}

// default function runs for every VU iteration.
// Each VU+iteration gets a unique token so mock Clark issues a unique user_id,
// avoiding duplicate-registration conflicts.
export default function ({ eventIDs }) {
  const userToken = `load-test-user-${__VU}-${__ITER}`;
  // Spread load evenly across the seeded events
  const eventID = eventIDs[__VU % eventIDs.length];

  const res = http.post(
    `${BASE_URL}/events/${eventID}/register`,
    JSON.stringify({
      registrant: {
        name: `Load Test User ${__VU}`,
        email: `loadtest-${__VU}-${__ITER}@example.com`,
        user_id: userToken,
      },
      registration_form_answers: {},
    }),
    {
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${userToken}`,
      },
    }
  );

  check(res, {
    'registration accepted (202)': (r) => r.status === 202,
    'response has request_id': (r) => {
      try { return !!JSON.parse(r.body).request_id; } catch { return false; }
    },
  });

  // small think time to avoid thundering herd on first iteration
  sleep(0.1);
}
