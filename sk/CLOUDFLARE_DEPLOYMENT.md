# Cloudflare Pages Deployment Guide

## Prerequisites

1. **Deploy PocketBase Backend First**: Deploy your PocketBase backend using Kamal and get the URL (e.g., `https://your-domain.com`)

## Cloudflare Pages Setup

### 1. Connect to GitHub Repository

1. Go to [Cloudflare Dashboard](https://dash.cloudflare.com) > Pages
2. Click "Create a project" > "Connect to Git"
3. Select your repository and the `sk/` directory as the source

### 2. Build Configuration

Set these build settings in Cloudflare Pages:

- **Framework preset**: SvelteKit
- **Build command**: `npm run build`
- **Build output directory**: `build`
- **Root directory**: `sk`

### 3. Environment Variables

In Cloudflare Pages dashboard, go to Settings > Environment Variables and add:

```
VITE_POCKETBASE_URL=https://your-pocketbase-domain.com
```

**Important**: Replace `https://your-pocketbase-domain.com` with your actual PocketBase backend URL.

### 4. Custom Domain (Optional)

If you want to use a custom domain instead of `*.pages.dev`:

1. Go to Custom domains in Cloudflare Pages
2. Add your domain
3. Update the `FRONTEND_URL` environment variable in your PocketBase deployment

## Development vs Production URLs

### Development
- Frontend: `http://localhost:5173` (Vite dev server)
- Backend: `http://localhost:8090` (PocketBase)

### Production
- Frontend: `https://your-app.pages.dev` (or custom domain)
- Backend: `https://your-pocketbase-domain.com` (Kamal deployed)

## CORS Configuration

The PocketBase backend is already configured to accept requests from:
- `localhost:*` (development)
- `*.pages.dev` (Cloudflare Pages)
- Custom domains set via `FRONTEND_URL` environment variable

## Testing the Setup

1. Deploy PocketBase backend first with Kamal
2. Deploy frontend to Cloudflare Pages with correct `VITE_POCKETBASE_URL`
3. Test authentication and API calls from the frontend

## Troubleshooting

### CORS Issues
- Ensure `VITE_POCKETBASE_URL` points to the correct backend URL
- Check that your domain is allowed in the CORS configuration
- Use browser developer tools to inspect network requests

### Build Errors
- Verify Node.js version compatibility (Node 18+)
- Check that all dependencies are properly installed
- Review build logs in Cloudflare Pages dashboard