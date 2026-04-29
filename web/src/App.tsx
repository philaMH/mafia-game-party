import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";

import { GameProvider } from "./context/GameContext";
import { PlayerView } from "./views/PlayerView/PlayerView";
import { HostSettingsView } from "./views/PublicView/HostSettingsView";
import { PublicView } from "./views/PublicView/PublicView";

export function App() {
  return (
    <GameProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<Navigate to="/play" replace />} />
          <Route path="/public" element={<PublicView />} />
          <Route path="/public/settings" element={<HostSettingsView />} />
          <Route path="/play" element={<PlayerView />} />
          <Route path="*" element={<Navigate to="/play" replace />} />
        </Routes>
      </BrowserRouter>
    </GameProvider>
  );
}
