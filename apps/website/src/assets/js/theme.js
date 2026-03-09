function toggleDarkMode() {
  var isDark = document.documentElement.classList.toggle('dark');
  localStorage.setItem('theme', isDark ? 'dark' : 'light');
}

document.addEventListener('click', function (e) {
  var btn = e.target.closest('[data-action]');
  if (!btn) return;
  if (btn.dataset.action === 'toggle-dark') toggleDarkMode();
  if (btn.dataset.action === 'toggle-menu') {
    var target = document.getElementById(btn.dataset.target);
    if (target) target.classList.toggle('hidden');
  }
});
