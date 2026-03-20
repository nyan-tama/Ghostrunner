"use client";

import { useEffect, useState, useCallback, useRef } from "react";

interface FileInfo {
  key: string;
  size: number;
  last_modified: string;
}

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "";

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export default function StoragePage() {
  const [files, setFiles] = useState<FileInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState("");
  const [dragOver, setDragOver] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const fetchFiles = useCallback(async () => {
    try {
      const res = await fetch(`${API_BASE}/api/storage/files`);
      if (!res.ok) throw new Error("Failed to fetch");
      const data = await res.json();
      setFiles(data);
      setError("");
    } catch {
      setError("Failed to connect to backend");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchFiles();
  }, [fetchFiles]);

  const uploadFile = async (file: File) => {
    setUploading(true);
    try {
      const formData = new FormData();
      formData.append("file", file);

      const res = await fetch(`${API_BASE}/api/storage/upload`, {
        method: "POST",
        body: formData,
      });
      if (!res.ok) throw new Error("Upload failed");
      await fetchFiles();
      setError("");
    } catch {
      setError("Failed to upload file");
    } finally {
      setUploading(false);
    }
  };

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) uploadFile(file);
    if (fileInputRef.current) fileInputRef.current.value = "";
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setDragOver(false);
    const file = e.dataTransfer.files?.[0];
    if (file) uploadFile(file);
  };

  const handleDelete = async (key: string) => {
    try {
      await fetch(`${API_BASE}/api/storage/files/${encodeURIComponent(key)}`, {
        method: "DELETE",
      });
      await fetchFiles();
    } catch {
      setError("Failed to delete file");
    }
  };

  return (
    <main className="mx-auto min-h-screen max-w-2xl p-8">
      <h1 className="mb-8 text-3xl font-bold">Storage</h1>

      <div
        onDragOver={(e) => {
          e.preventDefault();
          setDragOver(true);
        }}
        onDragLeave={() => setDragOver(false)}
        onDrop={handleDrop}
        className={`mb-8 flex flex-col items-center justify-center rounded-lg border-2 border-dashed p-8 transition ${
          dragOver
            ? "border-blue-500 bg-blue-50"
            : "border-gray-300 hover:border-gray-400"
        }`}
      >
        <p className="mb-4 text-gray-600">
          {uploading
            ? "Uploading..."
            : "Drag & drop a file here, or click to select"}
        </p>
        <input
          ref={fileInputRef}
          type="file"
          onChange={handleFileSelect}
          className="hidden"
        />
        <button
          onClick={() => fileInputRef.current?.click()}
          disabled={uploading}
          className="rounded bg-blue-600 px-4 py-2 text-white hover:bg-blue-700 disabled:opacity-50"
        >
          Select File
        </button>
      </div>

      {error && <p className="mb-4 text-red-600">{error}</p>}

      {loading ? (
        <p className="text-gray-500">Loading...</p>
      ) : files.length === 0 ? (
        <p className="text-gray-500">No files yet.</p>
      ) : (
        <ul className="space-y-3">
          {files.map((file) => (
            <li key={file.key} className="rounded-lg border p-4">
              <div className="flex items-center justify-between">
                <div className="min-w-0 flex-1">
                  <p className="truncate font-medium">{file.key}</p>
                  <p className="mt-1 text-sm text-gray-500">
                    {formatSize(file.size)} &middot; {file.last_modified}
                  </p>
                </div>
                <div className="ml-4 flex gap-2">
                  <a
                    href={`${API_BASE}/api/storage/files/${encodeURIComponent(file.key)}`}
                    className="text-sm text-blue-600 hover:underline"
                  >
                    Download
                  </a>
                  <button
                    onClick={() => handleDelete(file.key)}
                    className="text-sm text-red-600 hover:underline"
                  >
                    Delete
                  </button>
                </div>
              </div>
            </li>
          ))}
        </ul>
      )}
    </main>
  );
}
