/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blockchain

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"chainmaker.org/chainmaker-go/module/net"
	"chainmaker.org/chainmaker-go/module/subscriber"
	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	"chainmaker.org/chainmaker/common/v2/crypto/engine"
	"chainmaker.org/chainmaker/common/v2/helper"
	"chainmaker.org/chainmaker/common/v2/msgbus"
	localconf "chainmaker.org/chainmaker/localconf/v2"
	logger "chainmaker.org/chainmaker/logger/v2"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/txpool"
	protocol "chainmaker.org/chainmaker/protocol/v2"
)

var log = logger.GetLogger(logger.MODULE_BLOCKCHAIN)

const chainIdNotFoundErrorTemplate = "chain id %s not found"

// ChainMakerServer manage all blockchains
type ChainMakerServer struct {
	// net shared by all chains
	net protocol.Net

	// blockchains known by this node
	blockchains sync.Map // map[string]*Blockchain

	readyC chan struct{}
}

// NewChainMakerServer create a new ChainMakerServer instance.
func NewChainMakerServer() *ChainMakerServer {
	return &ChainMakerServer{}
}

// Init ChainMakerServer.
func (server *ChainMakerServer) Init() error {
	var err error
	log.Debug("begin init chain maker server...")
	server.readyC = make(chan struct{})
	// 1) init net
	if err = server.initNet(); err != nil {
		return err
	}
	// 2) init blockchains
	if err = server.initBlockchains(); err != nil {
		return err
	}
	log.Info("init chain maker server success!")
	return nil
}

