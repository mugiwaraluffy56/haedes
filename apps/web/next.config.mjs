/** @type {import('next').NextConfig} */
const dashboardOrigin = process.env.HAEDES_DASHBOARD_ORIGIN ?? 'http://127.0.0.1:3001';

const nextConfig = {
  async rewrites() {
    return [
      {
        source: '/dashboard',
        destination: `${dashboardOrigin}/dashboard`,
      },
      {
        source: '/dashboard/:path*',
        destination: `${dashboardOrigin}/dashboard/:path*`,
      },
    ];
  },
};

export default nextConfig;
