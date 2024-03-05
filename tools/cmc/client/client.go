/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package client operations about chain maker client can do
package client

import (
	"fmt"

	"chainmaker.org/chainmaker-go/tools/cmc/util"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	// 压测参数
	concurrency          int // 并发数
	totalCntPerGoroutine int // 每个并发协程请求数

	sdkConfPath string // SDK配置路径

	// 合约参数
	abiFilePath      string
	contractName     string
	contractAddress  string
	version          string
	byteCodePath     string
	runtimeType      string
	timeout          int64
	sendTimes        int
	method           string
	params           string
	orgId            string
	chainId          string
	syncResult       bool
	enableCertHash   bool
	blockHeight      uint64
	withRWSet        bool
	truncateValue    bool
	truncateValueLen string
	truncateModel    string
	isAgree          bool
	txId             string

	adminKeyFilePaths string
	adminCrtFilePaths string
	adminOrgIds       string

	payerKeyFilePath string
	payerCrtFilePath string
	payerOrgId       string

	userTlsKeyFilePath  string
	userTlsCrtFilePath  string
	userSignKeyFilePath string
	userSignCrtFilePath string

	blockInterval    uint32
	extraConfigKey   string
	extraConfigValue string
	txParameterSize  uint32
	nodeOrgId        string
	nodeIdOld        string
	nodeId           string
	nodeIds          string
	trustRootOrgId   string
	trustRootPaths   []string
	certFilePaths    string
	certCrlPath      string

	address   string
	amount    string
	delegator string
	validator string
	epochID   string

	grantContractList  []string
	revokeContractList []string

	trustMemberOrgId    string
	trustMemberInfoPath string
	trustMemberRole     string
	trustMemberNodeId   string

	gasLimit  uint64
	gasEnable bool

	addressType                      int32
	permissionResourceName           string
	permissionResourcePolicyRule     string
	permissionResourcePolicyOrgList  []string
	permissionResourcePolicyRoleList []string
	respResultToString               bool

	multiSignEnableManualRun bool
)

const (
	flagConcurrency                      = "concurrency"
	flagTotalCountPerGoroutine           = "total-count-per-goroutine"
	flagSdkConfPath                      = "sdk-conf-path"
	flagAbiFilePath                      = "abi-file-path"
	flagContractName                     = "contract-name"
	flagContractAddress                  = "contract-address"
	flagVersion                          = "version"
	flagMethod                           = "method"
	flagParams                           = "params"
	flagOrgId                            = "org-id"
	flagSyncResult                       = "sync-result"
	flagEnableCertHash                   = "enable-cert-hash"
	flagBlockHeight                      = "block-height"
	flagWithRWSet                        = "with-rw-set"
	flagTruncateModel                    = "truncate-model"
	flagTruncateValueLen                 = "truncate-value-len"
	flagTruncateValue                    = "truncate-value"
	flagIsAgree                          = "is-agree"
	flagTxId                             = "tx-id"
	flagByteCodePath                     = "byte-code-path"
	flagRuntimeType                      = "runtime-type"
	flagChainId                          = "chain-id"
	flagSendTimes                        = "send-times"
	flagAdminKeyFilePaths                = "admin-key-file-paths"
	flagAdminCrtFilePaths                = "admin-crt-file-paths"
	flagAdminOrgIds                      = "admin-org-ids"
	flagPayerKeyFilePath                 = "payer-key-file-path"
	flagPayerCrtFilePath                 = "payer-crt-file-path"
	flagPayerOrgId                       = "payer-org-id"
	flagUserTlsKeyFilePath               = "user-tlskey-file-path"
	flagUserTlsCrtFilePath               = "user-tlscrt-file-path"
	flagUserSignKeyFilePath              = "user-signkey-file-path"
	flagUserSignCrtFilePath              = "user-signcrt-file-path"
	flagTimeout                          = "timeout"
	flagBlockInterval                    = "block-interval"
	flagExtraConfigKey                   = "extra-config-key"
	flagExtraConfigValue                 = "extra-config-value"
	flagTxParameterSize                  = "tx-parameter-size"
	flagNodeOrgId                        = "node-org-id"
	flagNodeIdOld                        = "node-id-old"
	flagNodeId                           = "node-id"
	flagNodeIds                          = "node-ids"
	flagTrustRootOrgId                   = "trust-root-org-id"
	flagTrustRootCrtPath                 = "trust-root-path"
	flagTrustMemberOrgId                 = "trust-member-org-id"
	flagTrustMemberCrtPath               = "trust-member-path"
	flagTrustMemberRole                  = "trust-member-role"
	flagTrustMemberNodeId                = "trust-member-node-id"
	flagCertFilePaths                    = "cert-file-paths"
	flagCertCrlPath                      = "cert-crl-path"
	flagAddress                          = "address"
	flagAmount                           = "amount"
	flagDelegator                        = "delegator"
	flagValidator                        = "validator"
	flagEpochID                          = "epoch-id"
	flagGrantContractList                = "grant-contract-list"
	flagRevokeContractList               = "revoke-contract-list"
	flagGasLimit                         = "gas-limit"
	flagGasEnable                        = "gas-enable"
	flagAddressType                      = "address-type"
	flagPermissionResourceName           = "permission-resource-name"
	flagPermissionResourcePolicyRule     = "permission-resource-policy-rule"
	flagPermissionResourcePolicyOrgList  = "permission-resource-policy-orgList"
	flagPermissionResourcePolicyRoleList = "permission-resource-policy-roleList"
	flagRespResultToString               = "result-to-string"
	flagMultiSignEnableManualRun         = "multi-sign-enable-manual-run"
)

