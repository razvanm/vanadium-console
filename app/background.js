function onLaunched(launchData) {
    console.log("onLaunched: " + JSON.stringify(launchData));
    chrome.identity.getAuthToken({interactive: false}, function(token) {
        console.log("token: " + token);
        chrome.app.window.create("index.html", null, function(createdWindow) {
            createdWindow.contentWindow.token = token;
        });
    });
}

chrome.app.runtime.onLaunched.addListener(onLaunched);