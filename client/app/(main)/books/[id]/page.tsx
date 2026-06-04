export default function BookDetailPage({ params }: { params: { id: string } }) {
  return (
    <div className="container mx-auto px-4 py-8">
      <p className="text-muted-foreground">Book detail for {params.id} — coming soon.</p>
    </div>
  );
}
