export default function AuthLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen">
      {/* Left panel — brand */}
      <div
        className="relative hidden flex-col items-center justify-center p-12 lg:flex lg:w-5/12"
        style={{
          background: 'linear-gradient(160deg, hsl(152 52% 18%) 0%, hsl(165 45% 28%) 100%)',
        }}
      >
        {/* dot pattern */}
        <div
          className="pointer-events-none absolute inset-0 opacity-10"
          style={{
            backgroundImage: 'radial-gradient(circle, white 1px, transparent 1px)',
            backgroundSize: '24px 24px',
          }}
        />
        <div className="relative text-center text-white">
          <div className="mb-6 inline-flex h-16 w-16 items-center justify-center rounded-2xl bg-white/15 text-3xl font-bold ring-1 ring-white/20">
            東
          </div>
          <h1 className="text-3xl font-extrabold tracking-tight">二手書市集</h1>
          <p className="mt-2 text-base font-medium text-white/70">National Dong Hwa University</p>
          <p className="mt-6 max-w-xs text-sm leading-relaxed text-white/60">
            Campus-exclusive marketplace for NDHU students to buy and sell used textbooks at fair
            prices.
          </p>
        </div>
      </div>

      {/* Right panel — form */}
      <div className="flex flex-1 flex-col items-center justify-center bg-muted/30 px-6 py-12">
        {/* Mobile brand header */}
        <div className="mb-8 text-center lg:hidden">
          <p className="text-xs font-medium uppercase tracking-widest text-muted-foreground">
            國立東華大學
          </p>
          <p className="text-xl font-bold text-primary">NDHU 二手書市集</p>
        </div>
        {children}
      </div>
    </div>
  );
}
