import { check } from 'k6';
import http from 'k6/http';

export const options = {
  scenarios: {
    api: {
      executor: 'shared-iterations',
    },
  },
};

export default function () {
  const res = http.request('GET', 'http://localhost:3000');
  check(res, { 'status ok': res.status === 200 });
}
