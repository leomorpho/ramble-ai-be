# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Structure

This is a full-stack application with two main components:
- **pb/**: PocketBase backend server (Go) - handles API, database, auth, file storage
- **sk/**: SvelteKit frontend application (TypeScript/Svelte) - builds to static files only

**CRITICAL**: The SvelteKit app is configured as a fully static application with **ABSOLUTELY NO Node.js backend**. All backend functionality is handled by PocketBase. The SvelteKit build produces static HTML/CSS/JS files that are served by PocketBase.

**⚠️ IMPORTANT CONSTRAINTS:**
- **NO SERVER-SIDE CODE**: No server hooks, no server load functions, no API routes in SvelteKit
- **CLIENT-SIDE ONLY**: All authentication, routing, and logic must be client-side
- **STATIC BUILD**: Must work with `@sveltejs/adapter-static` and static file serving
- **PocketBase ONLY**: Any backend endpoints or auth logic must be added to the Go PocketBase code in `pb/` directory

## Development Commands

### Frontend (sk/ directory)
```bash
cd sk
npm run dev          # Start development server
npm run build        # Build for production
npm run preview      # Preview production build
npm run check        # Type checking with svelte-check
npm run lint         # Lint and format check (Prettier + ESLint)
npm run format       # Format code with Prettier
npm run test         # Run all tests (unit + e2e)
npm run test:unit    # Run Vitest unit tests
npm run test:e2e     # Run Playwright e2e tests
npm run storybook    # Start Storybook development server
```

### Backend (pb/ directory)
```bash
cd pb
go build            # Build the PocketBase binary
./pocketbase serve --dev --http 0.0.0.0:8090
```

For development with auto-reload, use `modd` (if installed):
```bash
cd pb
modd               # Watch Go files and auto-restart server
```

### PocketBase Database Management

**DEFAULT APPROACH**: Use PocketBase Admin UI for database schema management and security rules:

1. **Access Admin UI**: Navigate to `http://localhost:8090/_/` when PocketBase is running
2. **Create Collections**: Use the visual interface to create and modify collections
3. **Configure Fields**: Add fields with appropriate types and validation rules
4. **Set Security Rules**: Configure API rules for list/view/create/update/delete operations
   - Use rule expressions like `@request.auth.id != ""`
   - Set field-level permissions for sensitive data
   - Test rules thoroughly before deployment

**Benefits of Admin UI approach**:
- Visual interface for schema design
- Real-time validation of security rules
- Easy backup/restore of schema via Settings > Export/Import
- No migration conflicts or versioning issues
- Immediate testing of API rules

**ALTERNATIVE**: Code-based migrations (only if explicitly requested by developer):
```bash
cd pb
# Create migration file using PocketBase command
./pocketbase migrate create migration_name

# Then edit the generated file with your migration logic
# Apply migrations
./pocketbase migrate up

# Sync migration history if needed
./pocketbase migrate history-sync
```

**Note**: Code-based migrations require careful management and can cause tracking issues if not handled properly. Use only when version control of schema changes is critical.

## Architecture Overview

### Backend (PocketBase)
- Simple Go application using PocketBase framework
- Serves static files from `../sk/build` directory
- Runs on port 8090 in development
- Uses `modd.conf` for development auto-reload

### Frontend (SvelteKit)
- Fully static SvelteKit application (NO Node.js server-side rendering)
- **Svelte 5 with runes** - Use modern reactive syntax ($state, $derived, $effect, etc.)
- Uses TypeScript and Tailwind CSS
- Configured with `@sveltejs/adapter-static` for static file generation
- All API calls go directly to PocketBase backend
- Internationalization with Paraglide (English and French)
- Testing setup with Vitest (unit) and Playwright (e2e)
- Storybook for component development
- Uses pnpm as package manager

### Key Frontend Features
- **Internationalization**: Paraglide middleware handles locale routing and message interpolation
- **Component Library**: Uses shadcn-svelte components from https://www.shadcn-svelte.com/docs/components
- **Testing**: Comprehensive testing with unit tests (Vitest) and e2e tests (Playwright)
- **Styling**: Tailwind CSS with dark/light theme support and typography plugin
- **UI Components**: Prefer shadcn-svelte components for consistent design system

### File Structure Notes
- Messages for i18n are in `sk/messages/{locale}.json`
- Generated Paraglide files are in `sk/src/lib/paraglide/`
- Storybook stories are in `sk/src/stories/`
- E2E tests are in `sk/e2e/`

## Development Workflow

1. Start backend: `cd pb && modd` (or manually build and run)
2. Start frontend: `cd sk && npm run dev`
3. Frontend dev server proxies to backend on port 8090
4. Build frontend for production before running backend serve command to serve static files

## Testing

Always run the full test suite before commits:
```bash
cd sk
npm run test        # Runs both unit and e2e tests
```

For development, run tests individually:
```bash
npm run test:unit   # Faster feedback during development
npm run test:e2e    # Integration testing
```

### Test Automation
A git pre-commit hook is configured to automatically run all tests before allowing commits. This ensures:
- Unit tests (23 tests): LoginForm, SignupForm, and other component tests
- E2E tests (1 test): Homepage functionality validation
- Commits are blocked if any tests fail
- Code quality is maintained automatically

## UI Development Guidelines

### Component Library
- Use shadcn-svelte components from https://www.shadcn-svelte.com/docs/components
- Install components via the CLI: `npx shadcn-svelte@latest add <component-name>`
- Components are copied into your project and fully customizable

### Theme Support
- Implement dark/light theme switching using Tailwind CSS
- Use CSS custom properties for theme-aware colors
- Follow shadcn-svelte theming conventions for consistency
- Test components in both light and dark modes

### Styling Best Practices
- Use Tailwind utility classes for styling
- Leverage shadcn-svelte's design tokens and color system
- Maintain consistent spacing and typography scales
- Use the typography plugin for rich text content

## Svelte 5 Development Guidelines

### Runes Usage
- **ALWAYS use Svelte 5 runes** - No legacy reactive syntax
- Use `$state()` for reactive variables instead of `let` variables
- Use `$derived()` for computed values instead of `$:`
- Use `$effect()` for side effects instead of `$:`
- Use `$props()` for component props instead of `export let`
- Use `{@render children()}` for slot content

### Component Patterns
```svelte
<script lang="ts">
  // Props
  let { title, count = 0 }: { title: string; count?: number } = $props();
  
  // State
  let localCount = $state(0);
  
  // Derived values
  let doubled = $derived(localCount * 2);
  
  // Effects
  $effect(() => {
    console.log('Count changed:', localCount);
  });
</script>
```

### Migration Notes
- Convert all `export let` to `$props()`
- Convert reactive statements `$:` to `$derived()` or `$effect()`
- Update event handlers to use modern syntax
- Use `{@render children()}` instead of `<slot>`

## Stripe Integration

This application includes full Stripe payment processing capabilities for subscription management.

### Backend Stripe Configuration

**Environment Variables** (stored in `pb/.env`):
```bash
STRIPE_SECRET_KEY=sk_test_your_stripe_secret_key_here
STRIPE_SECRET_WHSEC=whsec_your_webhook_signing_secret_here
STRIPE_CANCEL_URL=http://localhost:5174/pricing?canceled=true
STRIPE_SUCCESS_URL=http://localhost:5174/billing?success=true
HOST=http://localhost:8090
DEVELOPMENT=true
```

**⚠️ SECURITY**: Never commit `.env` file to git. Use `.env.example` as template.

### Stripe Collections

The following PocketBase collections are automatically managed via webhooks:

- **products**: Stripe product catalog
- **prices**: Pricing tiers and subscription plans  
- **customers**: Links PocketBase users to Stripe customers
- **subscriptions**: User subscription status and billing info

### API Endpoints

- `POST /create-checkout-session` - Create Stripe checkout for subscriptions/payments
- `POST /create-portal-link` - Generate billing portal access for customers
- `POST /stripe` - Webhook endpoint for Stripe event processing

### Setup Instructions

1. **Create Stripe Account**: Set up test mode at https://dashboard.stripe.com
2. **Get API Keys**: Copy secret key from https://dashboard.stripe.com/test/apikeys
3. **Configure Webhook**: 
   - URL: `https://your-domain.com/stripe`
   - Events: Select all events
   - Copy signing secret
4. **Update Environment**: Replace placeholder values in `pb/.env`
5. **Import Schema**: Use `pb/pb_bootstrap/pb_schema.json` to create collections
6. **Create Products**: Add products and pricing in Stripe dashboard

### Development Workflow

1. Start PocketBase: `cd pb && go run main.go serve`
2. Test webhooks: `stripe listen --forward-to=127.0.0.1:8090/stripe`
3. Create test products in Stripe dashboard
4. Products/prices automatically sync to PocketBase via webhooks

## Email Verification & Testing

The application includes full email verification support with Mailpit for development testing.

### Email Features

- **User Registration**: Email verification required for new accounts
- **Email Changes**: Changing email in Personal Account requires verification of new address
- **Password Reset**: Email-based password reset functionality
- **Email Templates**: Customizable HTML email templates for all verification flows

### Development Email Testing

**Quick Start**:
```bash
# Start full development environment with email testing
make dev
# This automatically starts Mailpit at http://localhost:8025
```

**Manual Setup**:
```bash
# Start Mailpit email server
make mailpit-up

# Start backend and frontend
make dev-backend
make dev-frontend
```

**Mailpit Access**:
- Web UI: http://localhost:8025 (view all sent emails)
- SMTP: localhost:1025 (PocketBase sends emails here)

### Email Configuration

Environment variables in `pb/.env`:
```bash
# Email Testing (Mailpit)
SMTP_HOST=localhost
SMTP_PORT=1025
SMTP_USERNAME=
SMTP_PASSWORD=
SMTP_TLS=false
EMAIL_FROM=noreply@localhost
EMAIL_FROM_NAME=Pulse
```

### Email Verification Flow

1. **Registration**: Users receive verification email after signup
2. **Email Change**: When updating email in Personal Account:
   - Verification email sent to NEW address
   - Current email remains active until verification
   - User must click link in new email to complete change
3. **Password Reset**: Standard email-based password reset flow

### Testing Email Flows

1. Start development environment: `make dev`
2. Register new user or change email in dashboard
3. Check Mailpit at http://localhost:8025 for emails
4. Click verification links to test complete flow

### Production Email Setup

For production, replace Mailpit with real SMTP service:
```bash
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-app-password
SMTP_TLS=true
EMAIL_FROM=noreply@yourdomain.com
EMAIL_FROM_NAME=Your App Name
```