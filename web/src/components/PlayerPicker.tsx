import { memo } from "react";
import type { Player, PlayerID } from "../types/wire";

interface Props {
  players: Player[];
  value?: PlayerID;
  disabled?: boolean;
  onChange: (id: PlayerID) => void;
}

interface ItemProps {
  player: Player;
  selected: boolean;
  disabled: boolean;
  onSelect: (id: PlayerID) => void;
}

const Item = memo(function Item({ player, selected, disabled, onSelect }: ItemProps) {
  return (
    <button
      type="button"
      aria-pressed={selected}
      disabled={disabled || !player.alive}
      onClick={() => onSelect(player.id)}
      style={{
        background: selected ? "var(--accent)" : "var(--card)",
        color: selected ? "#fff" : "var(--fg)",
        opacity: player.alive ? 1 : 0.4,
        padding: "0.5rem 1rem",
        borderRadius: "0.375rem",
        border: "1px solid var(--border)",
        minWidth: "6rem",
      }}
    >
      {player.name}
      {!player.alive && <span aria-hidden> ✕</span>}
    </button>
  );
});

// PlayerPicker shows a horizontal radio-style list. Selection sends
// immediately (BR-U5-INPUT-2). Items are memoised by player id so
// re-renders triggered by unrelated state changes do not cascade.
export function PlayerPicker({ players, value, disabled, onChange }: Props) {
  return (
    <div
      role="radiogroup"
      style={{
        display: "flex",
        flexWrap: "wrap",
        gap: "0.5rem",
      }}
    >
      {players.map((p) => (
        <Item
          key={p.id}
          player={p}
          selected={p.id === value}
          disabled={!!disabled}
          onSelect={onChange}
        />
      ))}
    </div>
  );
}