// InitForRebuildDbs init ChainMakerServer.
func (server *ChainMakerServer) InitForRebuildDbs(chainId string) error {
	var err error
	log.Debug("begin init chain maker rebuild dbs server...")
	server.readyC = make(chan struct{})
	// 1) init net
	//if err = server.initNet(); err != nil {
	//	return err
	//}
	// 2) init blockchains
	if err = server.initBlockchainsForRebuildDbs(chainId); err != nil {
		return err
	}
	log.Info("init chain maker server success!")
	return nil
}
func (server *ChainMakerServer) initNet() error {
	var netType protocol.NetType
	var err error
	// load net type
	provider := localconf.ChainMakerConfig.NetConfig.Provider
	log.Infof("load net provider: %s", provider)
	switch strings.ToLower(provider) {
	case "libp2p":
		netType = protocol.Libp2p

	case "liquid":
		netType = protocol.Liquid
	default:
		return errors.New("unsupported net provider")
	}

	authType := localconf.ChainMakerConfig.AuthType
	emptyAuthType := ""

	// load tls keys and cert path
	keyPath := localconf.ChainMakerConfig.NetConfig.TLSConfig.PrivKeyFile
	if !filepath.IsAbs(keyPath) {
		keyPath, err = filepath.Abs(keyPath)
		if err != nil {
			return err
		}
	}
	log.Infof("load net tls key file path: %s", keyPath)

	var certPath string
	var pubKeyMode bool
	switch strings.ToLower(authType) {
	case protocol.PermissionedWithKey, protocol.Public:
		pubKeyMode = true
	case protocol.PermissionedWithCert, protocol.Identity, emptyAuthType:
		pubKeyMode = false
		certPath = localconf.ChainMakerConfig.NetConfig.TLSConfig.CertFile
		if !filepath.IsAbs(certPath) {
			certPath, err = filepath.Abs(certPath)
			if err != nil {
				return err
			}
		}
		log.Infof("load net tls cert file path: %s", certPath)
	default:
		return errors.New("wrong auth type")
	}
	//gmtls enc key/cert
	encKeyPath, _ := filepath.Abs(localconf.ChainMakerConfig.NetConfig.TLSConfig.PrivEncKeyFile)
	encCertPath, _ := filepath.Abs(localconf.ChainMakerConfig.NetConfig.TLSConfig.CertEncFile)

	// new net
	var netFactory net.NetFactory
	server.net, err = netFactory.NewNet(
		netType,
		net.WithReadySignalC(server.readyC),
		net.WithListenAddr(localconf.ChainMakerConfig.NetConfig.ListenAddr),
		net.WithCrypto(pubKeyMode, keyPath, certPath, encKeyPath, encCertPath),
		net.WithPeerStreamPoolSize(localconf.ChainMakerConfig.NetConfig.PeerStreamPoolSize),
		net.WithMaxPeerCountAllowed(localconf.ChainMakerConfig.NetConfig.MaxPeerCountAllow),
		net.WithPeerEliminationStrategy(localconf.ChainMakerConfig.NetConfig.PeerEliminationStrategy),
		net.WithSeeds(localconf.ChainMakerConfig.NetConfig.Seeds...),
		net.WithBlackAddresses(localconf.ChainMakerConfig.NetConfig.BlackList.Addresses...),
		net.WithBlackNodeIds(localconf.ChainMakerConfig.NetConfig.BlackList.NodeIds...),
		net.WithMsgCompression(localconf.ChainMakerConfig.DebugConfig.UseNetMsgCompression),
		net.WithInsecurity(localconf.ChainMakerConfig.DebugConfig.IsNetInsecurity),
		net.WithStunClient(localconf.ChainMakerConfig.NetConfig.StunClient.ListenAddr,
			localconf.ChainMakerConfig.NetConfig.StunClient.StunServerAddr,
			localconf.ChainMakerConfig.NetConfig.StunClient.NetworkType,
			localconf.ChainMakerConfig.NetConfig.StunClient.Enabled),
		net.WithStunServer(localconf.ChainMakerConfig.NetConfig.StunServer.Enabled,
			localconf.ChainMakerConfig.NetConfig.StunServer.TwoPublicAddress,
			localconf.ChainMakerConfig.NetConfig.StunServer.OtherStunServerAddr,
			localconf.ChainMakerConfig.NetConfig.StunServer.LocalNotifyAddr,
			localconf.ChainMakerConfig.NetConfig.StunServer.OtherNotifyAddr,
			localconf.ChainMakerConfig.NetConfig.StunServer.ListenAddr1,
			localconf.ChainMakerConfig.NetConfig.StunServer.ListenAddr2,
			localconf.ChainMakerConfig.NetConfig.StunServer.ListenAddr3,
			localconf.ChainMakerConfig.NetConfig.StunServer.ListenAddr4,
			localconf.ChainMakerConfig.NetConfig.StunServer.NetworkType),
		net.WithHolePunch(localconf.ChainMakerConfig.NetConfig.EnablePunch),
	)
	if err != nil {
		errMsg := fmt.Sprintf("new net failed, %s", err.Error())
		log.Error(errMsg)
		return errors.New(errMsg)
	}

	// read key file, then set the NodeId of local config
	file, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return err
	}
	privateKey, err := asym.PrivateKeyFromPEM(file, nil)
	if err != nil {
		return err
	}
	nodeId, err := helper.CreateLibp2pPeerIdWithPrivateKey(privateKey)
	if err != nil {
		return err
	}
	localconf.ChainMakerConfig.SetNodeId(nodeId)

	// load custom chain trust roots
	for _, chainTrustRoots := range localconf.ChainMakerConfig.NetConfig.CustomChainTrustRoots {
		roots := make([][]byte, 0, len(chainTrustRoots.TrustRoots))
		for _, r := range chainTrustRoots.TrustRoots {
			rootBytes, err2 := ioutil.ReadFile(r.Root)
			if err2 != nil {
				log.Errorf("load custom chain trust roots failed, %s", err2.Error())
				return err2
			}
			roots = append(roots, rootBytes)
		}
		server.net.SetChainCustomTrustRoots(chainTrustRoots.ChainId, roots)
		log.Infof("set custom trust roots for chain[%s] success.", chainTrustRoots.ChainId)
	}
	return nil
}

func (server *ChainMakerServer) initBlockchains() error {
	server.blockchains = sync.Map{}
	ok := false
	for _, chain := range localconf.ChainMakerConfig.GetBlockChains() {
		chainId := chain.ChainId
		if err := server.initBlockchain(chainId, chain.Genesis); err != nil {
			log.Error(err.Error())
			continue
		}
		ok = true
	}
	if !ok {
		return fmt.Errorf("init all blockchains fail")
	}
	go server.newBlockchainTaskListener()
	go server.deleteBlockchainTaskListener()
	return nil
}

