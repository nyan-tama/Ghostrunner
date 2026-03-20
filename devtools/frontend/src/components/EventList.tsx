"use client";

import { useEffect, useRef } from "react";
import type { DisplayEvent } from "@/types";
import EventItem from "./EventItem";

interface EventListProps {
  events: DisplayEvent[];
}

export default function EventList({ events }: EventListProps) {
  const listRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (listRef.current) {
      listRef.current.scrollTop = listRef.current.scrollHeight;
    }
  }, [events]);

  return (
    <div ref={listRef} className="p-0 max-h-[600px] overflow-y-auto">
      {events.map((event) => (
        <EventItem key={event.id} event={event} />
      ))}
    </div>
  );
}
