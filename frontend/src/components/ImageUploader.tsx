"use client";

import { useState, useRef, useCallback } from "react";
import type { ImageData } from "@/types";

const MAX_IMAGES = 5;
const MAX_SIZE_BYTES = 5 * 1024 * 1024; // 5MB
const ALLOWED_TYPES = ["image/jpeg", "image/png", "image/gif", "image/webp"];

interface ImageUploaderProps {
  images: ImageData[];
  onImagesChange: (images: ImageData[]) => void;
}

export default function ImageUploader({
  images,
  onImagesChange,
}: ImageUploaderProps) {
  const [error, setError] = useState("");
  const fileInputRef = useRef<HTMLInputElement>(null);

  const fileToBase64 = useCallback((file: File): Promise<string> => {
    return new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => {
        const result = reader.result as string;
        // "data:image/png;base64,..." から Base64 部分だけ抽出
        const base64 = result.split(",")[1];
        resolve(base64);
      };
      reader.onerror = () => reject(new Error("ファイルの読み込みに失敗しました"));
      reader.readAsDataURL(file);
    });
  }, []);

  const validateFile = useCallback(
    (file: File, currentCount: number): string | null => {
      if (currentCount >= MAX_IMAGES) {
        return `画像は最大${MAX_IMAGES}枚までです`;
      }
      if (file.size > MAX_SIZE_BYTES) {
        return `${file.name}: サイズが5MBを超えています`;
      }
      if (!ALLOWED_TYPES.includes(file.type)) {
        return `${file.name}: サポートされていない形式です (JPEG, PNG, GIF, WebP のみ)`;
      }
      return null;
    },
    []
  );

  const isDuplicate = useCallback(
    (file: File): boolean => {
      return images.some(
        (img) => img.name === file.name && img.data.length === Math.ceil((file.size * 4) / 3)
      );
    },
    [images]
  );

  const processFiles = useCallback(
    async (files: FileList | File[]) => {
      setError("");

      const fileArray = Array.from(files);
      const newImages: ImageData[] = [];
      let currentCount = images.length;

      for (const file of fileArray) {
        // 重複チェック
        if (isDuplicate(file)) {
          continue;
        }

        // バリデーション
        const validationError = validateFile(file, currentCount);
        if (validationError) {
          setError(validationError);
          break;
        }

        try {
          const base64 = await fileToBase64(file);
          newImages.push({
            name: file.name,
            data: base64,
            mimeType: file.type,
          });
          currentCount++;
        } catch {
          setError(`${file.name}: 読み込みに失敗しました`);
          break;
        }
      }

      if (newImages.length > 0) {
        onImagesChange([...images, ...newImages]);
      }
    },
    [images, onImagesChange, fileToBase64, validateFile, isDuplicate]
  );

  const handleFileSelect = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const files = e.target.files;
      if (files && files.length > 0) {
        processFiles(files);
      }
      // inputをリセットして同じファイルを再選択可能にする
      e.target.value = "";
    },
    [processFiles]
  );

  const handleRemove = useCallback(
    (index: number) => {
      setError("");
      const newImages = images.filter((_, i) => i !== index);
      onImagesChange(newImages);
    },
    [images, onImagesChange]
  );

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
  }, []);

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      e.stopPropagation();

      const files = e.dataTransfer.files;
      if (files && files.length > 0) {
        processFiles(files);
      }
    },
    [processFiles]
  );

  const handleZoneClick = useCallback(() => {
    fileInputRef.current?.click();
  }, []);

  return (
    <div className="mb-4">
      <label className="block mb-2 font-semibold text-gray-800">
        Images (optional)
      </label>

      {/* ドロップゾーン */}
      <div
        onClick={handleZoneClick}
        onDragOver={handleDragOver}
        onDrop={handleDrop}
        className="border-2 border-dashed border-gray-300 rounded-lg p-4 text-center cursor-pointer hover:border-blue-400 hover:bg-blue-50 transition-colors"
      >
        <input
          ref={fileInputRef}
          type="file"
          accept="image/jpeg,image/png,image/gif,image/webp"
          multiple
          onChange={handleFileSelect}
          className="hidden"
        />
        <p className="text-gray-500 text-sm">
          Click or drag images here ({images.length}/{MAX_IMAGES})
        </p>
        <p className="text-gray-400 text-xs mt-1">
          JPEG, PNG, GIF, WebP / Max 5MB each
        </p>
      </div>

      {/* エラー表示 */}
      {error && (
        <p className="text-red-500 text-sm mt-2">{error}</p>
      )}

      {/* プレビュー */}
      {images.length > 0 && (
        <div className="grid grid-cols-5 gap-2 mt-3">
          {images.map((img, index) => (
            <div key={`${img.name}-${index}`} className="relative group">
              {/* Base64 data URL のプレビュー用。Next.js Image は data URL をサポートしないため img を使用 */}
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img
                src={`data:${img.mimeType};base64,${img.data}`}
                alt={img.name}
                className="w-full h-16 object-cover rounded border border-gray-200"
              />
              <button
                type="button"
                onClick={() => handleRemove(index)}
                className="absolute -top-1 -right-1 w-5 h-5 bg-red-500 text-white rounded-full text-xs flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity hover:bg-red-600"
                title="Remove"
              >
                x
              </button>
              <p className="text-xs text-gray-500 truncate mt-1" title={img.name}>
                {img.name}
              </p>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
