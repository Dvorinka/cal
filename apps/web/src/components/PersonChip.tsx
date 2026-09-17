import { Cake, Heart } from "lucide-react";
import { useNavigate } from "react-router-dom";
import type { PersonOccurrence } from "../lib/people";

// PersonChip — a birthday/anniversary marker on the calendar. Clicking opens
// the person's profile page.
export function PersonChip({ occasion, onOpen }: { occasion: PersonOccurrence; onOpen?: () => void }) {
  const navigate = useNavigate();
  const turned = occasion.turning ? ` · turns ${occasion.turning}` : "";
  return (
    <button
      type="button"
      className={`person-chip color-${occasion.color}`}
      title={`${occasion.name} · ${occasion.label}${turned}`}
      onClick={(e) => {
        e.stopPropagation();
        onOpen?.();
        navigate(`/people/${occasion.personId}`);
      }}
    >
      {occasion.label === "birthday" ? <Cake size={11} /> : <Heart size={11} />}
      <span className="title">{occasion.name}</span>
    </button>
  );
}
