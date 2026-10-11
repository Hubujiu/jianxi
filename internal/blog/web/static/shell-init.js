// Apply remembered sidebar geometry before the first styled paint.
try {
  const collapsed = localStorage.getItem('jianxi-sidebar-collapsed') === 'true';
  const width = Math.max(208, Math.min(300, Number(localStorage.getItem('jianxi-sidebar-width')) || 248));
  document.documentElement.dataset.sidebarCollapsed = String(collapsed);
  document.documentElement.style.setProperty('--side', `${collapsed ? 64 : width}px`);
} catch {}
