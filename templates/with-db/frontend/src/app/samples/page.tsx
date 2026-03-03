"use client";

import { useEffect, useState, useCallback } from "react";

interface Sample {
  id: number;
  name: string;
  description: string;
  created_at: string;
  updated_at: string;
}

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "";

export default function Home() {
  const [samples, setSamples] = useState<Sample[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [editingId, setEditingId] = useState<number | null>(null);

  const fetchSamples = useCallback(async () => {
    try {
      const res = await fetch(`${API_BASE}/api/samples`);
      if (!res.ok) throw new Error("Failed to fetch");
      const data = await res.json();
      setSamples(data);
      setError("");
    } catch {
      setError("Failed to connect to backend");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchSamples();
  }, [fetchSamples]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;

    try {
      if (editingId !== null) {
        await fetch(`${API_BASE}/api/samples/${editingId}`, {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ name, description }),
        });
      } else {
        await fetch(`${API_BASE}/api/samples`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ name, description }),
        });
      }
      setName("");
      setDescription("");
      setEditingId(null);
      await fetchSamples();
    } catch {
      setError("Failed to save");
    }
  };

  const handleEdit = (sample: Sample) => {
    setEditingId(sample.id);
    setName(sample.name);
    setDescription(sample.description);
  };

  const handleCancel = () => {
    setEditingId(null);
    setName("");
    setDescription("");
  };

  const handleDelete = async (id: number) => {
    try {
      await fetch(`${API_BASE}/api/samples/${id}`, { method: "DELETE" });
      await fetchSamples();
    } catch {
      setError("Failed to delete");
    }
  };

  return (
    <main className="mx-auto min-h-screen max-w-2xl p-8">
      <h1 className="mb-8 text-3xl font-bold">{{PROJECT_NAME}}</h1>

      <form onSubmit={handleSubmit} className="mb-8 rounded-lg border p-4">
        <h2 className="mb-4 text-lg font-semibold">
          {editingId !== null ? "Edit" : "New"}
        </h2>
        <div className="mb-3">
          <input
            type="text"
            placeholder="Name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="w-full rounded border px-3 py-2"
          />
        </div>
        <div className="mb-3">
          <textarea
            placeholder="Description"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            className="w-full rounded border px-3 py-2"
            rows={3}
          />
        </div>
        <div className="flex gap-2">
          <button
            type="submit"
            className="rounded bg-blue-600 px-4 py-2 text-white hover:bg-blue-700"
          >
            {editingId !== null ? "Update" : "Create"}
          </button>
          {editingId !== null && (
            <button
              type="button"
              onClick={handleCancel}
              className="rounded border px-4 py-2 hover:bg-gray-100"
            >
              Cancel
            </button>
          )}
        </div>
      </form>

      {error && <p className="mb-4 text-red-600">{error}</p>}

      {loading ? (
        <p className="text-gray-500">Loading...</p>
      ) : samples.length === 0 ? (
        <p className="text-gray-500">No items yet.</p>
      ) : (
        <ul className="space-y-3">
          {samples.map((sample) => (
            <li key={sample.id} className="rounded-lg border p-4">
              <div className="flex items-start justify-between">
                <div>
                  <h3 className="font-semibold">{sample.name}</h3>
                  {sample.description && (
                    <p className="mt-1 text-sm text-gray-600">
                      {sample.description}
                    </p>
                  )}
                </div>
                <div className="flex gap-2">
                  <button
                    onClick={() => handleEdit(sample)}
                    className="text-sm text-blue-600 hover:underline"
                  >
                    Edit
                  </button>
                  <button
                    onClick={() => handleDelete(sample.id)}
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
