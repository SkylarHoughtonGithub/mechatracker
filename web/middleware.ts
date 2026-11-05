import { NextRequest, NextResponse } from 'next/server';

// Checks for the mt_session cookie's presence — not its validity. That's
// the backend's job; a stale cookie will 401 the API call and the
// AuthProvider will route the user back to /sign-in from the response.
//
// Public routes are the sign-in page, invite accept, and anything under
// /api/backend/* (which the Next.js rewrite forwards to the API and must
// not be redirected here). Static assets + Next.js internals are excluded
// via the matcher below.

const SESSION_COOKIE = 'mt_session';

const PUBLIC_PATH_PREFIXES = [
  '/sign-in',
  '/accept-invite',
  '/api/backend',
];

function isPublic(pathname: string): boolean {
  return PUBLIC_PATH_PREFIXES.some((p) => pathname === p || pathname.startsWith(`${p}/`));
}

export function middleware(req: NextRequest) {
  const { pathname } = req.nextUrl;
  if (isPublic(pathname)) {
    return NextResponse.next();
  }

  const session = req.cookies.get(SESSION_COOKIE);
  if (!session?.value) {
    const url = req.nextUrl.clone();
    url.pathname = '/sign-in';
    url.searchParams.set('next', pathname);
    return NextResponse.redirect(url);
  }

  return NextResponse.next();
}

export const config = {
  // Exclude Next.js internals (_next/*) and files with extensions (static
  // assets, images, fonts, manifests). Everything else runs through the
  // middleware above, which short-circuits on PUBLIC_PATH_PREFIXES.
  matcher: ['/((?!_next|[^?]*\\.(?:html?|css|js(?!on)|jpe?g|webp|png|gif|svg|ttf|woff2?|ico|csv|docx?|xlsx?|zip|webmanifest)).*)'],
};
