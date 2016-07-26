var constants = require('./constants');

window.onmessage = function(event) {
  console.log(event);
  switch (event.data.type) {
  case constants.URL:
    console.log('Update head with %d bytes and body with %d bytes',
        event.data.data.head.length, event.data.data.body.length);
    document.getElementsByTagName('head').item(0).innerHTML =
        event.data.data.head;
    document.getElementsByTagName('body').item(0).innerHTML =
        event.data.data.body;
    break;
  }
}

window.onload = function() {
  console.log(window.location);
  if (window.location.search != '') {
    // This will be triggered when the user clicks on a link.
    parent.postMessage({
      type: constants.URL,
      data: window.location.search
    }, '*');
  }
}