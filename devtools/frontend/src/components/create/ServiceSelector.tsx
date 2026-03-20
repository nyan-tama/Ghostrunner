"use client";

import type { DataService } from "@/types";

interface ServiceSelectorProps {
  selected: DataService[];
  onChange: (services: DataService[]) => void;
}

const SERVICES: readonly { id: DataService; label: string; description: string }[] = [
  { id: "database", label: "PostgreSQL + GORM", description: "Database with migration support" },
  { id: "storage", label: "Cloudflare R2 / MinIO", description: "Object storage for files" },
  { id: "cache", label: "Redis", description: "In-memory cache and session store" },
] as const;

export default function ServiceSelector({ selected, onChange }: ServiceSelectorProps) {
  const handleToggle = (serviceId: DataService) => {
    if (selected.includes(serviceId)) {
      onChange(selected.filter((s) => s !== serviceId));
    } else {
      onChange([...selected, serviceId]);
    }
  };

  return (
    <div>
      <label className="block text-sm font-medium text-gray-700 mb-2">
        Data Services (optional)
      </label>
      <div className="space-y-2">
        {SERVICES.map((service) => (
          <label
            key={service.id}
            className="flex items-start gap-3 p-3 rounded-lg border border-gray-200 hover:border-blue-300 cursor-pointer transition-colors"
          >
            <input
              type="checkbox"
              checked={selected.includes(service.id)}
              onChange={() => handleToggle(service.id)}
              className="mt-0.5 h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
            />
            <div>
              <div className="text-sm font-medium text-gray-800">{service.label}</div>
              <div className="text-xs text-gray-500">{service.description}</div>
            </div>
          </label>
        ))}
      </div>
    </div>
  );
}
