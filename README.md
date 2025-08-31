# PocketBase Starter Kit

A powerful, production-ready full-stack starter kit that combines SvelteKit and PocketBase for rapid SaaS development.

## ✨ Features

- **🚀 Full-Stack Solution**: SvelteKit frontend + PocketBase backend
- **🔐 Modern Authentication**: WebAuthn passkeys with progressive login flow
- **💳 Payment Integration**: Complete Stripe subscription management
- **📁 File Uploads**: TUS resumable uploads with image thumbnails
- **🌍 Internationalization**: Multi-language support (English/French)
- **🧪 Testing Suite**: Vitest unit tests + Playwright E2E tests
- **🎨 Modern UI**: shadcn-svelte components with dark/light themes
- **⚡ Performance**: Static site generation with optimal loading
- **🔒 Type Safety**: Full TypeScript coverage
- **🛠️ Developer Experience**: Pre-commit hooks, Storybook, linting

## 🚀 Quick Start

```bash
# Clone the repository
git clone <your-repo-url>
cd pb_starter

# Complete setup (dependencies + git hooks + build)
make setup

# Start development
make dev
```

The setup command will:
- Install all dependencies (frontend + backend)
- Set up git pre-commit hooks to run tests before commits
- Build the PocketBase backend
- Prepare everything for development

## 📖 Development

### Available Commands

```bash
make help          # Show all available commands
make dev           # Start both backend and frontend
make test          # Run all tests
make build         # Build for production
make clean         # Clean build artifacts
```

### Manual Setup (Alternative)

If you prefer manual setup:

```bash
# Install dependencies
cd sk && npm install
cd ../pb && go mod tidy

# Build backend
cd pb && go build

# Start development
make dev
```

## 🏗️ Architecture

- **Frontend**: SvelteKit 5 with runes, TypeScript, Tailwind CSS
- **Backend**: PocketBase (Go) with SQLite database
- **Authentication**: WebAuthn passkeys + traditional auth
- **Payments**: Stripe integration with webhooks
- **File Storage**: Built-in with TUS uploads
- **Testing**: Vitest + Playwright
- **Deployment**: Static files served by PocketBase

## 📱 Tech Stack

### Frontend
- SvelteKit 5 with runes
- TypeScript
- Tailwind CSS
- shadcn-svelte components
- Paraglide i18n
- GSAP animations

### Backend
- PocketBase (Go framework)
- SQLite database
- WebAuthn implementation
- Stripe webhooks
- TUS upload handler

### Developer Tools
- Vitest (unit testing)
- Playwright (E2E testing)
- Storybook (component development)
- ESLint + Prettier
- Pre-commit hooks

## 🔧 Configuration

Environment variables are stored in `pb/.env`:

```bash
STRIPE_SECRET_KEY=sk_test_...
STRIPE_SECRET_WHSEC=whsec_...
HOST=http://localhost:8090
DEVELOPMENT=true
```

### Payment Provider Integration

The application uses a provider-agnostic payment system. Currently, Stripe is the primary provider.

#### Stripe Configuration

**Webhook Endpoint:**
- `{HOST}/api/webhooks/stripe`

**Required Webhook Events:**
- `customer.subscription.created`
- `customer.subscription.updated` 
- `customer.subscription.deleted`
- `checkout.session.completed`
- `invoice.payment_failed`
- `invoice.payment_succeeded`
- `invoice.payment.paid`
- `invoice_payment.paid`

**Payment Endpoints:**
- Checkout: `POST /api/payment/checkout`
- Customer Portal: `POST /api/payment/portal`
- Plan Change: `POST /api/payment/change-plan`
- Switch to Free: `POST /api/subscription/switch-to-free`

**Redirect URLs:**
Dynamically constructed using `HOST + route paths`:
- Success URL: `{HOST}/pricing?success=true`  
- Cancel URL: `{HOST}/pricing?canceled=true`
- Portal Return URL: `{HOST}/pricing`

#### Adding New Payment Providers

To add support for additional providers (Paddle, Polar, etc.):

1. Implement the `PaymentProvider` interface in `internal/payment/`
2. Add provider-specific webhook endpoints to `main.go`
3. Update the database schema to support the new provider
4. Update this README with the new provider's configuration

**Note**: When webhook endpoints or events are modified, update this README section.

## 🧪 Testing

The project includes comprehensive testing with automatic pre-commit validation:

```bash
make test          # Run all tests
make test-unit     # Unit tests only
make test-e2e      # E2E tests only
```

Tests automatically run before git commits to ensure code quality.

## 📚 Documentation

Detailed project instructions are in `CLAUDE.md` for AI-assisted development.

## 🌐 Deployment

1. Build the frontend: `cd sk && npm run build`
2. Build the backend: `cd pb && go build`
3. Deploy the PocketBase binary with the `sk/build` directory
4. Configure environment variables for production

## 🔗 Key Endpoints

- **Frontend**: http://localhost:5174
- **Backend API**: http://localhost:8090
- **Admin Dashboard**: http://localhost:8090/_/
- **Storybook**: http://localhost:6006

## 📄 License

[Your License Here]