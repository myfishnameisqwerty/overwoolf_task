/*
 * How to embed on any website — paste this into your <head>:
 *
 *   <script>
 *     window._tracker = window._tracker || [];
 *     (function() {
 *       var s = document.createElement('script');
 *       s.src = 'http://localhost:8080/widget.js';
 *       s.async = true;
 *       document.head.appendChild(s);
 *     })();
 *   </script>
 *
 */

window._tracker = window._tracker || [];
(function() {
  var BACKEND_URL = 'http://localhost:8080';
  var s = document.createElement('script');
  s.src = BACKEND_URL + '/widget.js';
  s.async = true;
  document.head.appendChild(s);
})();
