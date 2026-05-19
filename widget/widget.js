(function() {
  var thisScript = document.currentScript;
  var BACKEND_URL = new URL(thisScript.src).origin;
  var clientId = location.hostname || 'unknown';

  var MAX_QUEUE = 500;
  var FETCH_TIMEOUT_MS = 10000;

  var sessionId;
  var flushInterval;
  var queue = [];
  var flushTimer = null;
  var clickListener = null;
  var badgeHost = null;
  var badgeShadowCount = null;

  function makeEvent(eventType, metadata) {
    return {
      eventId: crypto.randomUUID(),
      clientId: clientId,
      sessionId: sessionId,
      eventType: eventType,
      timestamp: new Date().toISOString(),
      metadata: metadata || {}
    };
  }

  function capQueue() {
    if (queue.length > MAX_QUEUE) {
      queue = queue.slice(queue.length - MAX_QUEUE);
    }
  }

  function handleCommand(cmd) {
    if (!Array.isArray(cmd) || cmd[0] !== 'track') return;
    queue.push(makeEvent(cmd[1], cmd[2] || {}));
    capQueue();
  }

  function flush() {
    if (queue.length === 0) return;
    var batch = queue.slice();
    queue = [];

    var controller = new AbortController();
    var timeoutId = setTimeout(function() { controller.abort(); }, FETCH_TIMEOUT_MS);

    fetch(BACKEND_URL + '/events', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(batch),
      signal: controller.signal
    }).then(function(r) {
      clearTimeout(timeoutId);
      if (!r.ok) throw new Error('http ' + r.status);
    }).catch(function() {
      clearTimeout(timeoutId);
      queue = batch.concat(queue);
      capQueue();
    });
  }

  function updateBadge() {
    fetch(BACKEND_URL + '/stats/' + encodeURIComponent(clientId))
      .then(function(r) { return r.json(); })
      .then(function(json) {
        var count = (json.data && json.data.sessionCount) || 0;
        if (badgeShadowCount) {
          badgeShadowCount.textContent = count + ' active viewer' + (count !== 1 ? 's' : '');
        }
      })
      .catch(function() {});
  }

  function tick() {
    flush();
    updateBadge();
  }

  function injectBadge() {
    badgeHost = document.createElement('div');
    var shadow = badgeHost.attachShadow({ mode: 'closed' });

    var style = document.createElement('style');
    style.textContent = `
      .badge {
        position: fixed;
        bottom: 16px;
        right: 16px;
        background: #1a1a2e;
        color: #ffffff;
        padding: 8px 14px;
        border-radius: 20px;
        font-family: sans-serif;
        font-size: 13px;
        display: flex;
        align-items: center;
        gap: 8px;
        z-index: 2147483647;
        box-shadow: 0 2px 10px rgba(0,0,0,0.4);
        cursor: default;
      }
      .dot {
        width: 8px;
        height: 8px;
        background: #00d084;
        border-radius: 50%;
        flex-shrink: 0;
      }
    `;

    var badge = document.createElement('div');
    badge.className = 'badge';

    var dot = document.createElement('span');
    dot.className = 'dot';

    badgeShadowCount = document.createElement('span');
    badgeShadowCount.textContent = '0 active viewers';

    badge.appendChild(dot);
    badge.appendChild(badgeShadowCount);
    shadow.appendChild(style);
    shadow.appendChild(badge);
    document.body.appendChild(badgeHost);

    updateBadge();
  }

  function destroy() {
    if (flushTimer) {
      clearInterval(flushTimer);
      flushTimer = null;
    }
    if (clickListener) {
      document.removeEventListener('click', clickListener);
      clickListener = null;
    }
    if (badgeHost && badgeHost.parentNode) {
      badgeHost.parentNode.removeChild(badgeHost);
      badgeHost = null;
    }
    badgeShadowCount = null;
    queue = [];
    delete window._tracker;
  }

  function startup() {
    sessionId = sessionStorage.getItem('_tid');
    if (!sessionId) {
      sessionId = crypto.randomUUID();
      sessionStorage.setItem('_tid', sessionId);
    }

    queue.push(makeEvent('pageview', { url: location.href, title: document.title }));

    clickListener = function(e) {
      var el = e.target.closest('[data-track]');
      if (el) {
        queue.push(makeEvent('click', {
          label: el.getAttribute('data-track'),
          tag: el.tagName.toLowerCase()
        }));
        capQueue();
      }
    };
    document.addEventListener('click', clickListener);

    var preload = window._tracker;
    if (Array.isArray(preload)) {
      for (var i = 0; i < preload.length; i++) {
        handleCommand(preload[i]);
      }
    }

    window._tracker = {
      push: handleCommand,
      destroy: destroy
    };

    flushTimer = setInterval(tick, flushInterval);
    injectBadge();
  }

  function init() {
    fetch(BACKEND_URL + '/config/' + encodeURIComponent(clientId))
      .then(function(r) { return r.json(); })
      .then(function(json) {
        flushInterval = (json.data && json.data.flushInterval) || 5000;
      })
      .catch(function() {
        flushInterval = 5000;
      })
      .then(function() {
        startup();
      });
  }

  init();
})();
