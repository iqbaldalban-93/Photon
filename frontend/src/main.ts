if (location.hash === '#tray') {
  await import('./tray')
} else {
  await import('./app')
}

export {}
