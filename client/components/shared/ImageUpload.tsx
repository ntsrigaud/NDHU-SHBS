'use client';

import { useCallback, useState } from 'react';
import Image from 'next/image';
import { useDropzone } from 'react-dropzone';
import { toast } from 'sonner';
import { Condition } from '@/lib/types';
import { cn } from '@/lib/utils';
import { X, Upload, Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';

const AI_URL = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080';
const MAX_FILES = 6;

interface UploadedImage {
  id: string;
  preview: string;
  cdn_url: string;
}

interface AiResult {
  condition: Condition;
  confidence: number;
}

interface Props {
  onChange: (imageIds: string[]) => void;
  onAiResult?: (result: AiResult) => void;
}

export default function ImageUpload({ onChange, onAiResult }: Props) {
  const [images, setImages] = useState<UploadedImage[]>([]);
  const [analyzing, setAnalyzing] = useState(false);

  const uploadFile = useCallback(
    async (file: File): Promise<UploadedImage | null> => {
      const formData = new FormData();
      formData.append('file', file);

      try {
        const res = await fetch(`${AI_URL}/api/v1/images/upload`, {
          method: 'POST',
          credentials: 'include',
          body: formData,
        });
        if (!res.ok) throw new Error('Upload failed');
        return res.json() as Promise<UploadedImage>;
      } catch {
        toast.error(`Failed to upload ${file.name}`);
        return null;
      }
    },
    [],
  );

  const analyzeCondition = useCallback(
    async (imageId: string) => {
      if (!onAiResult) return;
      setAnalyzing(true);
      try {
        const res = await fetch(`${AI_URL}/ai/analyze/condition`, {
          method: 'POST',
          credentials: 'include',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ image_id: imageId }),
        });
        if (!res.ok) return;
        const result: AiResult = await res.json();
        onAiResult(result);
        toast.info(
          `AI detected: ${result.condition} condition (${Math.round(result.confidence * 100)}% confidence)`,
        );
      } catch {
        // AI is best-effort — don't block the seller
      } finally {
        setAnalyzing(false);
      }
    },
    [onAiResult],
  );

  const onDrop = useCallback(
    async (acceptedFiles: File[]) => {
      const remaining = MAX_FILES - images.length;
      const toUpload = acceptedFiles.slice(0, remaining);

      const results = await Promise.all(toUpload.map(uploadFile));
      const uploaded = results.filter((r): r is UploadedImage => r !== null);

      const updated = [...images, ...uploaded];
      setImages(updated);
      onChange(updated.map((u) => u.id));

      // Run AI on the first image uploaded if this is the first batch
      if (images.length === 0 && uploaded.length > 0) {
        await analyzeCondition(uploaded[0].id);
      }
    },
    [images, uploadFile, analyzeCondition, onChange],
  );

  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop,
    accept: { 'image/jpeg': [], 'image/png': [], 'image/webp': [] },
    maxSize: 5 * 1024 * 1024,
    disabled: images.length >= MAX_FILES,
  });

  function remove(id: string) {
    const updated = images.filter((i) => i.id !== id);
    setImages(updated);
    onChange(updated.map((u) => u.id));
  }

  return (
    <div className="space-y-3">
      <div
        {...getRootProps()}
        className={cn(
          'flex cursor-pointer flex-col items-center justify-center rounded-lg border-2 border-dashed p-8 transition-colors',
          isDragActive ? 'border-primary bg-primary/5' : 'border-muted-foreground/25 hover:border-primary/50',
          images.length >= MAX_FILES && 'cursor-not-allowed opacity-50',
        )}
      >
        <input {...getInputProps()} />
        {analyzing ? (
          <Loader2 className="mb-2 h-8 w-8 animate-spin text-muted-foreground" />
        ) : (
          <Upload className="mb-2 h-8 w-8 text-muted-foreground" />
        )}
        <p className="text-sm text-muted-foreground">
          {isDragActive
            ? 'Drop images here…'
            : analyzing
              ? 'Analysing condition…'
              : `Drag & drop or click to upload (${images.length}/${MAX_FILES})`}
        </p>
        <p className="mt-1 text-xs text-muted-foreground">JPEG, PNG, WebP · max 5 MB each</p>
      </div>

      {images.length > 0 && (
        <div className="flex flex-wrap gap-2">
          {images.map((img) => (
            <div key={img.id} className="group relative h-20 w-20">
              <Image
                src={img.cdn_url || img.preview}
                alt="Book image"
                fill
                className="rounded-md object-cover"
                sizes="80px"
              />
              <Button
                type="button"
                variant="destructive"
                size="icon"
                className="absolute -right-1.5 -top-1.5 hidden h-5 w-5 group-hover:flex"
                onClick={() => remove(img.id)}
              >
                <X className="h-3 w-3" />
              </Button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
