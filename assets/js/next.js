
{ /* global localStorage fetch */
  const b = document.body

  const cacheScript = (
    url,
    fn,
    fresh = !localStorage.getItem('fresh') || location.host.includes('localhost'),
    cached = localStorage.getItem(url)
  ) => {
    if (cached != null && !fresh) return fn(cached)
    fetch(url).then(r => r.text()).then(src => localStorage.setItem(url, fn(src) || src))
  }

  cacheScript(
    location.host.includes('localhost') ? 'http://localhost:2018/dist/rilti.js' : 'https://rawgit.com/SaulDoesCode/rilti.js/experimental/dist/rilti.min.js',
    src => {
      const script = document.createElement('script')
      script.textContent = src + ';\n;'
      cacheScript('/assets/js/next-view.js', src => {
        script.textContent += `\n;\nrilti.run(() => {\n${src}\n});\n`
        document.head.appendChild(script)
        localStorage.setItem('fresh', false)
      })
    }
  )
  if (localStorage.getItem('cookie-ok')) {
    document.querySelector('.cookie-toast').remove()
  }
}
