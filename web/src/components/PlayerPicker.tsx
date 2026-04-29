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
  index: number;
  onSelect: (id: PlayerID) => void;
}

const Item = memo(function Item({ player, selected, disabled, index, onSelect }: ItemProps) {
  const inactive = disabled || !player.alive;
  return (
    <button
      type="button"
      role="radio"
      aria-checked={selected}
      disabled={inactive}
      onClick={() => onSelect(player.id)}
      className={"vote-tile" + (selected ? " selected" : "") + (!player.alive ? " dead" : "")}
      style={{ minWidth: "6.5rem" }}
    >
      <span className="vt-meta">{String(index + 1).padStart(2, "0")}</span>
      <div className={"avatar sm" + (!player.alive ? " dead" : selected ? " target" : "")}>
        {player.name.slice(0, 1)}
      </div>
      <span className="vt-name">
        {player.name}
        {!player.alive && <span aria-hidden> ✕</span>}
      </span>
    </button>
  );
});

// PlayerPicker shows a noir-styled grid of selectable player tiles.
// Selection sends immediately (BR-U5-INPUT-2). Items are memoised by
// player id so re-renders triggered by unrelated state changes do not
// cascade.
export function PlayerPicker({ players, value, disabled, onChange }: Props) {
  return (
    <div
      role="radiogroup"
      className="vote-tile-grid"
      style={{
        display: "grid",
        gridTemplateColumns: "repeat(auto-fit, minmax(6.5rem, 1fr))",
        gap: "0.5rem",
      }}
    >
      {players.map((p, i) => (
        <Item
          key={p.id}
          index={i}
          player={p}
          selected={p.id === value}
          disabled={!!disabled}
          onSelect={onChange}
        />
      ))}
    </div>
  );
}
