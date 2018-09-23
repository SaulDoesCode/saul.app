/* global localStorage fetch */
try { localStorage.getItem('cookie-ok') && document.querySelector('.cookie-toast').remove() } catch (e) {}

const cacheScript = (
  url,
  fn,
  fresh = !localStorage.getItem('fresh') || location.host.includes('localhost'),
  cached = localStorage.getItem(url)
) => {
  if (cached != null && !fresh) return fn(cached)
  fetch(url).then(r => r.text()).then(src => localStorage.setItem(url, fn(src) || src))
}
const isDevBox = location.host.includes('localhost')
cacheScript(isDevBox
  ? 'http://localhost:2018/dist/rilti.js'
  : 'https://rawgit.com/SaulDoesCode/rilti.js/experimental/dist/rilti.min.js',
  src => {
    const script = document.createElement('script')
    script.textContent = src + ';\n'
    cacheScript("/js/showdown.min.js", src => {
      script.textContent += src + ';\n'
      cacheScript('/js/next-view.js', src => {
        script.textContent += `\n;rilti.run(() => {\n${src}\n});\n`
        document.head.appendChild(script)
        localStorage.setItem('fresh', false)
      }, false)
    }, true)
  }
)
