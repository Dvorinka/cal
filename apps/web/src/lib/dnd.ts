/** Entry id being dragged, mirrored to dataTransfer for interop. */
export const ENTRY_MIME = "application/x-cal-entry";

export function startEntryDrag(event: React.DragEvent, entryId: string) {
  event.dataTransfer.setData(ENTRY_MIME, entryId);
  event.dataTransfer.setData("text/plain", entryId);
  event.dataTransfer.effectAllowed = "move";
}

export function draggedEntryId(event: React.DragEvent): string | undefined {
  return event.dataTransfer.getData(ENTRY_MIME) || event.dataTransfer.getData("text/plain") || undefined;
}

/** Highlight a drop target while a drag is over it; returns cleanup handlers. */
export function dropTargetProps(onDrop: (event: React.DragEvent) => void) {
  return {
    onDragOver(event: React.DragEvent) {
      if (event.dataTransfer.types.includes(ENTRY_MIME) || event.dataTransfer.types.includes("text/plain")) {
        event.preventDefault();
        event.dataTransfer.dropEffect = "move";
        event.currentTarget.classList.add("drop-target");
      }
    },
    onDragLeave(event: React.DragEvent) {
      event.currentTarget.classList.remove("drop-target");
    },
    onDrop(event: React.DragEvent) {
      event.preventDefault();
      event.currentTarget.classList.remove("drop-target");
      onDrop(event);
    },
  };
}