func (server *ChainMakerServer) initBlockchainsForRebuildDbs(chainId string) error {
	server.blockchains = sync.Map{}
	ok := false
	for _, chain := range localconf.ChainMakerConfig.GetBlockChains() {
		if chainId == chain.ChainId {
			if err := server.initBlockchainForRebuildDbs(chainId, chain.Genesis); err != nil {
				return err
			}
			ok = true
		}
		//if err := server.initBlockchainForRebuildDbs(chainId, chain.Genesis); err != nil {
		//	log.Error(err.Error())
		//	continue
		//}
	}
	if !ok {
		return fmt.Errorf("init %s blockchains fail for not exists", chainId)
	}
	go server.newBlockchainTaskListener()
	go server.deleteBlockchainTaskListener()
	return nil
}

func (server *ChainMakerServer) newBlockchainTaskListener() {
	for newChainId := range localconf.FindNewBlockChainNotifyC {
		_, ok := server.blockchains.Load(newChainId)
		if ok {
			log.Errorf("new block chain found existed(chain-id: %s)", newChainId)
			continue
		}
		log.Infof("new block chain found(chain-id: %s), start to init new block chain.", newChainId)
		for _, chain := range localconf.ChainMakerConfig.GetBlockChains() {
			if chain.ChainId == newChainId {
				if err := server.initBlockchain(newChainId, chain.Genesis); err != nil {
					log.Error(err.Error())
					continue
				}
				newBlockchain, _ := server.blockchains.Load(newChainId)
				go startBlockchain(newBlockchain.(*Blockchain))
			}
		}
	}
}

func (server *ChainMakerServer) deleteBlockchainTaskListener() {
	for deleteChainId := range localconf.FindDeleteBlockChainNotifyC {
		oldBlockChain, ok := server.blockchains.Load(deleteChainId)
		if !ok {
			log.Errorf("old block chain not found (chain-id: %s)", deleteChainId)
			continue
		}
		log.Infof("old block chain found(chain-id: %s), start to delete block chain.", deleteChainId)
		oldBlockChainS, _ := oldBlockChain.(*Blockchain)
		oldBlockChainS.Stop()
		server.blockchains.Delete(deleteChainId)
	}
}

func (server *ChainMakerServer) initBlockchain(chainId, genesis string) error {
	if !filepath.IsAbs(genesis) {
		var err error
		genesis, err = filepath.Abs(genesis)
		if err != nil {
			return err
		}
	}
	log.Infof("load genesis file path of chain[%s]: %s", chainId, genesis)
	blockchain := NewBlockchain(genesis, chainId, msgbus.NewMessageBus(), server.net)

	if err := blockchain.Init(); err != nil {
		errMsg := fmt.Sprintf("init blockchain[%s] failed, %s", chainId, err.Error())
		return errors.New(errMsg)
	}
	server.blockchains.Store(chainId, blockchain)
	log.Infof("init blockchain[%s] success!", chainId)
	return nil
}

func (server *ChainMakerServer) initBlockchainForRebuildDbs(chainId, genesis string) error {
	if !filepath.IsAbs(genesis) {
		var err error
		genesis, err = filepath.Abs(genesis)
		if err != nil {
			return err
		}
	}
	log.Infof("load genesis file path of chain[%s]: %s", chainId, genesis)
	blockchain := NewBlockchain(genesis, chainId, msgbus.NewMessageBus(), server.net)
	if err := blockchain.InitForRebuildDbs(); err != nil {
		errMsg := fmt.Sprintf("init blockchain[%s] failed, %s", chainId, err.Error())
		return errors.New(errMsg)
	}
	server.blockchains.Store(chainId, blockchain)
	log.Infof("init blockchain[%s] success!", chainId)
	return nil
}
func startBlockchain(chain *Blockchain) {
	if err := chain.Start(); err != nil {
		log.Errorf("[Core] start blockchain[%s] failed, %s", chain.chainId, err.Error())
		os.Exit(-1)
	}
	log.Infof("[Core] start blockchain[%s] success", chain.chainId)
}
func startBlockchainForRebuildDbs(chain *Blockchain, needVerify bool) {
	if err := chain.StartForRebuildDbs(); err != nil {
		log.Errorf("[Core] start blockchain[%s] rebuild-dbs failed, %s", chain.chainId, err.Error())
		os.Exit(-1)
	}
	log.Infof("[Core] start blockchain[%s] rebuild-dbs success", chain.chainId)
	chain.RebuildDbs(needVerify)
}

