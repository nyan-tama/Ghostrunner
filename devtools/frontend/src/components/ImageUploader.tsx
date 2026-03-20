"use client";

import { useState, useRef, useCallback } from "react";
import type { ImageData } from "@/types";

const MAX_IMAGES = 5;
const MAX_SIZE_BYTES = 5 * 1024 * 1024; // 5MB
const COMPRESSION_THRESHOLD_BYTES = 2 * 1024 * 1024; // 2MB - 圧縮を開始する閾値
const MAX_DIMENSION = 2048; // リサイズ時の最大長辺
const COMPRESSION_QUALITY = 0.8; // JPEG出力品質
const ALLOWED_TYPES = ["image/jpeg", "image/png", "image/gif", "image/webp"];
const COMPRESSIBLE_TYPES = ["image/jpeg", "image/png"];

interface ImageUploaderProps {
  images: ImageData[];
  onImagesChange: (images: ImageData[]) => void;
}

// CanvasのImageDataで透過があるかチェック
function hasTransparencyInCanvas(canvasImageData: globalThis.ImageData): boolean {
  const { data } = canvasImageData;
  // RGBA形式で4番目のチャンネル（alpha）をチェック
  for (let i = 3; i < data.length; i += 4) {
    if (data[i] < 255) {
      return true;
    }
  }
  return false;
}

// 圧縮結果の型定義
interface CompressionResult {
  base64: string;
  mimeType: string;
  name: string;
}

// Canvas APIで画像を圧縮する
async function compressImage(file: File): Promise<CompressionResult> {
  return new Promise((resolve, reject) => {
    const img = new Image();
    const url = URL.createObjectURL(file);

    img.onload = () => {
      URL.revokeObjectURL(url);

      // リサイズ寸法を計算
      let { width, height } = img;
      const maxDim = Math.max(width, height);

      if (maxDim > MAX_DIMENSION) {
        const scale = MAX_DIMENSION / maxDim;
        width = Math.round(width * scale);
        height = Math.round(height * scale);
      }

      // Canvasに描画
      const canvas = document.createElement("canvas");
      canvas.width = width;
      canvas.height = height;
      const ctx = canvas.getContext("2d");

      if (!ctx) {
        reject(new Error("Canvas context を取得できませんでした"));
        return;
      }

      ctx.drawImage(img, 0, 0, width, height);

      // PNG透過チェック
      let outputMimeType = "image/jpeg";
      let outputName = file.name;

      if (file.type === "image/png") {
        const canvasImageData = ctx.getImageData(0, 0, width, height);
        if (hasTransparencyInCanvas(canvasImageData)) {
          // 透過PNGはPNGのまま出力
          outputMimeType = "image/png";
        } else {
          // 透過なしPNGはJPEGに変換
          outputMimeType = "image/jpeg";
          outputName = file.name.replace(/\.png$/i, ".jpg");
        }
      }

      // BlobとしてCanvas出力
      const quality = outputMimeType === "image/jpeg" ? COMPRESSION_QUALITY : undefined;
      canvas.toBlob(
        (blob) => {
          if (!blob) {
            reject(new Error("画像の圧縮に失敗しました"));
            return;
          }

          // 圧縮後も5MBを超える場合はエラー
          if (blob.size > MAX_SIZE_BYTES) {
            reject(new Error(`圧縮後も5MBを超えています (${Math.round(blob.size / 1024 / 1024 * 10) / 10}MB)`));
            return;
          }

          // BlobをBase64に変換
          const reader = new FileReader();
          reader.onload = () => {
            const result = reader.result as string;
            const base64 = result.split(",")[1];
            resolve({
              base64,
              mimeType: outputMimeType,
              name: outputName,
            });
          };
          reader.onerror = () => reject(new Error("圧縮画像の読み込みに失敗しました"));
          reader.readAsDataURL(blob);
        },
        outputMimeType,
        quality
      );
    };

    img.onerror = () => {
      URL.revokeObjectURL(url);
      reject(new Error("画像の読み込みに失敗しました"));
    };

    img.src = url;
  });
}

// 圧縮が必要かどうかを判定
function shouldCompress(file: File): boolean {
  // GIF/WebPは圧縮しない
  if (!COMPRESSIBLE_TYPES.includes(file.type)) {
    return false;
  }
  // 2MB以下は圧縮しない
  if (file.size <= COMPRESSION_THRESHOLD_BYTES) {
    return false;
  }
  return true;
}

