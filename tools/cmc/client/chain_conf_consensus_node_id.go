/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"errors"
	"fmt"

	"chainmaker.org/chainmaker-go/tools/cmc/util"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"github.com/spf13/cobra"
)

const (
	addNodeId = iota
	removeNodeId
	updateNodeId
)

// configConsensueNodeIdCMD consensus node id management sub command
//用于管理Consensus下Org下的NodeId
// @return *cobra.Command
func configConsensueNodeIdCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "consensusnodeid",
		Short: "consensus node id management",
		Long:  "consensus node id management",
	}
	cmd.AddCommand(addConsensusNodeIdCMD())
	cmd.AddCommand(removeConsensusNodeIdCMD())
	cmd.AddCommand(updateConsensusNodeIdCMD())

	return cmd
}

// addConsensusNodeIdCMD add consensus node id command
// @return *cobra.Command
func addConsensusNodeIdCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "add consensus node id cmd",
		Long:  "add consensus node id cmd",
		RunE: func(_ *cobra.Command, _ []string) error {
			return configConsensusNodeId(addNodeId)
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath,
		flagSdkConfPath, flagOrgId, flagEnableCertHash, flagNodeOrgId, flagNodeId,
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagAdminOrgIds, flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
	})

	cmd.MarkFlagRequired(flagNodeOrgId)
	cmd.MarkFlagRequired(flagNodeId)

	return cmd
}

// removeConsensusNodeIdCMD remove consensus node id command
// @return *cobra.Command
func removeConsensusNodeIdCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "remove consensus node id cmd",
		Long:  "remove consensus node id cmd",
		RunE: func(_ *cobra.Command, _ []string) error {
			return configConsensusNodeId(removeNodeId)
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath,
		flagSdkConfPath, flagOrgId, flagEnableCertHash, flagNodeOrgId, flagNodeId,
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagAdminOrgIds, flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
	})

	cmd.MarkFlagRequired(flagNodeOrgId)
	cmd.MarkFlagRequired(flagNodeId)

	return cmd
}

// updateConsensusNodeIdCMD update consensus node id command
// @return *cobra.Command
func updateConsensusNodeIdCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "update consensus node id cmd",
		Long:  "update consensus node id cmd",
		RunE: func(_ *cobra.Command, _ []string) error {
			return configConsensusNodeId(updateNodeId)
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath,
		flagSdkConfPath, flagOrgId, flagEnableCertHash, flagNodeOrgId, flagNodeIdOld, flagNodeId,
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagAdminOrgIds, flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
	})

	cmd.MarkFlagRequired(flagNodeOrgId)
	cmd.MarkFlagRequired(flagNodeIdOld)
	cmd.MarkFlagRequired(flagNodeId)

	return cmd
}

// configConsensusNodeId update ChainConfig.Consensus
// @param op 0:add;1remove;2:update
// @return error
func configConsensusNodeId(op int) error {

	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
		userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return err
	}
	defer client.Stop()

	adminKeys, adminCrts, adminOrgs, err := util.MakeAdminInfo(client, adminKeyFilePaths, adminCrtFilePaths, adminOrgIds)
	if err != nil {
		return err
	}

	var payload *common.Payload
	switch op {
	case addNodeId:
		payload, err = client.CreateChainConfigConsensusNodeIdAddPayload(nodeOrgId, []string{nodeId})
	case removeNodeId:
		payload, err = client.CreateChainConfigConsensusNodeIdDeletePayload(nodeOrgId, nodeId)
	case updateNodeId:
		payload, err = client.CreateChainConfigConsensusNodeIdUpdatePayload(nodeOrgId, nodeIdOld, nodeId)
	default:
		err = errors.New("invalid node address operation")
	}
	if err != nil {
		return err
	}

	endorsementEntrys, err := util.MakeEndorsement(adminKeys, adminCrts, adminOrgs, client, payload)
	if err != nil {
		return err
	}

	resp, err := client.SendChainConfigUpdateRequest(payload, endorsementEntrys, -1, syncResult)
	if err != nil {
		return err
	}
	err = util.CheckProposalRequestResp(resp, false)
	if err != nil {
		return err
	}
	fmt.Printf("consensusnodeid response %+v\n", resp)
	return nil
}
