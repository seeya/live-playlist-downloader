const browser = window.browser || window.chrome;
const r = browser.webRequest;

const trackUrls = {
  urls: ["<all_urls>"],
};

const isFF = window.browser ? true : false;
const reqBodyHeaders = isFF ? ["requestBody"] : ["requestBody", "extraHeaders"];
const resHeaders = isFF ? ["responseHeaders"] : ["responseHeaders", "extraHeaders"];

const pattern = ["index-f1-v1-a1.m3u8", "index-v1-a1.m3u8"];

r.onCompleted.addListener(
  function (details) {
    for (let i = 0; i < pattern.length; i++) {
      const p = pattern[i];

      if (details.url.indexOf(p) != -1) {
        fetch("http://localhost:3000", {
          method: "POST",
          headers: {
            "Content-type": "application/json",
          },
          body: JSON.stringify({
            url: details.url,
          }),
        });
      }
    }
  },
  trackUrls,
  resHeaders
);
