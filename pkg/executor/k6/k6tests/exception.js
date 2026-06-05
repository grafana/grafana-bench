// Throws an uncaught exception inside the iteration. k6 logs the error with
// `source=stacktrace` and still exits 0, so bench must detect it to avoid
// reporting the test as passed.
export default function () {
  const client = undefined;
  client.request('GET', 'http://localhost');
}
