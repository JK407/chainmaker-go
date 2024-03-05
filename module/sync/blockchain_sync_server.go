/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sync

import (
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	commonErrors "chainmaker.org/chainmaker/common/v2/errors"
	"chainmaker.org/chainmaker/common/v2/msgbus"
	"chainmaker.org/chainmaker/localconf/v2"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	netPb "chainmaker.org/chainmaker/pb-go/v2/net"
	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
	syncPb "chainmaker.org/chainmaker/pb-go/v2/sync"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"
	"github.com/gogo/protobuf/proto"
)

var _ protocol.SyncService = (*BlockChainSyncServer)(nil)

//BlockChainSyncServer Service for synchronizing blocks
type BlockChainSyncServer struct {
	chainId string
	// receive/broadcast messages from net module
	net protocol.NetService
	// receive/broadcast messages from internal modules
	msgBus msgbus.MessageBus
	// The module that provides blocks storage/query
	blockChainStore protocol.BlockchainStore
	// Provides the latest chain state for the node
	ledgerCache protocol.LedgerCache
	// Verify Block Validity
	blockVerifier protocol.BlockVerifier
	// Adds a validated block to the chain to update the state of the chain
	blockCommitter protocol.BlockCommitter

	log protocol.Logger
	// The configuration in sync module
	conf *BlockSyncServerConf
	// Identification of module startup
	start int32
	// Identification of module close
	close chan bool
	// Service that get blocks from other nodes
	scheduler *Routine
	// Service that processes block data, adding valid blocks to the chain
	processor *Routine
	// ignore repeat block sync request when in process
	requestCache sync.Map
	//If the synced block height reaches the conf.minLabValue put a event to this channel
	minLagReachC chan struct{}
	//if a synced block is committed to local ledger put a event to this channel
	commitBlockC chan struct{}
}

//NewBlockChainSyncServer Create a new BlockChainSyncServer instance
func NewBlockChainSyncServer(
	chainId string,
	net protocol.NetService,
	msgBus msgbus.MessageBus,
	blockchainStore protocol.BlockchainStore,
	ledgerCache protocol.LedgerCache,
	blockVerifier protocol.BlockVerifier,
	blockCommitter protocol.BlockCommitter,
	log protocol.Logger) protocol.SyncService {

	syncServer := &BlockChainSyncServer{
		chainId:         chainId,
		net:             net,
		msgBus:          msgBus,
		blockChainStore: blockchainStore,
		ledgerCache:     ledgerCache,
		blockVerifier:   blockVerifier,
		blockCommitter:  blockCommitter,
		close:           make(chan bool),
		log:             log, //logger.GetLoggerByChain(logger.MODULE_SYNC, chainId),
		requestCache:    sync.Map{},
		minLagReachC:    make(chan struct{}),
		commitBlockC:    make(chan struct{}),
	}
	return syncServer
}

//Start BlockChainSyncServer preparing the required dependencies for the server to run properly
//if an error is encountered, the startup failed, please check it.
func (sync *BlockChainSyncServer) Start() error {
	if !atomic.CompareAndSwapInt32(&sync.start, 0, 1) {
		return commonErrors.ErrSyncServiceHasStarted
	}

	// 1. init conf
	sync.initSyncConfIfRequire()
	processor := newProcessor(sync, sync.ledgerCache, sync.log)
	scheduler := newScheduler(sync, sync.ledgerCache,
		sync.conf.blockPoolSize, sync.conf.timeOut,
		sync.conf.reqTimeThreshold, sync.conf.batchSizeFromOneNode,
		sync.log, sync.minLagReachC, sync.conf.minLagThreshold, sync.conf.minLagThresholdTime)
	if scheduler == nil {
		return fmt.Errorf("init scheduler failed")
	}
	sync.scheduler = NewRoutine("scheduler", scheduler.handler, scheduler.getServiceState, sync.log)
	sync.processor = NewRoutine("processor", processor.handler, processor.getServiceState, sync.log)

	// 2. register msgs handler
	if sync.msgBus != nil {
		sync.msgBus.Register(msgbus.BlockInfo, sync)
	}
	//3. register net subscribe handler
	if err := sync.net.Subscribe(netPb.NetMsg_SYNC_BLOCK_MSG, sync.blockSyncMsgHandler); err != nil {
		return err
	}
	if err := sync.net.ReceiveMsg(netPb.NetMsg_SYNC_BLOCK_MSG, sync.blockSyncMsgHandler); err != nil {
		return err
	}

	// 4. start internal service
	if err := sync.scheduler.begin(); err != nil {
		return err
	}
	if err := sync.processor.begin(); err != nil {
		return err
	}
	go sync.loop()
	go sync.blockRequestEntrance()
	return nil
}

