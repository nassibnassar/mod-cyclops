function sendRequest(method, path, data) {
  const params = { method };
  if (method === 'POST') {
    params.headers = {
      'Content-Type': 'application/json'
    };
    params.body = JSON.stringify(data);
  }
  fetch(path, params)
  .then(async (response) => {
    if (!response.ok) {
      const errorText = await response.text().catch(() => '');
      alert(`Request failed with status ${response.status}: ${errorText}`);
    } else {
      const result = await response.json().catch(() => null);
      alert('Success: ' + JSON.stringify(result, null, 2));
    }
  })
  .catch((error) => {
    alert(`${method} to ${path} failed:`, error);
  });
}

function postData(path, data) {
  sendRequest('POST', path, data);
}

function deleteObject(path) {
  sendRequest('DELETE', path);
}

function addRecord() {
  postData(
    '/cyclops/sets/test.mike/add',
    {
      from: 'reserve',
      cond: "author = 'Adams, John'",
      // filter: 'jurassic', // Not yet supported
      // tag: 'dino,ptero', // Not yet supported
    }
  )
}
