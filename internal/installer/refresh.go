package installer

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

const emulationStationReloadURL = "http://127.0.0.1:1234/reloadgames"

type OperationOutcome struct {
	GameListChanged         bool
	GameListRefreshAccepted bool
	RestartRequired         bool
}

// outcome performs the optional game-list refresh after a commit and returns
// what the caller should report.
func (m Manager) outcome(ctx context.Context, gameListChanged bool) OperationOutcome {
	outcome := OperationOutcome{GameListChanged: gameListChanged}
	if gameListChanged {
		if err := m.refreshGameList(ctx); err != nil {
			outcome.RestartRequired = true
			m.event("gamelist_refresh_unavailable", "error", err.Error(), "restart_required", "true")
		} else {
			outcome.GameListRefreshAccepted = true
			m.event("gamelist_refresh_accepted")
		}
	}
	return outcome
}

func (m Manager) refreshGameList(ctx context.Context) error {
	endpoint := m.RefreshURL
	if endpoint == "" {
		endpoint = emulationStationReloadURL
	}
	client := m.RefreshClient
	if client == nil {
		client = &http.Client{Timeout: 3 * time.Second, Transport: &http.Transport{Proxy: nil}}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("EmulationStation did not accept reload: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("EmulationStation reload returned HTTP %d", response.StatusCode)
	}
	return nil
}
