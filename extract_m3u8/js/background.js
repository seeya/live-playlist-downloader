const browser = window.browser || window.chrome;
const r = browser.webRequest;

const trackUrls = {
  urls: ["<all_urls>"],
};

const isFF = window.browser ? true : false;
const reqBodyHeaders = isFF ? ["requestBody"] : ["requestBody", "extraHeaders"];
const resHeaders = isFF ? ["responseHeaders"] : ["responseHeaders", "extraHeaders"];

const pattern = [".m3u8"];

r.onCompleted.addListener(
  function (details) {
    for (let i = 0; i < pattern.length; i++) {
      const p = pattern[i];

      if (details.url.indexOf(p) != -1) {
        console.log(details);

        chrome.tabs.query(
          {},
          function (tabs) {
            let tab = tabs.find(e => e.id == details.tabId)

            let title = "";
            if (tab) {
              title = tab.title;
              originUrl = tab.url
            }

            fetch("http://localhost:3002", {
              method: "POST",
              headers: {
                "Content-type": "application/json",
              },
              body: JSON.stringify({
                url: details.url,
                initiator: details.initiator,
                documentId: details.documentId,
                title,
                originUrl
              }),
            });
          }
        )
      }
    }
  },
  trackUrls,
  resHeaders
);

