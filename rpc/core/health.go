package core

import (
	"encoding/json"
	"fmt"
	"runtime"
	"time"

	"github.com/tendermint/tendermint/p2p"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	rpctypes "github.com/tendermint/tendermint/rpc/jsonrpc/types"
)

// Health gets node health. Returns empty result (200 OK) on success, no
// response - in case of an error.
// More: https://docs.tendermint.com/v0.34/rpc/#/Info/health
func Health(ctx *rpctypes.Context) (*ctypes.ResultHealth, error) {
	return &ctypes.ResultHealth{}, nil
}

// HealthDetailed returns comprehensive node health information including
// sync status, peer connections, consensus state, and memory metrics.
func HealthDetailed(ctx *rpctypes.Context) (*ctypes.ResultHealthDetailed, error) {
	result := &ctypes.ResultHealthDetailed{
		Timestamp: time.Now().UTC(),
		IsHealthy: true,
	}

	// Node identification
	if nodeInfo, ok := env.P2PTransport.NodeInfo().(p2p.DefaultNodeInfo); ok {
		result.NodeID = string(nodeInfo.DefaultNodeID)
		result.NodeInfo = ctypes.NodeHealthInfo{
			Version:    nodeInfo.ProtocolVersion.App.String(),
			Network:    nodeInfo.Network,
			Moniker:    nodeInfo.Moniker,
			TxIndexOn:  nodeInfo.Other.TxIndex == "on",
			RPCAddress: nodeInfo.Other.RPCAddress,
		}
	}

	// Sync status
	latestHeight := env.BlockStore.Height()
	result.SyncStatus = buildSyncStatus(latestHeight)
	result.IsSyncing = env.ConsensusReactor.WaitSync()

	// Peer information
	result.PeerInfo = buildPeerInfo()

	// Consensus information
	result.ConsensusInfo = buildConsensusInfo()

	// Memory metrics
	result.MemoryInfo = buildMemoryInfo()

	// Determine overall health
	result.IsHealthy = determineHealth(result)

	return result, nil
}

// buildSyncStatus constructs sync health status from block store state.
func buildSyncStatus(latestHeight int64) ctypes.SyncHealthStatus {
	status := ctypes.SyncHealthStatus{
		LatestBlockHeight: latestHeight,
		CatchingUp:        env.ConsensusReactor.WaitSync(),
	}

	if latestHeight > 0 {
		if latestBlockMeta := env.BlockStore.LoadBlockMeta(latestHeight); latestBlockMeta != nil {
			blockTime := latestBlockMeta.Header.Time
			status.LatestBlockTime = blockTime
			age := time.Since(blockTime)
			status.LatestBlockAge = formatDuration(age)
		}
	}

	if earliestBlockMeta := env.BlockStore.LoadBaseMeta(); earliestBlockMeta != nil {
		status.EarliestBlockHeight = earliestBlockMeta.Header.Height

		// Calculate blocks per second over known range
		if latestHeight > status.EarliestBlockHeight {
			if latestMeta := env.BlockStore.LoadBlockMeta(latestHeight); latestMeta != nil {
				elapsed := latestMeta.Header.Time.Sub(earliestBlockMeta.Header.Time)
				if elapsed.Seconds() > 0 {
					blocks := float64(latestHeight - status.EarliestBlockHeight)
					status.BlocksPerSecond = blocks / elapsed.Seconds()
				}
			}
		}
	}

	return status
}

// buildPeerInfo constructs peer health information.
func buildPeerInfo() ctypes.PeerHealthInfo {
	info := ctypes.PeerHealthInfo{
		IsListening: env.P2PTransport.IsListening(),
	}

	peersList := env.P2PPeers.Peers().List()
	info.TotalPeers = len(peersList)

	for _, peer := range peersList {
		if peer.IsOutbound() {
			info.OutboundPeers++
		} else {
			info.InboundPeers++
		}
	}

	return info
}

// buildConsensusInfo constructs consensus health information.
func buildConsensusInfo() ctypes.ConsensusHealthInfo {
	info := ctypes.ConsensusHealthInfo{}

	csHeight, validators := env.ConsensusState.GetValidators()
	info.Height = csHeight
	info.ValidatorCount = len(validators)

	var totalVotingPower int64
	for _, v := range validators {
		totalVotingPower += v.VotingPower
	}
	info.VotingPower = totalVotingPower

	// Extract round and step from consensus state JSON
	if roundStateJSON, err := env.ConsensusState.GetRoundStateSimpleJSON(); err == nil {
		var roundState map[string]interface{}
		if err := json.Unmarshal(roundStateJSON, &roundState); err == nil {
			if rs, ok := roundState["round_state"].(map[string]interface{}); ok {
				if step, ok := rs["step"].(string); ok {
					info.Step = step
				}
				if round, ok := rs["round"].(float64); ok {
					info.Round = int32(round)
				}
			}
		}
	}

	return info
}

// buildMemoryInfo constructs memory usage metrics.
func buildMemoryInfo() ctypes.MemoryHealthInfo {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return ctypes.MemoryHealthInfo{
		AllocMB:      float64(memStats.Alloc) / 1024 / 1024,
		TotalAllocMB: float64(memStats.TotalAlloc) / 1024 / 1024,
		SysMB:        float64(memStats.Sys) / 1024 / 1024,
		NumGC:        memStats.NumGC,
		GoRoutines:   runtime.NumGoroutine(),
	}
}

// determineHealth checks various conditions to determine overall health.
func determineHealth(result *ctypes.ResultHealthDetailed) bool {
	// Unhealthy if no peers
	if result.PeerInfo.TotalPeers == 0 {
		return false
	}

	// Unhealthy if latest block is older than 5 minutes
	if !result.SyncStatus.LatestBlockTime.IsZero() {
		age := time.Since(result.SyncStatus.LatestBlockTime)
		if age > 5*time.Minute {
			return false
		}
	}

	return true
}

// formatDuration formats a duration into a human-readable string.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.0fm%.0fs", d.Minutes(), d.Seconds()-d.Minutes()*60)
	}
	return fmt.Sprintf("%.0fh%.0fm", d.Hours(), d.Minutes()-d.Hours()*60)
}

// nodeInfoFromVersion creates a simple version string.
func nodeInfoFromVersion(v p2p.ProtocolVersion) string {
	return fmt.Sprintf("p2p:%d block:%d app:%d", v.P2P, v.Block, v.App)
}
