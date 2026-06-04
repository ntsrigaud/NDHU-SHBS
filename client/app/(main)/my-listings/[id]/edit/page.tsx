export default function EditListingPage({ params }: { params: { id: string } }) {
  return (
    <div className="container mx-auto px-4 py-8">
      <h1 className="text-3xl font-bold">Edit Listing</h1>
      <p className="mt-2 text-muted-foreground">Editing listing {params.id} — coming soon.</p>
    </div>
  );
}