// Start ChainMakerServer.
func (server *ChainMakerServer) Start() error {
	// 1) start Net
	if err := server.net.Start(); err != nil {
		log.Errorf("[Net] start failed, %s", err.Error())
		return err
	}
	log.Infof("[Net] start success!")

	//init crypto engine for ac
	engine.InitCryptoEngine(localconf.ChainMakerConfig.CryptoEngine, false)

	// 2) start blockchains
	server.blockchains.Range(func(_, value interface{}) bool {
		chain, _ := value.(*Blockchain)
		go startBlockchain(chain)
		return true
	})
	// 3) ready
	close(server.readyC)
	return nil
}

// Start ChainMakerServer for rebuild dbs.
func (server *ChainMakerServer) StartForRebuildDbs(needVerify bool) error {
	// 1) start Net
	//if err := server.net.Start(); err != nil {
	//	log.Errorf("[Net] start failed, %s", err.Error())
	//	return err
	//}
	//log.Infof("[Net] start success!")
	// 2) start blockchains
	server.blockchains.Range(func(_, value interface{}) bool {
		chain, _ := value.(*Blockchain)
		go startBlockchainForRebuildDbs(chain, needVerify)
		return true
	})

	// 3) ready
	close(server.readyC)
	return nil
}

// Stop ChainMakerServer.
func (server *ChainMakerServer) Stop() {
	// stop all blockchains
	var wg sync.WaitGroup
	server.blockchains.Range(func(_, value interface{}) bool {
		chain, _ := value.(*Blockchain)
		wg.Add(1)
		go func(chain *Blockchain) {
			defer wg.Done()
			chain.Stop()
		}(chain)
		return true
	})
	wg.Wait()
	log.Info("ChainMaker server is stopped!")

	// stop net
	if err := server.net.Stop(); err != nil {
		log.Errorf("stop net failed, %s", err.Error())
	}
	log.Info("net is stopped!")

}

// AddTx add a transaction.
func (server *ChainMakerServer) AddTx(chainId string, tx *common.Transaction, source protocol.TxSource) error {
	if blockchain, ok := server.blockchains.Load(chainId); ok {
		return blockchain.(*Blockchain).txPool.AddTx(tx, source)
	}
	return fmt.Errorf(chainIdNotFoundErrorTemplate, chainId)
}

// GetPoolStatus Returns the max size of config transaction pool and common transaction pool,
// the num of config transaction in queue and pendingCache,
// and the the num of common transaction in queue and pendingCache.
func (server *ChainMakerServer) GetPoolStatus(chainId string) (*txpool.TxPoolStatus, error) {
	if blockchain, ok := server.blockchains.Load(chainId); ok {
		return blockchain.(*Blockchain).txPool.GetPoolStatus(), nil
	}
	return nil, fmt.Errorf(chainIdNotFoundErrorTemplate, chainId)
}

// GetTxIdsByTypeAndStage Returns config or common txIds in different stage.
// txType may be TxType_CONFIG_TX, TxType_COMMON_TX, (TxType_CONFIG_TX|TxType_COMMON_TX)
// txStage may be TxStage_IN_QUEUE, TxStage_IN_PENDING, (TxStage_IN_QUEUE|TxStage_IN_PENDING)
func (server *ChainMakerServer) GetTxIdsByTypeAndStage(chainId string, txType, txStage int32) ([]string, error) {
	if blockchain, ok := server.blockchains.Load(chainId); ok {
		return blockchain.(*Blockchain).txPool.GetTxIdsByTypeAndStage(txType, txStage), nil
	}
	return nil, fmt.Errorf(chainIdNotFoundErrorTemplate, chainId)
}

