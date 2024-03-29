package minter

import (
	"encoding/hex"
	"errors"
	snapshottypes "github.com/cosmos/cosmos-sdk/snapshots/types"
	abci "github.com/tendermint/tendermint/abci/types"
)

// List available snapshots
func (blockchain *Blockchain) ListSnapshots(req abci.RequestListSnapshots) abci.ResponseListSnapshots {
	blockchain.logger.Debug("ListSnapshots")
	resp := abci.ResponseListSnapshots{Snapshots: []*abci.Snapshot{}}
	if blockchain.snapshotManager == nil {
		return resp
	}

	snapshots, err := blockchain.snapshotManager.List()
	if err != nil {
		//blockchain.logger.Error("failed to list snapshots", "err", err)
		return resp
	}

	for _, snapshot := range snapshots {
		abciSnapshot, err := snapshot.ToABCI()
		if err != nil {
			//blockchain.logger.Error("failed to list snapshots", "err", err)
			return resp
		}
		resp.Snapshots = append(resp.Snapshots, &abciSnapshot)
	}

	return resp
}

// Offer a snapshot to the application
func (blockchain *Blockchain) OfferSnapshot(req abci.RequestOfferSnapshot) abci.ResponseOfferSnapshot {
	blockchain.logger.Info("Processing OfferSnapshot...",
		"AppHash", hex.EncodeToString(req.AppHash),
		"Height", req.Snapshot.Height,
		"Format", req.Snapshot.Format,
		"Chunks", req.Snapshot.Chunks,
	)
	if blockchain.snapshotManager == nil {
		//blockchain.logger.Error("snapshot manager not configured")
		return abci.ResponseOfferSnapshot{Result: abci.ResponseOfferSnapshot_ABORT}
	}

	if req.Snapshot == nil {
		blockchain.logger.Error("received nil snapshot")
		return abci.ResponseOfferSnapshot{Result: abci.ResponseOfferSnapshot_REJECT}
	}

	snapshot, err := snapshottypes.SnapshotFromABCI(req.Snapshot)
	if err != nil {
		//blockchain.logger.Error("failed to decode snapshot metadata", "err", err)
		return abci.ResponseOfferSnapshot{Result: abci.ResponseOfferSnapshot_REJECT}
	}

	err = blockchain.snapshotManager.Restore(snapshot)
	switch {
	case err == nil:
		blockchain.logger.Info("Done OfferSnapshot!",
			"AppHash", hex.EncodeToString(req.AppHash),
			"Height", req.Snapshot.Height,
			"Format", req.Snapshot.Format,
			"Chunks", req.Snapshot.Chunks,
		)
		return abci.ResponseOfferSnapshot{Result: abci.ResponseOfferSnapshot_ACCEPT}

	case errors.Is(err, snapshottypes.ErrUnknownFormat):
		return abci.ResponseOfferSnapshot{Result: abci.ResponseOfferSnapshot_REJECT_FORMAT}

	case errors.Is(err, snapshottypes.ErrInvalidMetadata):
		blockchain.logger.Error(
			"rejecting invalid snapshot",
			"height", req.Snapshot.Height,
			"format", req.Snapshot.Format,
			"err", err,
		)
		return abci.ResponseOfferSnapshot{Result: abci.ResponseOfferSnapshot_REJECT}

	default:
		blockchain.logger.Error(
			"failed to restore snapshot",
			"height", req.Snapshot.Height,
			"format", req.Snapshot.Format,
			"err", err,
		)

		// We currently don't support resetting the IAVL stores and retrying a different snapshot,
		// so we ask Tendermint to abort all snapshot restoration.
		return abci.ResponseOfferSnapshot{Result: abci.ResponseOfferSnapshot_ABORT}
	}
}

// Load a snapshot chunk
func (blockchain *Blockchain) LoadSnapshotChunk(req abci.RequestLoadSnapshotChunk) abci.ResponseLoadSnapshotChunk {
	blockchain.logger.Info("Processing LoadSnapshotChunk...", "req", req.String())
	if blockchain.snapshotManager == nil {
		return abci.ResponseLoadSnapshotChunk{}
	}
	chunk, err := blockchain.snapshotManager.LoadChunk(req.Height, req.Format, req.Chunk)
	if err != nil {
		blockchain.logger.Error(
			"failed to load snapshot chunk",
			"height", req.Height,
			"format", req.Format,
			"chunk", req.Chunk,
			"err", err,
		)
		return abci.ResponseLoadSnapshotChunk{}
	}
	blockchain.logger.Debug("Done LoadSnapshotChunk!", "req", req.String())
	return abci.ResponseLoadSnapshotChunk{Chunk: chunk}
}

// Apply a shapshot chunk
func (blockchain *Blockchain) ApplySnapshotChunk(req abci.RequestApplySnapshotChunk) abci.ResponseApplySnapshotChunk {
	blockchain.logger.Info("Processing ApplySnapshotChunk...", "Index", req.Index, "Sender", req.Sender)
	if blockchain.snapshotManager == nil {
		blockchain.logger.Error("snapshot manager not configured")
		return abci.ResponseApplySnapshotChunk{Result: abci.ResponseApplySnapshotChunk_ABORT}
	}

	_, err := blockchain.snapshotManager.RestoreChunk(req.Chunk)
	switch {
	case err == nil:
		blockchain.logger.Info("Done ApplySnapshotChunk!", "Index", req.Index, "Sender", req.Sender)
		return abci.ResponseApplySnapshotChunk{Result: abci.ResponseApplySnapshotChunk_ACCEPT}

	case errors.Is(err, snapshottypes.ErrChunkHashMismatch):
		blockchain.logger.Error(
			"chunk checksum mismatch; rejecting sender and requesting refetch",
			"chunk", req.Index,
			"sender", req.Sender,
			"err", err,
		)
		return abci.ResponseApplySnapshotChunk{
			Result:        abci.ResponseApplySnapshotChunk_RETRY,
			RefetchChunks: []uint32{req.Index},
			RejectSenders: []string{req.Sender},
		}

	default:
		blockchain.logger.Error("failed to restore snapshot", "err", err)
		return abci.ResponseApplySnapshotChunk{Result: abci.ResponseApplySnapshotChunk_ABORT}
	}
}

// snapshot takes a snapshot of the current state and prunes any old snapshottypes.
func (blockchain *Blockchain) snapshot(height int64) {
	if blockchain.stopped {
		blockchain.logger.Info("node stopped, snapshot skipped", "height", height)
		return
	}

	blockchain.wgSnapshot.Wait()
	blockchain.wgSnapshot.Add(1)
	defer blockchain.wgSnapshot.Done()

	blockchain.logger.Info("creating state snapshot", "height", height)

	snapshot, err := blockchain.snapshotManager.Create(uint64(height))
	if err != nil {
		blockchain.appDB.WG.Done()
		blockchain.logger.Error("failed to create state snapshot", "height", height, "err", err)
		return
	}

	blockchain.logger.Info("completed state snapshot", "height", height, "format", snapshot.Format)

	if blockchain.snapshotKeepRecent > 0 {
		blockchain.logger.Debug("pruning state snapshots")

		pruned, err := blockchain.snapshotManager.Prune(blockchain.snapshotKeepRecent)
		if err != nil {
			blockchain.logger.Error("Failed to prune state snapshots", "err", err)
			return
		}

		blockchain.logger.Debug("pruned state snapshots", "pruned", pruned)
	}
}
