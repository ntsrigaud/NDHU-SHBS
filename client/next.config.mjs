/** @type {import('next').NextConfig} */
const nextConfig = {
  // 'standalone' output is required by the Docker runtime image.
  // Vercel manages its own output pipeline, so we omit it there.
  // Set NEXT_OUTPUT_STANDALONE=1 at build time (e.g. in the Dockerfile)
  // to enable this mode.
  ...(process.env.NEXT_OUTPUT_STANDALONE === '1' && { output: 'standalone' }),
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
