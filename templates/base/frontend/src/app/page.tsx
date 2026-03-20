"use client";

import { useEffect, useState } from "react";

export default function Home() {
  const [message, setMessage] = useState<string>("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const apiBase = process.env.NEXT_PUBLIC_API_URL ?? "";

    fetch(`${apiBase}/api/hello`)
      .then((res) => res.json())
      .then((data) => {
        setMessage(data.message);
        setLoading(false);
      })
      .catch(() => {
        setMessage("Failed to connect to backend");
        setLoading(false);
      });
  }, []);

  return (
    <main className="flex min-h-screen flex-col items-center justify-center p-8">
      <h1 className="mb-4 text-4xl font-bold">{{PROJECT_NAME}}</h1>
      {loading ? (
        <p className="text-lg text-gray-500">Loading...</p>
      ) : (
        <p className="text-lg">{message}</p>
      )}
    </main>
  );
}
