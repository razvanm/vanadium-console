var constants = require('./constants');
var globalToken = '';

chrome.runtime.onConnect.addListener(function(port) {
    port.postMessage({
        type: constants.TOKEN,
        data: globalToken
    });
});

function onLaunched(launchData) {
    console.log('onLaunched: ' + JSON.stringify(launchData));
        chrome.identity.getAuthToken({interactive: false}, function(token) {
            if (chrome.runtime.lastError) {
                console.log('no token: ' + chrome.runtime.lastError.message);
                return;
            }
            globalToken = token;
            console.log('token: ' + token);
            chrome.app.window.create('main.html', null,
                function(createdWindow) {
                    console.log(createdWindow);
                    createdWindow.contentWindow.postMessage({
                        type: constants.TOKEN,
                        data: token
                    }, '*');
                }
            );
        });
}

chrome.app.runtime.onLaunched.addListener(onLaunched);
