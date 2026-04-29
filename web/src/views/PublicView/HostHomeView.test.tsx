import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";

import { GameContext, type GameContextValue } from "../../context/GameContext";
import { initialState } from "../../context/reducer";
import { defaultOptions } from "../../types/wire";

import { HostHomeView } from "./HostHomeView";

function makeCtx(overrides: Partial<GameContextValue> = {}): GameContextValue {
  return {
    ...initialState,
    send: vi.fn(),
    toggleVoice: vi.fn(),
    ackError: vi.fn(),
    logout: vi.fn(),
    hostOptions: defaultOptions(8),
    saveHostOptions: vi.fn(),
    ...overrides,
  };
}

function renderHome(ctx: GameContextValue) {
  return render(
    <GameContext.Provider value={ctx}>
      <MemoryRouter initialEntries={["/public"]}>
        <Routes>
          <Route path="/public" element={<HostHomeView />} />
          <Route
            path="/public/settings"
            element={<div data-testid="settings-route">SETTINGS</div>}
          />
        </Routes>
      </MemoryRouter>
    </GameContext.Provider>,
  );
}

describe("HostHomeView", () => {
  it("renders the two main-menu buttons", () => {
    renderHome(makeCtx());
    expect(screen.getByRole("button", { name: /게임 시작/ })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /설정/ })).toBeInTheDocument();
  });

  it("clicking 게임 시작 dispatches host:open-room with the current options", () => {
    const send = vi.fn();
    const opts = { ...defaultOptions(8), discussionSeconds: 240 };
    renderHome(makeCtx({ send, hostOptions: opts }));
    fireEvent.click(screen.getByRole("button", { name: /게임 시작/ }));
    expect(send).toHaveBeenCalledWith({ type: "host:open-room", options: opts });
  });

  it("clicking 설정 navigates to /public/settings", () => {
    renderHome(makeCtx());
    fireEvent.click(screen.getByRole("button", { name: /설정/ }));
    expect(screen.getByTestId("settings-route")).toBeInTheDocument();
  });
});
