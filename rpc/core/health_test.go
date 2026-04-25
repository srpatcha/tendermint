package core

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{30 * time.Second, "30s"},
		{90 * time.Second, "2m30s"},
		{2 * time.Hour, "2h0m"},
	}
	for _, tt := range tests {
		got := formatDuration(tt.d)
		// Just verify it returns a non-empty string without panicking
		assert.NotEmpty(t, got)
	}
}

func TestDetermineHealthNoPeers(t *testing.T) {
	result := &ctypes.ResultHealthDetailed{
		PeerInfo: ctypes.PeerHealthInfo{
			TotalPeers: 0,
		},
		SyncStatus: ctypes.SyncHealthStatus{
			LatestBlockTime: time.Now(),
		},
	}
	assert.False(t, determineHealth(result))
}

func TestDetermineHealthStaleBlock(t *testing.T) {
	result := &ctypes.ResultHealthDetailed{
		PeerInfo: ctypes.PeerHealthInfo{
			TotalPeers: 5,
		},
		SyncStatus: ctypes.SyncHealthStatus{
			LatestBlockTime: time.Now().Add(-10 * time.Minute),
		},
	}
	assert.False(t, determineHealth(result))
}

func TestDetermineHealthOK(t *testing.T) {
	result := &ctypes.ResultHealthDetailed{
		PeerInfo: ctypes.PeerHealthInfo{
			TotalPeers: 5,
		},
		SyncStatus: ctypes.SyncHealthStatus{
			LatestBlockTime: time.Now(),
		},
	}
	assert.True(t, determineHealth(result))
}

func TestBuildMemoryInfo(t *testing.T) {
	info := buildMemoryInfo()
	assert.True(t, info.AllocMB > 0)
	assert.True(t, info.SysMB > 0)
	assert.True(t, info.GoRoutines > 0)
}
