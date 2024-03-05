/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sync

import (
	"testing"
	"time"

	"chainmaker.org/chainmaker/localconf/v2"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	netPb "chainmaker.org/chainmaker/pb-go/v2/net"
	syncPb "chainmaker.org/chainmaker/pb-go/v2/sync"
	"chainmaker.org/chainmaker/protocol/v2/test"

	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func getNodeStatusReq(t *testing.T) []byte {
	msg := &syncPb.SyncMsg{Type: syncPb.SyncMsg_NODE_STATUS_REQ}
	bz, err := msg.Marshal()
	require.NoError(t, err)
	return bz
}

func getNodeStatusResp(t *testing.T, height uint64) []byte {
	msg := &syncPb.BlockHeightBCM{BlockHeight: height}
	bz, err := msg.Marshal()
	require.NoError(t, err)
	msg2 := &syncPb.SyncMsg{Type: syncPb.SyncMsg_NODE_STATUS_RESP, Payload: bz}
	bz, err = msg2.Marshal()
	require.NoError(t, err)
	return bz
}

func getBlockReq(t *testing.T, height, batchSize uint64) []byte {
	msg := &syncPb.BlockSyncReq{BlockHeight: height, BatchSize: batchSize}
	bz, err := msg.Marshal()
	require.NoError(t, err)
	msg2 := &syncPb.SyncMsg{Type: syncPb.SyncMsg_BLOCK_SYNC_REQ, Payload: bz}
	bz, err = msg2.Marshal()
	require.NoError(t, err)
	return bz
}

func getBlockResp(t *testing.T, height uint64) []byte {
	msg := &syncPb.SyncBlockBatch{
		Data: &syncPb.SyncBlockBatch_BlockinfoBatch{
			BlockinfoBatch: &syncPb.BlockInfoBatch{
				Batch: []*commonPb.BlockInfo{
					{Block: &commonPb.Block{Header: &commonPb.BlockHeader{BlockHeight: height}}},
				},
			},
		},
		WithRwset: false,
	}
	bz, err := msg.Marshal()
	require.NoError(t, err)
	msg2 := &syncPb.SyncMsg{Type: syncPb.SyncMsg_BLOCK_SYNC_RESP, Payload: bz}
	bz, err = msg2.Marshal()
	require.NoError(t, err)
	return bz
}

func getBlockRespWithRWset(height uint64) ([]byte, error) {
	msg := &syncPb.SyncBlockBatch{
		Data: &syncPb.SyncBlockBatch_BlockinfoBatch{
			BlockinfoBatch: &syncPb.BlockInfoBatch{
				Batch: []*commonPb.BlockInfo{
					{Block: &commonPb.Block{Header: &commonPb.BlockHeader{BlockHeight: height}}},
				},
			},
		},
		WithRwset: true,
	}
	bz, err := msg.Marshal()
	if err != nil {
		return nil, err
	}
	msg2 := &syncPb.SyncMsg{Type: syncPb.SyncMsg_BLOCK_SYNC_RESP, Payload: bz}
	bz, err = msg2.Marshal()
	if err != nil {
		return nil, err
	}
	return bz, nil
}

func initTestSync(t *testing.T) (protocol.SyncService, func()) {
	ctrl := gomock.NewController(t)
	mockNet := newMockNet(ctrl)
	mockMsgBus := newMockMessageBus(ctrl)
	mockVerify := newMockVerifier(ctrl)
	mockStore := newMockBlockChainStore(ctrl)
	mockLedger := newMockLedgerCache(ctrl, &commonPb.Block{Header: &commonPb.BlockHeader{BlockHeight: 10}})
	mockCommit := newMockCommitter(ctrl, mockLedger)
	log := &test.GoLogger{}
	// localconf.ChainMakerConfig.SyncConfig.SchedulerTick = 10
	// localconf.ChainMakerConfig.SyncConfig.ProcessBlockTick = 10
	service := NewBlockChainSyncServer("chain1", mockNet, mockMsgBus, mockStore, mockLedger, mockVerify, mockCommit, log)
	require.NoError(t, service.Start())
	return service, func() {
		service.Stop()
		ctrl.Finish()
	}
}

func TestBlockChainSyncServer_Start(t *testing.T) {
	service, fn := initTestSync(t)
	defer fn()

	// consume message
	implSync := service.(*BlockChainSyncServer)
	bz := getNodeStatusResp(t, 110)
	require.NoError(t, implSync.blockSyncMsgHandler("node1", bz, netPb.NetMsg_SYNC_BLOCK_MSG))
	bz = getNodeStatusResp(t, 120)
	require.NoError(t, implSync.blockSyncMsgHandler("node2", bz, netPb.NetMsg_SYNC_BLOCK_MSG))
	time.Sleep(3 * time.Millisecond)
	require.EqualValues(t, "pendingRecvHeight: 11, peers num: 2, blockStates num: 110, "+
		"pendingBlocks num: 0, receivedBlocks num: 0", implSync.scheduler.getServiceState())
	require.EqualValues(t, "pendingBlockHeight: 11, queue num: 0", implSync.processor.getServiceState())

	bz = getBlockResp(t, 11)
	require.NoError(t, implSync.blockSyncMsgHandler("node2", bz, netPb.NetMsg_SYNC_BLOCK_MSG)) // TODO
	//require.EqualValues(t, "pendingRecvHeight: 11, peers num: 2, blockStates num: 110, pendingBlocks num: 0, receivedBlocks num: 1",
	//	implSync.scheduler.getServiceState())
	//time.Sleep(time.Second)
	//require.EqualValues(t, "pendingBlockHeight: 12, queue num: 0", implSync.processor.getServiceState())
}

func TestSyncMsg_NODE_STATUS_REQ(t *testing.T) {
	service, fn := initTestSync(t)
	defer fn()
	implSync := service.(*BlockChainSyncServer)

	// 1. req node status
	require.NoError(t, implSync.blockSyncMsgHandler("node1", getNodeStatusReq(t), netPb.NetMsg_SYNC_BLOCK_MSG))
	//require.EqualValues(t, 1, len(implSync.net.(*MockNet).sendMsgs))
	//require.EqualValues(t, "msgType: 6, to: [node1]", implSync.net.(*MockNet).sendMsgs[0])

	require.NoError(t, implSync.blockSyncMsgHandler("node2", getNodeStatusReq(t), netPb.NetMsg_SYNC_BLOCK_MSG))
	//require.EqualValues(t, 2, len(implSync.net.(*MockNet).sendMsgs))
	//require.EqualValues(t, "msgType: 6, to: [node2]", implSync.net.(*MockNet).sendMsgs[1])
}

func TestSyncBlock_Req(t *testing.T) {
	service, fn := initTestSync(t)
	defer fn()
	implSync := service.(*BlockChainSyncServer)

	_ = implSync.blockChainStore.PutBlock(&commonPb.Block{Header: &commonPb.BlockHeader{BlockHeight: 99}}, nil)
	_ = implSync.blockChainStore.PutBlock(&commonPb.Block{Header: &commonPb.BlockHeader{BlockHeight: 100}}, nil)
	_ = implSync.blockChainStore.PutBlock(&commonPb.Block{Header: &commonPb.BlockHeader{BlockHeight: 101}}, nil)

	require.NoError(t, implSync.blockSyncMsgHandler("node1", getBlockReq(t, 99, 1), netPb.NetMsg_SYNC_BLOCK_MSG))
	//require.EqualValues(t, 1, len(implSync.net.(*MockNet).sendMsgs))
	//require.EqualValues(t, "msgType: 6, to: [node1]", implSync.net.(*MockNet).sendMsgs[0])

	require.NoError(t, implSync.blockSyncMsgHandler("node2", getBlockReq(t, 100, 2), netPb.NetMsg_SYNC_BLOCK_MSG))
	//require.EqualValues(t, 3, len(implSync.net.(*MockNet).sendMsgs))
	//require.EqualValues(t, "msgType: 6, to: [node2]", implSync.net.(*MockNet).sendMsgs[1])

	require.Error(t, implSync.blockSyncMsgHandler("node2", getBlockReq(t, 110, 2), netPb.NetMsg_SYNC_BLOCK_MSG))
}

func TestSyncMsg_NODE_STATUS_RESP(t *testing.T) {
	service, fn := initTestSync(t)
	defer fn()
	implSync := service.(*BlockChainSyncServer)

	bz := getNodeStatusResp(t, 120)
	require.NoError(t, implSync.blockSyncMsgHandler("node2", bz, netPb.NetMsg_SYNC_BLOCK_MSG))
	time.Sleep(3 * time.Second)
	require.EqualValues(t, "pendingRecvHeight: 11, peers num: 1, blockStates num: 110, "+
		"pendingBlocks num: 110, receivedBlocks num: 0", implSync.scheduler.getServiceState())
	require.EqualValues(t, "pendingBlockHeight: 11, queue num: 0", implSync.processor.getServiceState())
}

func TestSyncMsg_BLOCK_SYNC_RESP(t *testing.T) {
	service, fn := initTestSync(t)
	defer fn()
	implSync := service.(*BlockChainSyncServer)
	// modify config for a stable unit test result
	implSync.conf.livenessTick = 10 * time.Second
	// 1. add peer status
	bz := getNodeStatusResp(t, 120)
	require.NoError(t, implSync.blockSyncMsgHandler("node2", bz, netPb.NetMsg_SYNC_BLOCK_MSG))
	//测试概率失败：收到netPb.NetMsg_SYNC_BLOCK_MSG消息后会添加一个NodeStatusMsg的任务，此任务优先级较低，netPb.NetMsg_SYNC_BLOCK_MSG消息会添加一个SyncedBlockMsg类型的任务,优先级为中级
	//如果NodeStatusMsg的任务在消费前添加了一个SyncedBlockMsg类型的任务, 则SyncedBlockMsg的任务会优先被消费，此时scheduler中updateSchedulerBySyncBlockBatch函数中sch.blockStates为空，即needToProcess会被判定为false，消息不能被处理
	//sleep为了避免此情况发生
	time.Sleep(200 * time.Microsecond)
	// 2. receive block
	blkBz := getBlockResp(t, 11)
	require.NoError(t, implSync.blockSyncMsgHandler("node2", blkBz, netPb.NetMsg_SYNC_BLOCK_MSG))
	time.Sleep(4 * time.Second)
	require.EqualValues(t, "pendingRecvHeight: 12, peers num: 1, blockStates num: 109, "+
		"pendingBlocks num: 109, receivedBlocks num: 0", implSync.scheduler.getServiceState())
	require.EqualValues(t, "pendingBlockHeight: 12, queue num: 0", implSync.processor.getServiceState())
}

//func TestStopSyncBlock(t *testing.T) {
//	localconf.ChainMakerConfig.NodeConfig.FastSyncConfig.Enable = true
//	service, fn := initTestSync(t)
//	defer fn()
//	implSync := service.(*BlockChainSyncServer)
//	// modify config for a stable unit test result
//	implSync.conf.livenessTick = 10 * time.Second
//	// 1. add peer status
//	bz := getNodeStatusResp(t, 21)
//	require.NoError(t, implSync.blockSyncMsgHandler("node2", bz, netPb.NetMsg_SYNC_BLOCK_MSG))
//	time.Sleep(200 * time.Microsecond)
//	// 2. receive block
//	blkBz, err := getBlockRespWithRWset(11)
//	require.NoError(t, err)
//	require.NoError(t, implSync.blockSyncMsgHandler("node2", blkBz, netPb.NetMsg_SYNC_BLOCK_MSG))
//	time.Sleep(1 * time.Second)
//	require.EqualValues(t, "pendingRecvHeight: 12, peers num: 1, blockStates num: 10, pendingBlocks num: 10, receivedBlocks num: 0", implSync.scheduler.getServiceState())
//	//3.StopBlockSync
//	service.StopBlockSync()
//	time.Sleep(100 * time.Microsecond)
//	//4.sync node status again
//	bz = getNodeStatusResp(t, 41)
//	require.NoError(t, implSync.blockSyncMsgHandler("node2", bz, netPb.NetMsg_SYNC_BLOCK_MSG))
//	time.Sleep(1 * time.Second)
//	require.EqualValues(t, "pendingRecvHeight: 12, peers num: 1, blockStates num: 30, pendingBlocks num: 10, receivedBlocks num: 10", implSync.scheduler.getServiceState())
//}

func TestStopSyncBlock(t *testing.T) {
	localconf.ChainMakerConfig.NodeConfig.FastSyncConfig.Enable = true
	service, fn := initTestSync(t)
	defer fn()
	implSync := service.(*BlockChainSyncServer)
	// modify config for a stable unit test result
	implSync.conf.livenessTick = 10 * time.Second
	// 1. add peer status
	bz := getNodeStatusResp(t, 21)
	require.NoError(t, implSync.blockSyncMsgHandler("node2", bz, netPb.NetMsg_SYNC_BLOCK_MSG))
	time.Sleep(200 * time.Microsecond)
	// 2. receive block
	blkBz, err := getBlockRespWithRWset(11)
	require.NoError(t, err)
	require.NoError(t, implSync.blockSyncMsgHandler("node2", blkBz, netPb.NetMsg_SYNC_BLOCK_MSG))
	time.Sleep(3 * time.Second)
	require.EqualValues(t, "pendingRecvHeight: 12, peers num: 1, blockStates num: 10, pendingBlocks num: 10, receivedBlocks num: 0", implSync.scheduler.getServiceState())
	//3.StopBlockSync
	service.StopBlockSync()
	time.Sleep(100 * time.Microsecond)
	//4.sync node status again
	bz = getNodeStatusResp(t, 41)
	require.NoError(t, implSync.blockSyncMsgHandler("node2", bz, netPb.NetMsg_SYNC_BLOCK_MSG))
	time.Sleep(1 * time.Second)
	require.EqualValues(t, "pendingRecvHeight: 12, peers num: 1, blockStates num: 30, pendingBlocks num: 0, receivedBlocks num: 0", implSync.scheduler.getServiceState())
}
