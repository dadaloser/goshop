import http from 'k6/http';
import { check, sleep } from 'k6';

const base = __ENV.BASE_URL || 'http://127.0.0.1:8049/v1';
const token = __ENV.ACCESS_TOKEN || '';
const goodsID = __ENV.GOODS_ID || '1';

export const options = {
  scenarios: { core: { executor: 'constant-arrival-rate', rate: Number(__ENV.TPS || 200), timeUnit: '1s', duration: __ENV.DURATION || '10m', preAllocatedVUs: 100, maxVUs: 1000 } },
  thresholds: { http_req_duration: ['p(95)<500', 'p(99)<1000'], http_req_failed: ['rate<0.01'], http_reqs: ['rate>190'] },
};

export default function () {
  const headers = token ? { Authorization: `Bearer ${token}` } : {};
  const list = http.get(`${base}/goods?page=1&page_per_nums=20`, { headers, tags: { flow: 'browse' } });
  check(list, { 'goods list 2xx': (r) => r.status >= 200 && r.status < 300 });
  const detail = http.get(`${base}/goods/${goodsID}/reviews`, { headers, tags: { flow: 'review' } });
  check(detail, { 'review list 2xx': (r) => r.status >= 200 && r.status < 300 });
  if (token) {
    const orders = http.get(`${base}/user/orders?page=1&page_per_nums=10`, { headers, tags: { flow: 'orders' } });
    check(orders, { 'order list 2xx': (r) => r.status >= 200 && r.status < 300 });
  }
  sleep(0.1);
}
