{ /* global localStorage fetch */
  const b = document.body
  const see = () => {
    b.textContent = ''
    fetch('/mainview.html').then(r => r.text()).then(v => { b.innerHTML = v })
  }

  localStorage.getItem('see') ? see()
    : document.querySelector('div.come-see').addEventListener('click', e => {
      b.className = 'transition-view'
      see()
      setTimeout(() => { b.className = '' }, 600)
      localStorage.setItem('see', true)
    }, {once: true})
}