// GetTxsInPoolByTxIds Retrieve the transactions by the txIds from the txPool,
// return transactions in the txPool and txIds not in txPool.
// default query upper limit is 1w transaction, and error is returned if the limit is exceeded.
func (server *ChainMakerServer) GetTxsInPoolByTxIds(chainId string,
	txIds []string) ([]*common.Transaction, []string, error) {
	if blockchain, ok := server.blockchains.Load(chainId); ok {
		return blockchain.(*Blockchain).txPool.GetTxsInPoolByTxIds(txIds)
	}
	return nil, nil, fmt.Errorf(chainIdNotFoundErrorTemplate, chainId)
}

// GetStore get the store instance of chain which id is the given.
func (server *ChainMakerServer) GetStore(chainId string) (protocol.BlockchainStore, error) {
	if blockchain, ok := server.blockchains.Load(chainId); ok {
		return blockchain.(*Blockchain).store, nil
	}

	return nil, fmt.Errorf(chainIdNotFoundErrorTemplate, chainId)
}

// GetChainConf get protocol.ChainConf of chain which id is the given.
func (server *ChainMakerServer) GetChainConf(chainId string) (protocol.ChainConf, error) {
	if blockchain, ok := server.blockchains.Load(chainId); ok {
		return blockchain.(*Blockchain).chainConf, nil
	}

	return nil, fmt.Errorf(chainIdNotFoundErrorTemplate, chainId)
}

// GetAllChainConf get all protocol.ChainConf of all the chains.
func (server *ChainMakerServer) GetAllChainConf() ([]protocol.ChainConf, error) {
	var chainConfs []protocol.ChainConf
	server.blockchains.Range(func(_, value interface{}) bool {
		blockchain, _ := value.(*Blockchain)
		chainConfs = append(chainConfs, blockchain.chainConf)
		return true
	})

	if len(chainConfs) == 0 {
		return nil, fmt.Errorf("all chain not found")
	}

	return chainConfs, nil
}

// GetVmManager get protocol.VmManager of chain which id is the given.
func (server *ChainMakerServer) GetVmManager(chainId string) (protocol.VmManager, error) {
	if blockchain, ok := server.blockchains.Load(chainId); ok {
		return blockchain.(*Blockchain).vmMgr, nil
	}

	return nil, fmt.Errorf(chainIdNotFoundErrorTemplate, chainId)
}

// GetEventSubscribe get subscriber.EventSubscriber of chain which id is the given.
func (server *ChainMakerServer) GetEventSubscribe(chainId string) (*subscriber.EventSubscriber, error) {
	if blockchain, ok := server.blockchains.Load(chainId); ok {
		return blockchain.(*Blockchain).eventSubscriber, nil
	}

	return nil, fmt.Errorf(chainIdNotFoundErrorTemplate, chainId)
}

// GetNetService get protocol.NetService of chain which id is the given.
func (server *ChainMakerServer) GetNetService(chainId string) (protocol.NetService, error) {
	if blockchain, ok := server.blockchains.Load(chainId); ok {
		return blockchain.(*Blockchain).netService, nil
	}

	return nil, fmt.Errorf(chainIdNotFoundErrorTemplate, chainId)
}

// GetBlockchain get Blockchain of chain which id is the given.
func (server *ChainMakerServer) GetBlockchain(chainId string) (*Blockchain, error) {
	if blockchain, ok := server.blockchains.Load(chainId); ok {
		return blockchain.(*Blockchain), nil
	}

	return nil, fmt.Errorf(chainIdNotFoundErrorTemplate, chainId)
}

// GetAllAC get all protocol.AccessControlProvider of all the chains.
func (server *ChainMakerServer) GetAllAC() ([]protocol.AccessControlProvider, error) {
	var accessControls []protocol.AccessControlProvider
	server.blockchains.Range(func(_, value interface{}) bool {
		blockchain, ok := value.(*Blockchain)
		if !ok {
			panic("invalid blockchain obj")
		}
		accessControls = append(accessControls, blockchain.GetAccessControl())
		return true
	})

	if len(accessControls) == 0 {
		return nil, fmt.Errorf("all chain not found")
	}

	return accessControls, nil
}

// Version of chainmaker.
func (server *ChainMakerServer) Version() string {
	return CurrentVersion
}