// ClientCMD new client series command
func ClientCMD() *cobra.Command {
	clientCmd := &cobra.Command{
		Use:   "client",
		Short: "client command",
		Long:  "client command",
	}

	clientCmd.AddCommand(contractCMD())
	clientCmd.AddCommand(chainConfigCMD())
	clientCmd.AddCommand(getChainMakerServerVersionCMD())
	clientCmd.AddCommand(certManageCMD())
	clientCmd.AddCommand(blockChainsCMD())
	clientCmd.AddCommand(enableOrDisableGasCMD())
	clientCmd.AddCommand(certAliasCMD())

	return clientCmd
}

var flags *pflag.FlagSet

func init() {
	flags = &pflag.FlagSet{}

	// 压测参数
	flags.IntVarP(&concurrency, flagConcurrency, "c", 1, "specify concurrency count")
	flags.IntVarP(&totalCntPerGoroutine, flagTotalCountPerGoroutine, "t", 1, "specify total count per goroutine")

	// sdk配置路径
	flags.StringVar(&sdkConfPath, flagSdkConfPath, "", "specify sdk config path")

	// 用户合约
	flags.StringVar(&abiFilePath, flagAbiFilePath, "", "specify user EVM contract abi file path, eg: /home/abi.json")
	flags.StringVar(&contractName, flagContractName, "", "specify user contract name, eg: counter-go-1")
	flags.StringVar(&contractAddress, flagContractAddress, "", "specify user contract address")
	flags.StringVar(&version, flagVersion, "", "specify user contract version, eg: 1.0.0")
	flags.StringVar(&byteCodePath, flagByteCodePath, "", "specify user contract byte code path")
	flags.StringVar(&runtimeType, flagRuntimeType, "", "specify user contract runtime type, such as: "+
		"NATIVE | WASMER | WXVM | GASM | EVM | DOCKER_GO | JAVA")
	flags.StringVar(&chainId, flagChainId, "", "specify the chain id, such as: chain1, chain2 etc.")
	flags.IntVar(&sendTimes, flagSendTimes, 1, "specify SendTimes , default once")
	flags.Int64Var(&timeout, flagTimeout, 10, "specify timeout in seconds, default 10s")
	flags.StringVar(&method, flagMethod, "", "specify invoke contract method")
	flags.StringVar(&params, flagParams, "", "specify invoke contract params, json format, "+
		"such as: \"{\\\"key\\\":\\\"value\\\",\\\"key1\\\":\\\"value1\\\"}\"")
	flags.StringVar(&orgId, flagOrgId, "", "specify the orgId, such as wx-org1.chainmaker.com")
	flags.BoolVar(&syncResult, flagSyncResult, false, "whether wait the result of the transaction, default false")
	flags.BoolVar(&enableCertHash, flagEnableCertHash, true, "whether enable cert hash, default true")
	flags.BoolVar(&withRWSet, flagWithRWSet, true, "whether with RWSet, default true")
	flags.BoolVar(&truncateValue, flagTruncateValue, false, "enable truncate value, default false")
	flags.Uint64Var(&blockHeight, flagBlockHeight, 0, "specify block height, default 0")
	flags.StringVar(&txId, flagTxId, "", "specify tx id")
	flags.BoolVar(&isAgree, flagIsAgree, true, "specify multi sign vote choice")

	// Admin秘钥和证书列表
	//    - 使用逗号','分割
	//    - 列表中的key与crt需一一对应
	//    - 如果只有一对，将采用单签模式；如果有多对，将采用多签模式，第一对用于发起多签请求，其余的用于多签投票
	flags.StringVar(&adminKeyFilePaths, flagAdminKeyFilePaths, "", "specify admin key file paths, use ',' to separate")
	flags.StringVar(&adminCrtFilePaths, flagAdminCrtFilePaths, "", "specify admin cert file paths, use ',' to separate")
	flags.StringVar(&adminOrgIds, flagAdminOrgIds, "", "specify admin org-ids, use ',' to separate")

	flags.StringVar(&payerKeyFilePath, flagPayerKeyFilePath, "", "specify payer key file path")
	flags.StringVar(&payerCrtFilePath, flagPayerCrtFilePath, "", "specify payer cert file path")
	flags.StringVar(&payerOrgId, flagPayerOrgId, "", "specify payer org-id")

	flags.StringVar(&userTlsKeyFilePath, flagUserTlsKeyFilePath, "", "specify user tls key file path for "+
		"chainclient tls connection")
	flags.StringVar(&userTlsCrtFilePath, flagUserTlsCrtFilePath, "", "specify user tls cert file path for "+
		"chainclient tls connection")
	flags.StringVar(&userSignKeyFilePath, flagUserSignKeyFilePath, "", "specify user sign key file path to sign tx")
	flags.StringVar(&userSignCrtFilePath, flagUserSignCrtFilePath, "", "specify user sign cert file path to sign tx")

	// 链配置
	flags.Uint32Var(&blockInterval, flagBlockInterval, 2000, "block interval timeout in milliseconds, default 2000ms")
	flags.Uint32Var(&txParameterSize, flagTxParameterSize, 10, "tx parameter size, default 10MB")
	flags.StringVar(&nodeOrgId, flagNodeOrgId, "", "specify node org id")
	flags.StringVar(&nodeIdOld, flagNodeIdOld, "", "specify old node id")
	flags.StringVar(&nodeId, flagNodeId, "", "specify node id(which will be added or update to")
	flags.StringVar(&nodeIds, flagNodeIds, "", "specify node ids(which will be added or update to")

	flags.StringVar(&trustRootOrgId, flagTrustRootOrgId, "", "specify the ca org id")
	flags.StringSliceVar(&trustRootPaths, flagTrustRootCrtPath, nil, "specify the ca file path")

	flags.StringVar(&trustMemberOrgId, flagTrustMemberOrgId, "", "specify the ca org id")
	flags.StringVar(&trustMemberInfoPath, flagTrustMemberCrtPath, "", "specify the ca file path")
	flags.StringVar(&trustMemberRole, flagTrustMemberRole, "", "specify trust member role")
	flags.StringVar(&trustMemberNodeId, flagTrustMemberNodeId, "", "specify trust member node id")

	// consensus extra config
	flags.StringVar(&extraConfigKey, flagExtraConfigKey, "", "extra config key")
	flags.StringVar(&extraConfigValue, flagExtraConfigValue, "", "extra config value")

	// 证书管理
	flags.StringVar(&certFilePaths, flagCertFilePaths, "", "specify cert file paths, use ',' to separate")
	flags.StringVar(&certCrlPath, flagCertCrlPath, "", "specify cert crl path")

	// dpos 系统合约
	flags.StringVar(&address, flagAddress, "", "specify use address")
	flags.StringVar(&amount, flagAmount, "", "specify amount")
	flags.StringVar(&delegator, flagDelegator, "", "specify delegator address")
	flags.StringVar(&validator, flagValidator, "", "specify validator address")
	flags.StringVar(&epochID, flagEpochID, "", "specify epoch id")
	flags.StringSliceVar(&grantContractList, flagGrantContractList, nil, "specify grant list")
	flags.StringSliceVar(&revokeContractList, flagRevokeContractList, nil, "specify revoke list")

	// gas limit
	flags.Uint64Var(&gasLimit, flagGasLimit, 0, "gas limit in uint64 type")
	flags.BoolVar(&gasEnable, flagGasEnable, false, "enable or disable gas feature")

	flags.Int32Var(&addressType, flagAddressType, 0, "address type, eg. ChainMaker:0, ZXL:1")
	flags.StringVar(&permissionResourceName, flagPermissionResourceName, "", "chain config permission resource name")
	flags.StringVar(&permissionResourcePolicyRule, flagPermissionResourcePolicyRule, "",
		"chain config permission resource policy rule")
	flags.StringSliceVar(&permissionResourcePolicyOrgList, flagPermissionResourcePolicyOrgList, []string{},
		"chain config permission resource policy org list")
	flags.StringSliceVar(&permissionResourcePolicyRoleList, flagPermissionResourcePolicyRoleList, []string{},
		"chain config permission resource policy role list")
	flags.BoolVar(&respResultToString, flagRespResultToString, false,
		"enable convert TxResponse.ContractResult.Result to string for readable output")

	// multi-sign enable manual_run
	flags.BoolVar(&multiSignEnableManualRun,
		flagMultiSignEnableManualRun,
		false,
		"enable or disable manual run feature of multi-sign")
	flags.StringVar(&truncateModel,
		flagTruncateModel,
		"",
		"the type of truncating multi-sign info")
	flags.StringVar(&truncateValueLen,
		flagTruncateValueLen,
		"",
		"the max length of truncating multi-sign info")

	if sdkConfPath == "" {
		sdkConfPath = util.EnvSdkConfPath
	}
}

func attachFlags(cmd *cobra.Command, names []string) {
	cmdFlags := cmd.Flags()
	for _, name := range names {
		if flag := flags.Lookup(name); flag != nil {
			flagCopied := *flag
			cmdFlags.AddFlag(&flagCopied)
			//cmdFlags.AddFlag(flag)
		}
	}
}

// getChainMakerServerVersionCMD a command about get chainmaker server version
// @return *cobra.Command
func getChainMakerServerVersionCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cmversion",
		Short: "get chainmaker server version",
		Long:  "get chainmaker server version",
		RunE: func(_ *cobra.Command, _ []string) error {
			return getChainMakerServerVersion()
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath,
		flagSdkConfPath, flagOrgId,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
	})

	return cmd
}

// getChainMakerServerVersion query ChainMaker server version
// @return error
func getChainMakerServerVersion() error {
	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
		userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return err
	}
	defer client.Stop()
	version, err := client.GetChainMakerServerVersion()
	if err != nil {
		return fmt.Errorf("get chainmaker server version failed, %s", err.Error())
	}
	fmt.Printf("current chainmaker server version: %s \n", version)
	return nil
}
