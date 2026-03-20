"use client";

import { useEffect, useState, useCallback } from "react";

interface CacheEntry {
  key: string;
  value: string;
  ttl_seconds: number;
}

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "";

export default function CachePage() {
  const [entries, setEntries] = useState<CacheEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const [key, setKey] = useState("");
  const [value, setValue] = useState("");
  const [ttl, setTtl] = useState("");

  const fetchEntries = useCallback(async () => {
    try {
      const res = await fetch(`${API_BASE}/api/cache`);
      if (!res.ok) throw new Error("Failed to fetch");
      const data = await res.json();
      setEntries(data);
      setError("");
    } catch {
      setError("Failed to connect to backend");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchEntries();
  }, [fetchEntries]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!key.trim() || !value.trim()) return;

    try {
      const body: { key: string; value: string; ttl_seconds?: number } = {
        key: key.trim(),
        value: value.trim(),
      };
      if (ttl.trim()) {
        body.ttl_seconds = parseInt(ttl, 10);
      }

      const res = await fetch(`${API_BASE}/api/cache`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (!res.ok) throw new Error("Failed to set");

      setKey("");
      setValue("");
      setTtl("");
      await fetchEntries();
    } catch {
      setError("Failed to save");
    }
  };

  const handleDelete = async (targetKey: string) => {
    try {
      await fetch(`${API_BASE}/api/cache/${encodeURIComponent(targetKey)}`, {
        method: "DELETE",
      });
      await fetchEntries();
    } catch {
      setError("Failed to delete");
    }
  };

  const formatTtl = (seconds: number): string => {
    if (seconds < 0) return "no expiry";
    if (seconds < 60) return `${seconds}s`;
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
    return `${Math.floor(seconds / 3600)}h`;
  };

  return (
    <main className="mx-auto min-h-screen max-w-2xl p-8">
      <h1 className="mb-8 text-3xl font-bold">Cache</h1>

      <form onSubmit={handleSubmit} className="mb-8 rounded-lg border p-4">
        <h2 className="mb-4 text-lg font-semibold">Set Key</h2>
        <div className="mb-3">
          <input
            type="text"
            placeholder="Key"
            value={key}
            onChange={(e) => setKey(e.target.value)}
            className="w-full rounded border px-3 py-2"
          />
        </div>
        <div className="mb-3">
          <textarea
            placeholder="Value"
            value={value}
            onChange={(e) => setValue(e.target.value)}
            className="w-full rounded border px-3 py-2"
            rows={3}
          />
        </div>
        <div className="mb-3">
          <input
            type="number"
            placeholder="TTL (seconds, optional)"
            value={ttl}
            onChange={(e) => setTtl(e.target.value)}
            className="w-full rounded border px-3 py-2"
            min="0"
          />
        </div>
        <button
          type="submit"
          className="rounded bg-blue-600 px-4 py-2 text-white hover:bg-blue-700"
        >
          Set
        </button>
      </form>

      {error && <p className="mb-4 text-red-600">{error}</p>}

      {loading ? (
        <p className="text-gray-500">Loading...</p>
      ) : entries.length === 0 ? (
        <p className="text-gray-500">No keys stored.</p>
      ) : (
        <ul className="space-y-3">
          {entries.map((entry) => (
            <li key={entry.key} className="rounded-lg border p-4">
              <div className="flex items-start justify-between">
                <div className="min-w-0 flex-1">
                  <h3 className="font-semibold">{entry.key}</h3>
                  <p className="mt-1 break-all text-sm text-gray-600">
                    {entry.value}
                  </p>
                  <p className="mt-1 text-xs text-gray-400">
                    TTL: {formatTtl(entry.ttl_seconds)}
                  </p>
                </div>
                <button
                  onClick={() => handleDelete(entry.key)}
                  className="ml-4 text-sm text-red-600 hover:underline"
                >
                  Delete
                </button>
              </div>
            </li>
          ))}
        </ul>
      )}
    </main>
  );
}