//init sync server config
func (sync *BlockChainSyncServer) initSyncConfIfRequire() {
	defer func() {
		sync.log.Infof(sync.conf.print())
	}()
	if sync.conf != nil {
		return
	}
	sync.conf = NewBlockSyncServerConf()
	if localconf.ChainMakerConfig.SyncConfig.BlockPoolSize > 0 {
		sync.conf.SetBlockPoolSize(uint64(localconf.ChainMakerConfig.SyncConfig.BlockPoolSize))
	}
	if localconf.ChainMakerConfig.SyncConfig.WaitTimeOfBlockRequestMsg > 0 {
		sync.conf.SetWaitTimeOfBlockRequestMsg(int64(localconf.ChainMakerConfig.SyncConfig.WaitTimeOfBlockRequestMsg))
	}
	if localconf.ChainMakerConfig.SyncConfig.BatchSizeFromOneNode > 0 {
		sync.conf.SetBatchSizeFromOneNode(uint64(localconf.ChainMakerConfig.SyncConfig.BatchSizeFromOneNode))
	}
	if localconf.ChainMakerConfig.SyncConfig.LivenessTick > 0 {
		sync.conf.SetLivenessTicker(localconf.ChainMakerConfig.SyncConfig.LivenessTick)
	}
	if localconf.ChainMakerConfig.SyncConfig.NodeStatusTick > 0 {
		sync.conf.SetNodeStatusTicker(localconf.ChainMakerConfig.SyncConfig.NodeStatusTick)
	}
	if localconf.ChainMakerConfig.SyncConfig.DataDetectionTick > 0 {
		sync.conf.SetDataDetectionTicker(localconf.ChainMakerConfig.SyncConfig.DataDetectionTick)
	}
	if localconf.ChainMakerConfig.SyncConfig.ProcessBlockTick > 0 {
		sync.conf.SetProcessBlockTicker(localconf.ChainMakerConfig.SyncConfig.ProcessBlockTick)
	}
	if localconf.ChainMakerConfig.SyncConfig.SchedulerTick > 0 {
		sync.conf.SetSchedulerTicker(localconf.ChainMakerConfig.SyncConfig.SchedulerTick)
	}
	if localconf.ChainMakerConfig.SyncConfig.ReqTimeThreshold > 0 {
		sync.conf.SetReqTimeThreshold(localconf.ChainMakerConfig.SyncConfig.ReqTimeThreshold)
	}
	if localconf.ChainMakerConfig.SyncConfig.BlockRequestTime > 0 {
		sync.conf.SetBlockRequestTime(localconf.ChainMakerConfig.SyncConfig.BlockRequestTime)
	}
}

//handle messages received from the network that care about
//do the corresponding processing according to the type of the message
func (sync *BlockChainSyncServer) blockSyncMsgHandler(from string, msg []byte, msgType netPb.NetMsg_MsgType) error {
	if atomic.LoadInt32(&sync.start) != 1 {
		return commonErrors.ErrSyncServiceHasStoped
	}
	if msgType != netPb.NetMsg_SYNC_BLOCK_MSG {
		return nil
	}
	var (
		err     error
		syncMsg = syncPb.SyncMsg{}
	)
	if err = proto.Unmarshal(msg, &syncMsg); err != nil {
		sync.log.Errorf("fail to proto.Unmarshal the syncPb.SyncMsg:%s", err.Error())
		return err
	}
	sync.log.Debugf("receive the NetMsg_SYNC_BLOCK_MSG:the Type is %d", syncMsg.Type)

	switch syncMsg.Type {
	case syncPb.SyncMsg_NODE_STATUS_REQ:
		//received a request to get own state from other nodes
		return sync.handleNodeStatusReq(from)
	case syncPb.SyncMsg_NODE_STATUS_RESP:
		//received a response with peer state data from other nodes
		return sync.handleNodeStatusResp(&syncMsg, from)
	case syncPb.SyncMsg_BLOCK_SYNC_REQ:
		//received a request to sync blocks from other nodes
		return sync.handleBlockReq(&syncMsg, from)
	case syncPb.SyncMsg_BLOCK_SYNC_RESP:
		//received a response with block data from other nodes
		sync.log.Debug("receive [SyncMsg_BLOCK_SYNC_RESP] msg, put into scheduler...")
		return sync.scheduler.addTask(&SyncedBlockMsg{msg: syncMsg.Payload, from: from})
	}
	return fmt.Errorf("not support the syncPb.SyncMsg.Type as %d", syncMsg.Type)
}

