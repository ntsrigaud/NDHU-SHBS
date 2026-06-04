'use client';

import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { BookListing, Condition } from '@/lib/types';
import { CreateListingPayload } from '@/hooks/useListings';

const listingSchema = z.object({
  title: z.string().min(1, 'Title is required'),
  author: z.string().min(1, 'Author is required'),
  price: z.coerce.number().min(0, 'Price must be 0 or more'),
  condition: z.enum(['good', 'moderate', 'poor']),
  isbn: z.string().optional(),
  course_code: z.string().optional(),
  department: z.string().optional(),
  description: z.string().optional(),
});

export type ListingFormValues = z.infer<typeof listingSchema>;

interface Props {
  defaultValues?: Partial<BookListing>;
  onSubmit: (values: CreateListingPayload) => void;
  isPending: boolean;
  submitLabel?: string;
  aiCondition?: { condition: Condition; confidence: number } | null;
}

export default function ListingForm({
  defaultValues,
  onSubmit,
  isPending,
  submitLabel = 'Save listing',
  aiCondition,
}: Props) {
  const form = useForm<ListingFormValues>({
    resolver: zodResolver(listingSchema),
    defaultValues: {
      title: defaultValues?.title ?? '',
      author: defaultValues?.author ?? '',
      price: defaultValues?.price ?? 0,
      condition: defaultValues?.condition ?? 'good',
      isbn: defaultValues?.isbn ?? '',
      course_code: defaultValues?.course_code ?? '',
      department: defaultValues?.department ?? '',
      description: defaultValues?.description ?? '',
    },
  });

  function handleSubmit(values: ListingFormValues) {
    onSubmit({
      ...values,
      isbn: values.isbn || undefined,
      course_code: values.course_code || undefined,
      department: values.department || undefined,
      description: values.description || undefined,
    });
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(handleSubmit)} className="space-y-4">
        <div className="grid gap-4 sm:grid-cols-2">
          <FormField
            control={form.control}
            name="title"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Title *</FormLabel>
                <FormControl>
                  <Input placeholder="Introduction to Algorithms" {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name="author"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Author *</FormLabel>
                <FormControl>
                  <Input placeholder="Thomas H. Cormen" {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
        </div>

        <div className="grid gap-4 sm:grid-cols-2">
          <FormField
            control={form.control}
            name="price"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Price (NT$) *</FormLabel>
                <FormControl>
                  <Input type="number" min={0} placeholder="200" {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name="condition"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Condition *</FormLabel>
                {aiCondition && (
                  <p className="text-xs text-muted-foreground">
                    AI suggests:{' '}
                    <span className="font-medium capitalize">{aiCondition.condition}</span>
                    {' '}({Math.round(aiCondition.confidence * 100)}% confidence)
                  </p>
                )}
                <Select onValueChange={field.onChange} defaultValue={field.value}>
                  <FormControl>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                  </FormControl>
                  <SelectContent>
                    <SelectItem value="good">Good</SelectItem>
                    <SelectItem value="moderate">Moderate</SelectItem>
                    <SelectItem value="poor">Poor</SelectItem>
                  </SelectContent>
                </Select>
                <FormMessage />
              </FormItem>
            )}
          />
        </div>

        <div className="grid gap-4 sm:grid-cols-3">
          <FormField
            control={form.control}
            name="isbn"
            render={({ field }) => (
              <FormItem>
                <FormLabel>ISBN</FormLabel>
                <FormControl>
                  <Input placeholder="978-..." {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name="department"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Department</FormLabel>
                <FormControl>
                  <Input placeholder="Computer Science" {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name="course_code"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Course code</FormLabel>
                <FormControl>
                  <Input placeholder="CS101" {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
        </div>

        <FormField
          control={form.control}
          name="description"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Description</FormLabel>
              <FormControl>
                <textarea
                  className="flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                  placeholder="Condition notes, highlights, missing pages…"
                  {...field}
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />

        <Button type="submit" disabled={isPending}>
          {isPending ? 'Saving…' : submitLabel}
        </Button>
      </form>
    </Form>
  );
}
