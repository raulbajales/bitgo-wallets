# BitGo Wallets Web App

This is a Next.js application built with TypeScript and Tailwind CSS.

## Getting Started

1. **Install Dependencies:**

   ```bash
   npm install
   # or
   yarn install
   # or
   pnpm install
   ```

2. **Run the Development Server:**

   ```bash
   npm run dev
   # or
   yarn dev
   # or
   pnpm dev
   ```

3. **Open Your Browser:**
   Navigate to [http://localhost:3000](http://localhost:3000) to see the landing page.

## Project Structure

```
webapp/
├── src/
│   └── app/
│       ├── layout.tsx       # Root layout component
│       ├── page.tsx         # Landing page
│       └── globals.css      # Global styles
├── package.json             # Dependencies and scripts
├── tsconfig.json           # TypeScript configuration
├── tailwind.config.js      # Tailwind CSS configuration
├── next.config.js          # Next.js configuration
└── postcss.config.js       # PostCSS configuration
```

## Features

- ⚡ **Next.js 15** with App Router
- 🎨 **Tailwind CSS** for styling
- 📝 **TypeScript** for type safety
- 🔍 **ESLint** for code quality
- 📱 **Responsive Design** mobile-first approach

## Available Scripts

- `npm run dev` - Start development server
- `npm run build` - Build for production
- `npm run start` - Start production server
- `npm run lint` - Run ESLint

## Landing Page Features

The landing page includes:

- Clean, modern design with gradient backgrounds
- Hero section with call-to-action buttons
- Features showcase with icons
- Responsive layout that works on all devices
- Professional styling using Tailwind CSS

## Customization

You can customize the landing page by editing `src/app/page.tsx`. The page uses Tailwind CSS classes for styling, making it easy to modify colors, spacing, and layout.
