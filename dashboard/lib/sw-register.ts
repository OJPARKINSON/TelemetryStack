// Service Worker registration for tile caching
export async function registerServiceWorker() {
  if (typeof window === 'undefined' || !('serviceWorker' in navigator)) {
    return;
  }

  // Disable Service Worker in development to avoid conflicts
  if (process.env.NODE_ENV === 'development') {
    console.log('Service Worker disabled in development mode');
    return;
  }

  try {
    // First, unregister any existing service workers to prevent conflicts
    const existingRegistrations = await navigator.serviceWorker.getRegistrations();
    for (const registration of existingRegistrations) {
      if (registration.scope !== window.location.origin + '/') {
        await registration.unregister();
      }
    }

    const registration = await navigator.serviceWorker.register('/sw.js', {
      scope: '/',
      updateViaCache: 'none' // Always check for updates
    });

    console.log('Service Worker registered:', registration);

    // Handle updates
    registration.addEventListener('updatefound', () => {
      const newWorker = registration.installing;
      if (newWorker) {
        newWorker.addEventListener('statechange', () => {
          if (newWorker.state === 'installed' && navigator.serviceWorker.controller) {
            // New content available, could show update notification
            console.log('New Service Worker available');
          }
        });
      }
    });

    // Handle Service Worker errors
    registration.addEventListener('error', (error) => {
      console.warn('Service Worker error:', error);
    });

    return registration;
  } catch (error) {
    console.error('Service Worker registration failed:', error);
  }
}

// Warm tile cache for current map view
export function warmTileCache(center: [number, number], zoom: number, theme = 'dark') {
  if (!navigator.serviceWorker?.controller || process.env.NODE_ENV === 'development') {
    return;
  }

  // Calculate tile bounds for current view + buffer
  const tileUrls: string[] = [];
  const buffer = 2; // Preload surrounding tiles
  
  // Convert coordinates to tile numbers
  const tileZ = Math.floor(zoom);
  const tileX = Math.floor((center[0] + 180) / 360 * Math.pow(2, tileZ));
  const tileY = Math.floor((1 - Math.log(Math.tan(center[1] * Math.PI / 180) + 1 / Math.cos(center[1] * Math.PI / 180)) / Math.PI) / 2 * Math.pow(2, tileZ));

  // Generate URLs for tiles in view + buffer
  for (let x = tileX - buffer; x <= tileX + buffer; x++) {
    for (let y = tileY - buffer; y <= tileY + buffer; y++) {
      if (x >= 0 && y >= 0 && x < Math.pow(2, tileZ) && y < Math.pow(2, tileZ)) {
        const basePath = window.location.pathname.includes('dashboard') ? '/dashboard' : '';
        tileUrls.push(`${basePath}/api/tiles/${theme}/${tileZ}/${x}/${y}`);
      }
    }
  }

  // Send cache warming request to Service Worker
  navigator.serviceWorker.controller.postMessage({
    type: 'WARM_TILE_CACHE',
    tiles: tileUrls,
  });

  console.log(`Warming cache for ${tileUrls.length} tiles`);
}

// Unregister Service Worker (useful for development)
export async function unregisterServiceWorker() {
  if (typeof window === 'undefined' || !('serviceWorker' in navigator)) {
    return;
  }

  try {
    const registrations = await navigator.serviceWorker.getRegistrations();
    for (const registration of registrations) {
      await registration.unregister();
      console.log('Service Worker unregistered');
    }
  } catch (error) {
    console.error('Service Worker unregistration failed:', error);
  }
}