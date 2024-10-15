import http from 'k6/http';

export default function () {
  const res = http.get('https://httpbin.org/status/200');
}