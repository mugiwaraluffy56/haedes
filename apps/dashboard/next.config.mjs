/** @type {import('next').NextConfig} */
const nextConfig = {
  transpilePackages: ['@haedes/sdk', '@haedes/api-types'],
  webpack(config) {
    config.resolve.extensionAlias = {
      ...(config.resolve.extensionAlias ?? {}),
      '.js': ['.js', '.ts', '.tsx'],
    };
    return config;
  },
};

export default nextConfig;
