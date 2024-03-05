/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package snapshot

import (
	"sync"

	"go.uber.org/atomic"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"
)

type ManagerDelegate struct {
	lock            sync.Mutex
	blockchainStore protocol.BlockchainStore
	log             protocol.Logger
}

func (m *ManagerDelegate) calcSnapshotFingerPrint(snapshot *SnapshotImpl) utils.BlockFingerPrint {
	if snapshot == nil {
		return ""
	}
	chainId := snapshot.chainId
	blockHeight := snapshot.blockHeight
	blockTimestamp := snapshot.blockTimestamp
	blockProposer := snapshot.blockProposer
	preBlockHash := snapshot.preBlockHash
	blockProposerBytes, _ := blockProposer.Marshal()
	return utils.CalcFingerPrint(chainId, blockHeight, blockTimestamp, blockProposerBytes, preBlockHash,
		snapshot.txRoot, snapshot.dagHash, snapshot.rwSetHash)
}

func (m *ManagerDelegate) calcSnapshotFingerPrintWithoutTx(snapshot *SnapshotImpl) utils.BlockFingerPrint {
	if snapshot == nil {
		return ""
	}
	chainId := snapshot.chainId
	blockHeight := snapshot.blockHeight
	blockTimestamp := snapshot.blockTimestamp
	blockProposer := snapshot.blockProposer
	preBlockHash := snapshot.preBlockHash
	blockProposerBytes, _ := blockProposer.Marshal()
	return utils.CalcFingerPrint(chainId, blockHeight, blockTimestamp, blockProposerBytes, preBlockHash,
		nil, nil, nil)
}

func (m *ManagerDelegate) makeSnapshotImpl(block *commonPb.Block) *SnapshotImpl {
	// If the corresponding Snapshot does not exist, create one
	txCount := len(block.Txs) // as map init size
	lastChainConfig, err := m.blockchainStore.GetLastChainConfig()
	if err != nil || lastChainConfig == nil {
		m.log.Errorf("failed to get last chain config, %v", err)
		return nil
	}
	snapshotImpl := &SnapshotImpl{
		blockchainStore: m.blockchainStore,
		sealed:          atomic.NewBool(false),
		preSnapshot:     nil,
		log:             m.log,
		txResultMap:     make(map[string]*commonPb.Result, txCount),

		chainId:         block.Header.ChainId,
		blockHeight:     block.Header.BlockHeight,
		blockVersion:    block.Header.BlockVersion,
		blockTimestamp:  block.Header.BlockTimestamp,
		blockProposer:   block.Header.Proposer,
		preBlockHash:    block.Header.PreBlockHash,
		lastChainConfig: lastChainConfig,

		txTable:    nil,
		readTable:  make(map[string]*sv, txCount),
		writeTable: make(map[string]*sv, txCount),

		txRoot:    block.Header.TxRoot,
		dagHash:   block.Header.DagHash,
		rwSetHash: block.Header.RwSetRoot,
	}
	return snapshotImpl
}