export default function ImageUploader({
  images,
  onImagesChange,
}: ImageUploaderProps) {
  const [error, setError] = useState("");
  const [isCompressing, setIsCompressing] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const cameraInputRef = useRef<HTMLInputElement>(null);

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
      if (!ALLOWED_TYPES.includes(file.type)) {
        return `${file.name}: サポートされていない形式です (JPEG, PNG, GIF, WebP のみ)`;
      }
      // 圧縮不可能なファイル（GIF/WebP）は5MBを超えていたらエラー
      if (!shouldCompress(file) && file.size > MAX_SIZE_BYTES) {
        return `${file.name}: サイズが5MBを超えています`;
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
      let hasCompressibleFile = false;

      // 圧縮が必要なファイルがあるかチェック
      for (const file of fileArray) {
        if (shouldCompress(file)) {
          hasCompressibleFile = true;
          break;
        }
      }

      if (hasCompressibleFile) {
        setIsCompressing(true);
      }

      try {
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
            if (shouldCompress(file)) {
              // 圧縮処理
              const result = await compressImage(file);
              newImages.push({
                name: result.name,
                data: result.base64,
                mimeType: result.mimeType,
              });
            } else {
              // 圧縮不要なファイルはそのまま変換
              const base64 = await fileToBase64(file);
              newImages.push({
                name: file.name,
                data: base64,
                mimeType: file.type,
              });
            }
            currentCount++;
          } catch (err) {
            const message = err instanceof Error ? err.message : "読み込みに失敗しました";
            setError(`${file.name}: ${message}`);
            break;
          }
        }

        if (newImages.length > 0) {
          onImagesChange([...images, ...newImages]);
        }
      } finally {
        setIsCompressing(false);
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

  const handleCameraClick = useCallback(() => {
    cameraInputRef.current?.click();
  }, []);

  const handleCameraCapture = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const files = e.target.files;
      if (files && files.length > 0) {
        processFiles(files);
      }
      e.target.value = "";
    },
    [processFiles]
  );

  return (
    <div className="mb-4">
      <label className="block mb-2 font-semibold text-gray-800">
        Images (optional)
      </label>

      {/* 圧縮中インジケーター */}
      {isCompressing && (
        <div className="flex items-center gap-2 mb-2 text-blue-600">
          <svg
            className="animate-spin h-4 w-4"
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
            viewBox="0 0 24 24"
          >
            <circle
              className="opacity-25"
              cx="12"
              cy="12"
              r="10"
              stroke="currentColor"
              strokeWidth="4"
            />
            <path
              className="opacity-75"
              fill="currentColor"
              d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
            />
          </svg>
          <span className="text-sm">圧縮中...</span>
        </div>
      )}

      {/* ドロップゾーン + カメラ撮影ゾーン */}
      <div className="grid grid-cols-2 gap-2">
        {/* ドロップゾーン（左側） */}
        <div
          onClick={isCompressing ? undefined : handleZoneClick}
          onDragOver={isCompressing ? undefined : handleDragOver}
          onDrop={isCompressing ? undefined : handleDrop}
          className={`border-2 border-dashed border-gray-300 rounded-lg p-4 text-center transition-colors ${
            isCompressing
              ? "opacity-50 cursor-not-allowed"
              : "cursor-pointer hover:border-blue-400 hover:bg-blue-50"
          }`}
        >
          <input
            ref={fileInputRef}
            type="file"
            accept="image/jpeg,image/png,image/gif,image/webp"
            multiple
            onChange={handleFileSelect}
            disabled={isCompressing}
            className="hidden"
          />
          <p className="text-gray-500 text-sm">
            Click or drag images here ({images.length}/{MAX_IMAGES})
          </p>
          <p className="text-gray-400 text-xs mt-1">
            JPEG, PNG, GIF, WebP / Max 5MB each
          </p>
        </div>

        {/* カメラ撮影ゾーン（右側） */}
        <div
          onClick={isCompressing ? undefined : handleCameraClick}
          className={`border-2 border-dashed border-gray-300 rounded-lg p-4 flex flex-col items-center justify-center transition-colors ${
            isCompressing
              ? "opacity-50 cursor-not-allowed"
              : "cursor-pointer hover:border-blue-400 hover:bg-blue-50"
          }`}
        >
          <input
            ref={cameraInputRef}
            type="file"
            accept="image/*"
            capture="environment"
            onChange={handleCameraCapture}
            disabled={isCompressing}
            className="hidden"
          />
          <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth={2}
            strokeLinecap="round"
            strokeLinejoin="round"
            className="w-6 h-6 text-gray-400"
          >
            <path d="M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z" />
            <circle cx="12" cy="13" r="4" />
          </svg>
          <p className="text-gray-500 text-sm mt-1">Take photo</p>
        </div>
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