//handleNodeStatusReq get own block state information and send it to where the request is from
func (sync *BlockChainSyncServer) handleNodeStatusReq(from string) error {
	var (
		height uint64
		bz     []byte
		err    error
	)
	if height, err = sync.ledgerCache.CurrentHeight(); err != nil {
		return err
	}
	archivedHeight := sync.blockChainStore.GetArchivedPivot()
	sync.log.Debugf("receive node status request from node [%s]", from)
	if bz, err = proto.Marshal(&syncPb.BlockHeightBCM{BlockHeight: height, ArchivedHeight: archivedHeight}); err != nil {
		return err
	}
	return sync.sendMsg(syncPb.SyncMsg_NODE_STATUS_RESP, bz, from)
}

//handleNodeStatusResp notify the peer's state information send by 'from' to scheduler
func (sync *BlockChainSyncServer) handleNodeStatusResp(syncMsg *syncPb.SyncMsg, from string) error {
	msg := syncPb.BlockHeightBCM{}
	if err := proto.Unmarshal(syncMsg.Payload, &msg); err != nil {
		return err
	}
	sync.log.Debugf("receive node[%s] status, height [%d], archived height [%d]", from, msg.BlockHeight,
		msg.ArchivedHeight)
	return sync.scheduler.addTask(&NodeStatusMsg{msg: msg, from: from})
}

//handleBlockReq to avoid repeated requests for the same block data in a short period of time
//cache requests using 'requestCache', key consists of the request source and block height
//value is current time
//firstly check if the request already exists in the cache，if yes, reject
//otherwise, get the corresponding block data from the local ledger and send it back
func (sync *BlockChainSyncServer) handleBlockReq(syncMsg *syncPb.SyncMsg, from string) error {
	var (
		err error
		req syncPb.BlockSyncReq
	)

	if err = proto.Unmarshal(syncMsg.Payload, &req); err != nil {
		sync.log.Errorf("fail to proto.Unmarshal the syncPb.SyncMsg:%s", err.Error())
		return err
	}
	// 针对 `SyncMsg_BLOCK_SYNC_REQ` 消息处理函数，添加处理状态检查，要求同一个 `请求来源 + 高度` 不会重复返回多次数据
	// create a key-value pair when receive block request, ignore repeat request
	processKey := fmt.Sprintf("%s_%d", from, req.BlockHeight)
	if _, loaded := sync.requestCache.LoadOrStore(processKey, time.Now()); loaded {
		sync.log.Warnf("received duplicate request to get block [height: %d, batch_size: %d] from "+
			"node [%s]", req.BlockHeight, req.BatchSize, from)
		return nil
	}

	sync.log.Infof("receive request to get block [height: %d, batch_size: %d] from "+
		"node [%s]"+"WithRwset [%v]", req.BlockHeight, req.BatchSize, from, req.WithRwset)
	return sync.sendInfos(&req, from)
}

//sendInfos send block data to 'from'.
//`req
// BlockHeight: get block data starting from the this height
// BatchSize: the number of blocks to be acquired at one time
// WithRwset: the block data has a read-write set or not
//`
//we should get block data whose height is in [req.BlockHeight, req.BlockHeight+req.BatchSize)
//the request height may exceed the block height in the local ledger
//if so, blockChainStore will return a nil block without a error, need to skip it instead of sending it
//since a block data will be large, we send it one by one instead of all at once
//to reduce errors during network transmission.
func (sync *BlockChainSyncServer) sendInfos(req *syncPb.BlockSyncReq, from string) error {
	var (
		bz        []byte
		err       error
		blk       *commonPb.Block
		blkRwInfo *storePb.BlockWithRWSet
	)
	for i := uint64(0); i < req.BatchSize; i++ {
		if req.WithRwset {
			if blkRwInfo, err = sync.blockChainStore.GetBlockWithRWSets(req.BlockHeight + i); err != nil {
				return err
			}
			if blkRwInfo == nil {
				sync.log.Warnf("GetBlockWithRWSets get block height: [%d] is nil", req.BlockHeight+i)
				continue
			}
		} else {
			if blk, err = sync.blockChainStore.GetBlock(req.BlockHeight + i); err != nil {
				sync.log.Debugf("[SyncMsg_BLOCK_SYNC_RESP] get block without reset with err: %s", err.Error())
				return err
			}
			if blk == nil {
				sync.log.Warnf("GetBlock get block height: [%d] is nil", req.BlockHeight+i)
				continue
			}
			blkRwInfo = &storePb.BlockWithRWSet{
				Block:    blk,
				TxRWSets: nil,
			}
		}
		info := &commonPb.BlockInfo{Block: blkRwInfo.Block, RwsetList: blkRwInfo.TxRWSets}
		if bz, err = proto.Marshal(&syncPb.SyncBlockBatch{
			Data: &syncPb.SyncBlockBatch_BlockinfoBatch{BlockinfoBatch: &syncPb.BlockInfoBatch{
				Batch: []*commonPb.BlockInfo{info}}}, WithRwset: req.WithRwset,
		}); err != nil {
			return err
		}
		if err := sync.sendMsg(syncPb.SyncMsg_BLOCK_SYNC_RESP, bz, from); err != nil {
			return err
		}
	}
	return nil
}

