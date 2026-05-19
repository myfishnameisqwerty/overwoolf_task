(function() {
  const thisScript = document.currentScript;
  const BACKEND_URL = new URL(thisScript.src).origin;
  const clientId = location.hostname || 'unknown';

  const MAX_QUEUE = 500;
  const FETCH_TIMEOUT_MS = 10000;
  const DEFAULT_FLUSH_INTERVAL = 5000;
  const DEFAULT_TRACKED_EVENTS = ['pageview', 'click'];

  let sessionId;
  let flushInterval = DEFAULT_FLUSH_INTERVAL;
  let trackedEvents = DEFAULT_TRACKED_EVENTS;
  let queue = [];
  let flushTimer = null;
  let teardowns = [];
  let badgeHost = null;
  let badgeShadowCount = null;

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

  // To add a new auto-tracked event type:
  //   1. Add an entry below: key = eventType string, value = setup fn that
  //      attaches its listener (or fires its one-shot event) and returns a
  //      teardown fn called from destroy().
  //   2. Include the eventType in the client's trackedEvents in the DB.
  const autoTrackers = {
    pageview: function() {
      queue.push(makeEvent('pageview', { url: location.href, title: document.title }));
      capQueue();
      return function() {};
    },
    click: function() {
      const listener = function(e) {
        const el = e.target.closest('[data-track]');
        if (el) {
          queue.push(makeEvent('click', {
            label: el.getAttribute('data-track'),
            tag: el.tagName.toLowerCase()
          }));
          capQueue();
        }
      };
      document.addEventListener('click', listener);
      return function() { document.removeEventListener('click', listener); };
    },
  };

  function handleCommand(cmd) {
    if (!Array.isArray(cmd) || cmd[0] !== 'track') return;
    queue.push(makeEvent(cmd[1], cmd[2] || {}));
    capQueue();
  }

  function flush() {
    if (queue.length === 0) return;
    const batch = queue.slice();
    queue = [];

    const controller = new AbortController();
    const timeoutId = setTimeout(function() { controller.abort(); }, FETCH_TIMEOUT_MS);

    fetch(BACKEND_URL + '/events', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(batch),
      signal: controller.signal
    }).then(function(r) {
      clearTimeout(timeoutId);
      if (!r.ok) throw new Error('HTTP ' + r.status);
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
        const count = (json.data && json.data.sessionCount) || 0;
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
    const shadow = badgeHost.attachShadow({ mode: 'closed' });

    const style = document.createElement('style');
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

    const badge = document.createElement('div');
    badge.className = 'badge';

    const dot = document.createElement('span');
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
    for (const teardown of teardowns) {
      teardown();
    }
    teardowns = [];
    if (badgeHost && badgeHost.parentNode) {
      badgeHost.parentNode.removeChild(badgeHost);
      badgeHost = null;
    }
    badgeShadowCount = null;
    queue = [];
    window._tracker = [];
  }

  function startup() {
    sessionId = sessionStorage.getItem('_tid');
    if (!sessionId) {
      sessionId = crypto.randomUUID();
      sessionStorage.setItem('_tid', sessionId);
    }

    for (const type of trackedEvents) {
      const factory = autoTrackers[type];
      if (factory) {
        teardowns.push(factory());
      }
    }

    const preload = window._tracker;
    if (Array.isArray(preload)) {
      for (let i = 0; i < preload.length; i++) {
        handleCommand(preload[i]);
      }
      capQueue();
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
        if (json && json.data) {
          flushInterval = json.data.flushInterval || flushInterval;
          if (Array.isArray(json.data.trackedEvents) && json.data.trackedEvents.length > 0) {
            trackedEvents = json.data.trackedEvents;
          }
        }
      })
      .catch(function() {})
      .then(function() { startup(); });
  }

  init();
})();
