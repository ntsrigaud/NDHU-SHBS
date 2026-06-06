export default function AuthLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-muted/40 px-4">
      <div className="mb-6 text-center">
        <p className="text-xs font-medium tracking-widest text-muted-foreground uppercase">
          國立東華大學
        </p>
        <p className="text-lg font-bold text-primary">NDHU 二手書市集</p>
      </div>
      {children}
    </div>
  );
}