func (sync *BlockChainSyncServer) sendMsg(msgType syncPb.SyncMsg_MsgType, msg []byte, to string) error {
	var (
		bs  []byte
		err error
	)
	if bs, err = proto.Marshal(&syncPb.SyncMsg{
		Type:    msgType,
		Payload: msg,
	}); err != nil {
		sync.log.Error(err)
		return err
	}
	if err = sync.net.SendMsg(bs, netPb.NetMsg_SYNC_BLOCK_MSG, to); err != nil {
		sync.log.Warnf("send [%s] message to [%s] error: %v", netPb.NetMsg_SYNC_BLOCK_MSG.String(), to, err)
		return err
	}
	return nil
}

//broadcastMsg broadcast messages to other nodes
func (sync *BlockChainSyncServer) broadcastMsg(msgType syncPb.SyncMsg_MsgType, msg []byte) error {
	var (
		bs  []byte
		err error
	)
	if bs, err = proto.Marshal(&syncPb.SyncMsg{
		Type:    msgType,
		Payload: msg,
	}); err != nil {
		sync.log.Error(err)
		return err
	}
	if err = sync.net.BroadcastMsg(bs, netPb.NetMsg_SYNC_BLOCK_MSG); err != nil {
		sync.log.Error(err)
		return err
	}
	return nil
}

//synchronization service advance workflow in an event-driven way
//some events are driven by timers
//some events are emitted from worker modules
//loop waiting for an event to happen and do the corresponding processing
func (sync *BlockChainSyncServer) loop() {
	var (
		// task: trigger the flow of the block process
		doProcessBlockTk = time.NewTicker(sync.conf.processBlockTick)
		// task: trigger the state acquisition process for the node
		doScheduleTk = time.NewTicker(sync.conf.schedulerTick)
		// task: trigger the flow of the node status acquisition from connected peers
		doNodeStatusTk = time.NewTicker(sync.conf.nodeStatusTick)
		// task: trigger the check of the liveness with connected peers
		doLivenessTk = time.NewTicker(sync.conf.livenessTick)
		// task: trigger the check of the data in processor and scheduler
		doDataDetect = time.NewTicker(sync.conf.dataDetectionTick)
	)
	defer func() {
		doProcessBlockTk.Stop()
		doScheduleTk.Stop()
		doLivenessTk.Stop()
		doNodeStatusTk.Stop()
		doDataDetect.Stop()
	}()
	_ = sync.broadcastMsg(syncPb.SyncMsg_NODE_STATUS_REQ, nil)
	for {
		select {
		//service close
		case <-sync.close:
			return

			// timing drives the processor to process block data
		case <-doProcessBlockTk.C:
			if err := sync.processor.addTask(&ProcessBlockMsg{}); err != nil {
				sync.log.Errorf("add process block task to processor failed, reason: %s", err)
			}
			// timing drives the scheduler to schedule the sending of a synchronized block request
		case <-doScheduleTk.C:
			if err := sync.scheduler.addTask(&SchedulerMsg{}); err != nil {
				sync.log.Errorf("add scheduler task to scheduler failed, reason: %s", err)
			}
			// timing drives to do live detection
		case <-doLivenessTk.C:
			if err := sync.scheduler.addTask(&LivenessMsg{}); err != nil {
				sync.log.Errorf("add livenessMsg task to scheduler failed, reason: %s", err)
			}
			// timing drives to request to get the status of other nodes
		case <-doNodeStatusTk.C:
			sync.log.Debugf("broadcast request of the node status")
			if err := sync.broadcastMsg(syncPb.SyncMsg_NODE_STATUS_REQ, nil); err != nil {
				sync.log.Errorf("request node status failed by broadcast", err)
			}
			// timing drives to do data validity check
		case <-doDataDetect.C:
			if err := sync.processor.addTask(&DataDetection{}); err != nil {
				sync.log.Errorf("add data detection task to processor failed, reason: %s", err)
			}
			if err := sync.scheduler.addTask(&DataDetection{}); err != nil {
				sync.log.Errorf("add data detection task to scheduler failed, reason: %s", err)
			}

		// State processing results in state machine
		//send the result obtained from scheduler to processor for processing
		case resp := <-sync.scheduler.out:
			sync.log.Debugf("sync.processor add task, type: %v", reflect.TypeOf(resp))
			if err := sync.processor.addTask(resp); err != nil {
				sync.log.Errorf("add scheduler task to processor failed, reason: %s", err)
			}
			//send the result obtained from processor to scheduler for processing
		case resp := <-sync.processor.out:
			sync.log.Debugf("sync.scheduler add task, type: %v", reflect.TypeOf(resp))
			if err := sync.scheduler.addTask(resp); err != nil {
				sync.log.Errorf("add processor task to scheduler failed, reason: %s", err)
			}
		}
	}
}

