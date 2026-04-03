const http = require('http');

http.get('http://localhost:8080/products', (res) => {
  let data = '';
  res.on('data', (chunk) => { data += chunk; });
  res.on('end', () => {
    try {
      const parsed = JSON.parse(data);
      if (parsed.success && Array.isArray(parsed.data.data) && parsed.data.data.length > 0) {
        console.log('SUCCESS: Frontend can correctly receive products array length:', parsed.data.data.length);
        process.exit(0);
      } else {
        console.log('FAIL: Bad API Structure:', data);
        process.exit(1);
      }
    } catch(err) {
      console.log('JSON Parse error', err);
      process.exit(1);
    }
  });
}).on('error', (err) => {
  console.log('Network error', err.message);
  process.exit(1);
});
