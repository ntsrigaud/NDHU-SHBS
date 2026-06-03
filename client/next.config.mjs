/** @type {import('next').NextConfig} */
const nextConfig = {
  // Required for Docker standalone deployment.
  output: 'standalone',
  images: {
    remotePatterns: [
      {
        protocol: 'https',
        // Allow images from any CloudFront subdomain.
        hostname: '*.cloudfront.net',
      },
    ],
  },
};

export default nextConfig;
