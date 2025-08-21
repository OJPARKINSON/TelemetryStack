import { NextRequest, NextResponse } from 'next/server';

const TILE_SOURCES = {
  dark: 'https://cartodb-basemaps-a.global.ssl.fastly.net/dark_all',
  light: 'https://cartodb-basemaps-a.global.ssl.fastly.net/light_all',
  satellite: 'https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile',
};

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ params: string[] }> }
) {
  try {
    const resolvedParams = await params;
    const [theme, z, x, y] = resolvedParams.params;
    
    // Validate parameters
    if (!theme || !z || !x || !y) {
      return NextResponse.json({ error: 'Invalid tile parameters' }, { status: 400 });
    }

    // Validate theme
    if (!(theme in TILE_SOURCES)) {
      return NextResponse.json({ error: 'Invalid theme' }, { status: 400 });
    }

    // Validate numeric parameters
    const zNum = parseInt(z);
    const xNum = parseInt(x);
    const yNum = parseInt(y);

    if (isNaN(zNum) || isNaN(xNum) || isNaN(yNum)) {
      return NextResponse.json({ error: 'Invalid tile coordinates' }, { status: 400 });
    }

    // Construct tile URL
    const baseUrl = TILE_SOURCES[theme as keyof typeof TILE_SOURCES];
    const tileUrl = theme === 'satellite' 
      ? `${baseUrl}/${z}/${y}/${x}` 
      : `${baseUrl}/${z}/${x}/${y}.png`;

    // Fetch the tile
    const response = await fetch(tileUrl, {
      headers: {
        'User-Agent': 'IRacing-Display/1.0',
        'Referer': request.headers.get('referer') || '',
      },
    });

    if (!response.ok) {
      return NextResponse.json({ error: 'Failed to fetch tile' }, { status: response.status });
    }

    const tileData = await response.arrayBuffer();

    // Return the tile with appropriate headers
    return new NextResponse(tileData, {
      status: 200,
      headers: {
        'Content-Type': response.headers.get('content-type') || 'image/png',
        'Cache-Control': 'public, max-age=86400, immutable',
        'Cross-Origin-Resource-Policy': 'cross-origin',
        'Access-Control-Allow-Origin': '*',
      },
    });

  } catch (error) {
    console.error('Tile proxy error:', error);
    return NextResponse.json({ error: 'Internal server error' }, { status: 500 });
  }
}

export async function OPTIONS() {
  return new NextResponse(null, {
    status: 200,
    headers: {
      'Access-Control-Allow-Origin': '*',
      'Access-Control-Allow-Methods': 'GET, OPTIONS',
      'Access-Control-Allow-Headers': 'Content-Type',
    },
  });
}