// auto check block request from other node
//regularly check whether the cached request information has expired
//if it expires, remove it from the cache
func (sync *BlockChainSyncServer) blockRequestEntrance() {
	ticker := time.NewTicker(sync.conf.blockRequestTime)
	dealFunc := func(key, value interface{}) bool {
		if value == nil {
			return true
		}
		if t, ok := value.(time.Time); ok {
			if time.Since(t) > sync.conf.blockRequestTime {
				sync.requestCache.Delete(key)
			}
			return true
		}
		return true
	}
	for {
		select {
		case <-sync.close:
			return

		case <-ticker.C:
			sync.requestCache.Range(dealFunc)
		}
	}
}

// ListenSyncToIdealHeight listen local block height has synced to ideal height
func (sync *BlockChainSyncServer) ListenSyncToIdealHeight() <-chan struct{} {
	return sync.minLagReachC
}

//verify and submit the block data does not carry the read-write set
//different types of status are returned depending on the stage in which the error occurred
//verify failed then return validateFailed status
//commit failed if block has been committed return hasProcessed, if not return addErr
//all succeeded return ok
func (sync *BlockChainSyncServer) validateAndCommitBlock(block *commonPb.Block) processedBlockStatus {
	if blk := sync.ledgerCache.GetLastCommittedBlock(); blk != nil && blk.Header.BlockHeight >= block.Header.BlockHeight {
		sync.log.Infof("the block: %d has been committed in the blockChainStore ", block.Header.BlockHeight)
		return hasProcessed
	}
	startTick := utils.CurrentTimeMillisSeconds()
	sync.log.Debugf("VerifyBlock start, height is: %d ....", block.Header.BlockHeight)
	if err := sync.blockVerifier.VerifyBlock(block, protocol.SYNC_VERIFY); err != nil {
		if err == commonErrors.ErrBlockHadBeenCommited {
			sync.log.Warnf("the block: %d has been committed in the blockChainStore ", block.Header.BlockHeight)
			return hasProcessed
		}
		sync.log.Warnf("fail to verify the block whose height is %d, err: %s", block.Header.BlockHeight, err)
		return validateFailed
	}
	lastTime := utils.CurrentTimeMillisSeconds() - startTick
	sync.log.Infof("block [%d] VerifyBlock spend %d", block.Header.BlockHeight, lastTime)
	if err := sync.blockCommitter.AddBlock(block); err != nil {
		if err == commonErrors.ErrBlockHadBeenCommited {
			sync.log.Warnf("the block: %d has been committed in the blockChainStore ", block.Header.BlockHeight)
			return hasProcessed
		}
		sync.log.Warnf("fail to commit the block whose height is %d, err: %s", block.Header.BlockHeight, err)
		return addErr
	}
	return ok
}

