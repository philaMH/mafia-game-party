import { useEffect } from "react";

interface Toast {
  code: string;
  message: string;
  addedAt: number;
}

interface Props {
  errors: Toast[];
  onDismiss: (addedAt: number) => void;
}

const TTL_MS = 5000;

export function ToastList({ errors, onDismiss }: Props) {
  useEffect(() => {
    if (errors.length === 0) return;
    const oldest = errors[0];
    if (!oldest) return;
    const remaining = Math.max(0, TTL_MS - (Date.now() - oldest.addedAt));
    const timer = setTimeout(() => onDismiss(oldest.addedAt), remaining);
    return () => clearTimeout(timer);
  }, [errors, onDismiss]);

  if (errors.length === 0) return null;

  return (
    <div
      role="status"
      aria-live="polite"
      style={{
        position: "fixed",
        top: "1rem",
        right: "1rem",
        display: "flex",
        flexDirection: "column",
        gap: "0.5rem",
        maxWidth: "20rem",
        zIndex: 200,
      }}
    >
      {errors.map((e) => (
        <button
          type="button"
          key={e.addedAt}
          onClick={() => onDismiss(e.addedAt)}
          className="serif"
          style={{
            background: "linear-gradient(180deg, rgba(58,14,14,0.92), rgba(20,4,4,0.95))",
            color: "var(--paper)",
            border: "1px solid var(--red)",
            padding: "0.6rem 0.85rem",
            cursor: "pointer",
            textAlign: "left",
            fontStyle: "italic",
            boxShadow: "0 0 18px var(--red-glow)",
            letterSpacing: "0.02em",
            borderRadius: 0,
          }}
        >
          <span
            className="eyebrow red"
            style={{ display: "block", marginBottom: "0.25rem", fontSize: "0.65rem" }}
          >
            ALERT
          </span>
          {e.message}
        </button>
      ))}
    </div>
  );
}
