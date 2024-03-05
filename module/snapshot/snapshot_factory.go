/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package snapshot

import (
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"
)

//var log = logger.GetLogger(logger.MODULE_SNAPSHOT)

type Factory struct {
}

// NewSnapshotManager returns new snapshot manager
func (f *Factory) NewSnapshotManager(blockchainStore protocol.BlockchainStore,
	log protocol.Logger) protocol.SnapshotManager {
	log.Debugf("use the common Snapshot.")
	return &ManagerImpl{
		snapshots: make(map[utils.BlockFingerPrint]*SnapshotImpl, 1024),
		delegate: &ManagerDelegate{
			blockchainStore: blockchainStore,
			log:             log,
		},
		log: log,
	}
}

// NewSnapshotEvidenceMgr returns new snapshot evidence manager
func (f *Factory) NewSnapshotEvidenceMgr(blockchainStore protocol.BlockchainStore,
	log protocol.Logger) protocol.SnapshotManager {
	log.Debugf("use the evidence Snapshot.")
	return &ManagerEvidence{
		snapshots: make(map[utils.BlockFingerPrint]*SnapshotEvidence, 1024),
		delegate: &ManagerDelegate{
			blockchainStore: blockchainStore,
			log:             log,
		},
		log: log,
	}
}