//verify and submit the block data carries the read-write set
//different types of status are returned depending on the stage in which the error occurred
//verify failed then return validateFailed status
//commit failed if block has been committed return hasProcessed, if not return addErr
//all succeeded return ok
func (sync *BlockChainSyncServer) validateAndCommitBlockWithRwSets(block *commonPb.Block,
	rwsets []*commonPb.TxRWSet) processedBlockStatus {
	//if the height of the local ledger is not lower than this block height
	//indicates that the block has been processed
	if blk := sync.ledgerCache.GetLastCommittedBlock(); blk != nil && blk.Header.BlockHeight >= block.Header.BlockHeight {
		sync.log.Infof("the block: %d has been committed in the blockChainStore ", block.Header.BlockHeight)
		return hasProcessed
	}
	startTick := utils.CurrentTimeMillisSeconds()
	if err := sync.blockVerifier.VerifyBlockWithRwSets(block, rwsets, protocol.SYNC_VERIFY); err != nil {
		if err == commonErrors.ErrBlockHadBeenCommited {
			sync.log.Warnf("the block: %d has been committed in the blockChainStore ", block.Header.BlockHeight)
			return hasProcessed
		}
		sync.log.Warnf("fail to verify the block with Rwset whose height is %d, err: %s", block.Header.BlockHeight, err)
		return validateFailed
	}
	lastTime := utils.CurrentTimeMillisSeconds() - startTick
	sync.log.Infof("block [%d] VerifyBlockWithRwSets spend %d", block.Header.BlockHeight, lastTime)

	sync.log.Debugf("VerifyBlock end, height is: %d ....", block.Header.BlockHeight)
	sync.log.Debugf("AddBlock start, height is: %d ....", block.Header.BlockHeight)
	if err := sync.blockCommitter.AddBlock(block); err != nil {
		if err == commonErrors.ErrBlockHadBeenCommited {
			sync.log.Warnf("the block: %d has been committed in the blockChainStore ", block.Header.BlockHeight)
			return hasProcessed
		}
		sync.log.Warnf("fail to commit the block whose height is %d, err: %s", block.Header.BlockHeight, err)
		return addErr
	}
	sync.log.Debugf("AddBlock end, height is: %d ....", block.Header.BlockHeight)
	return ok
}

//StopBlockSync make sync service stop sending sync block requests to other peer nodes
func (sync *BlockChainSyncServer) StopBlockSync() {
	_ = sync.scheduler.addTask(&StopSyncMsg{})
	_ = sync.processor.addTask(&StopSyncMsg{})
}

//StartBlockSync make sync service resume sending sync block requests to other peer nodes
func (sync *BlockChainSyncServer) StartBlockSync() {
	_ = sync.scheduler.addTask(&StartSyncMsg{})
}

//Stop stop sync service all work
func (sync *BlockChainSyncServer) Stop() {
	if !atomic.CompareAndSwapInt32(&sync.start, 1, 0) {
		return
	}
	sync.scheduler.end()
	sync.processor.end()
	close(sync.close)
}

//OnMessage msgbus Subscriber interface implementation
//used to receive block notifications from msgbus, if the height of the newly received block is divisible by 3,
//then broadcast own block height to other peers
func (sync *BlockChainSyncServer) OnMessage(message *msgbus.Message) {
	if message == nil || message.Payload == nil {
		sync.log.Errorf("receive the empty message")
		return
	}
	if message.Topic != msgbus.BlockInfo {
		sync.log.Errorf("receive the message from the topic as %d, but not msgbus.BlockInfo ", message.Topic)
		return
	}
	switch blockInfo := message.Payload.(type) {
	case *commonPb.BlockInfo:
		if blockInfo == nil || blockInfo.Block == nil {
			sync.log.Errorf("error message BlockInfo = nil")
			return
		}
		height := blockInfo.Block.Header.BlockHeight
		if height%3 != 0 {
			return
		}
		bz, err := proto.Marshal(&syncPb.BlockHeightBCM{BlockHeight: height})
		if err != nil {
			sync.log.Errorf("marshal BlockHeightBCM failed, reason: %s", err)
			return
		}
		if err := sync.broadcastMsg(syncPb.SyncMsg_NODE_STATUS_RESP, bz); err != nil {
			sync.log.Errorf("fail to broadcast the height as %d, and the error is %s", height, err)
		}
	default:
		sync.log.Errorf("not support the message type as %T", message.Payload)
	}
}

//OnQuit msgbus Subscriber interface implementation
func (sync *BlockChainSyncServer) OnQuit() {
	sync.log.Infof("stop to listen the msgbus.BlockInfo")
}
