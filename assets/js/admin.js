/* global localStorage fetch */
try { localStorage.getItem('cookie-ok') && document.querySelector('.cookie-toast').remove() }catch(e) {}

const cacheScript = (
  url,
  fn,
  fresh = !localStorage.getItem('fresh') || location.host.includes('localhost'),
  cached = localStorage.getItem(url)
) => {
  if (cached != null && !fresh) return fn(cached)
  fetch(url).then(r => r.text()).then(src => localStorage.setItem(url, fn(src) || src))
}

cacheScript('/js/admin-view.js', src => {
  const script = document.createElement('script')
  script.textContent += `\n;rilti.run(() => {\n${src}\n});\n`
  document.head.appendChild(script)
  localStorage.setItem('fresh', false)
})
