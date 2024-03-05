/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"chainmaker.org/chainmaker-go/tools/cmc/util"
	commonpb "chainmaker.org/chainmaker/pb-go/v2/common"
	sdkutils "chainmaker.org/chainmaker/sdk-go/v2/utils"
	"github.com/hokaccha/go-prettyjson"
	"github.com/spf13/cobra"
)

const (
	operationFreeze   = "freeze"
	operationUnfreeze = "unfreeze"
)

// certManageCMD cert manage sub command
// @return *cobra.Command
func certManageCMD() *cobra.Command {
	chainConfigCmd := &cobra.Command{
		Use:   "certmanage",
		Short: "cert manage command",
		Long:  "cert manage command",
	}
	chainConfigCmd.AddCommand(freezeCertCMD())
	chainConfigCmd.AddCommand(unfreezeCertCMD())
	chainConfigCmd.AddCommand(revokeCertCMD())
	return chainConfigCmd
}

// freezeCertCMD freeze a cert
// @return *cobra.Command
func freezeCertCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "freeze",
		Short: "freeze cert",
		Long:  "freeze cert",
		RunE: func(_ *cobra.Command, _ []string) error {
			return freezeOrUnfreezeCert(1)
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath,
		flagSdkConfPath, flagOrgId, flagChainId, flagSyncResult,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath, flagAdminCrtFilePaths, flagAdminKeyFilePaths,
		flagCertFilePaths, flagEnableCertHash,
	})

	cmd.MarkFlagRequired(flagAdminCrtFilePaths)
	cmd.MarkFlagRequired(flagAdminKeyFilePaths)
	cmd.MarkFlagRequired(flagCertFilePaths)

	return cmd
}

// unfreezeCertCMD unfreeze a cert
// @return *cobra.Command
func unfreezeCertCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unfreeze",
		Short: "unfreeze cert",
		Long:  "unfreeze cert",
		RunE: func(_ *cobra.Command, _ []string) error {
			return freezeOrUnfreezeCert(2)
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath,
		flagSdkConfPath, flagOrgId, flagChainId, flagSyncResult,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath, flagAdminCrtFilePaths, flagAdminKeyFilePaths,
		flagCertFilePaths, flagEnableCertHash,
	})

	cmd.MarkFlagRequired(flagAdminCrtFilePaths)
	cmd.MarkFlagRequired(flagAdminKeyFilePaths)
	cmd.MarkFlagRequired(flagCertFilePaths)

	return cmd
}

// revokeCertCMD revoke a cert
// @return *cobra.Command
func revokeCertCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revoke",
		Short: "revoke cert",
		Long:  "revoke cert",
		RunE: func(_ *cobra.Command, _ []string) error {
			return revokeCert()
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath,
		flagSdkConfPath, flagOrgId, flagChainId, flagSyncResult,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath, flagAdminCrtFilePaths, flagAdminKeyFilePaths,
		flagCertCrlPath, flagEnableCertHash,
	})

	cmd.MarkFlagRequired(flagAdminCrtFilePaths)
	cmd.MarkFlagRequired(flagAdminKeyFilePaths)
	cmd.MarkFlagRequired(flagCertCrlPath)

	return cmd
}

// freezeOrUnfreezeCert
// @param which 1: frozen, 2: unfrozen
// @return error
func freezeOrUnfreezeCert(which int) error {
	var adminKeys, adminCrts []string

	if adminKeyFilePaths != "" {
		adminKeys = strings.Split(adminKeyFilePaths, ",")
	}
	if adminCrtFilePaths != "" {
		adminCrts = strings.Split(adminCrtFilePaths, ",")
	}
	if len(adminKeys) != len(adminCrts) {
		return fmt.Errorf(ADMIN_ORGID_KEY_CERT_LENGTH_NOT_EQUAL_FORMAT, len(adminKeys), len(adminCrts))
	}

	certFiles := strings.Split(certFilePaths, ",")
	for idx := range certFiles {
		path := certFiles[idx]
		path = filepath.Join(path)
		certBytes, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		certStr := string(certBytes)
		certFiles[idx] = certStr
	}

	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
		userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return fmt.Errorf("create user client failed, %s", err.Error())
	}
	defer client.Stop()

	var payload *commonpb.Payload
	var whichOperation string
	switch which {
	case 1:
		payload = client.CreateCertManageFrozenPayload(certFiles)
		whichOperation = operationFreeze
	case 2:
		payload = client.CreateCertManageUnfrozenPayload(certFiles)
		whichOperation = operationUnfreeze
	default:
		err = fmt.Errorf("wrong which param")
	}
	if err != nil {
		return fmt.Errorf("create cert manage %s payload failed, %s", whichOperation, err.Error())
	}

	endorsementEntrys := make([]*commonpb.EndorsementEntry, len(adminKeys))
	for i := range adminKeys {
		e, err := sdkutils.MakeEndorserWithPath(adminKeys[i], adminCrts[i], payload)
		if err != nil {
			return fmt.Errorf("sign cert manage payload failed, %s", err.Error())
		}

		endorsementEntrys[i] = e
	}

	resp, err := client.SendCertManageRequest(payload, endorsementEntrys, -1, syncResult)
	if err != nil {
		return fmt.Errorf("send cert manage request failed, %s", err.Error())
	}
	err = util.CheckProposalRequestResp(resp, false)
	if err != nil {
		return fmt.Errorf("check proposal request resp failed, %s", err.Error())
	}

	output, err := prettyjson.Marshal(resp)
	if err != nil {
		return err
	}
	fmt.Println(string(output))
	return nil
}

// revokeCert use CRL to revoke cert
// @return error
func revokeCert() error {
	var adminKeys, adminCrts []string

	if adminKeyFilePaths != "" {
		adminKeys = strings.Split(adminKeyFilePaths, ",")
	}
	if adminCrtFilePaths != "" {
		adminCrts = strings.Split(adminCrtFilePaths, ",")
	}
	if len(adminKeys) != len(adminCrts) {
		return fmt.Errorf(ADMIN_ORGID_KEY_CERT_LENGTH_NOT_EQUAL_FORMAT, len(adminKeys), len(adminCrts))
	}

	crlBytes, err := ioutil.ReadFile(certCrlPath)
	if err != nil {
		return err
	}
	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
		userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return fmt.Errorf("create user client failed, %s", err.Error())
	}
	defer client.Stop()

	payload := client.CreateCertManageRevocationPayload(string(crlBytes))

	endorsementEntrys := make([]*commonpb.EndorsementEntry, len(adminKeys))
	for i := range adminKeys {
		e, err := sdkutils.MakeEndorserWithPath(adminKeys[i], adminCrts[i], payload)
		if err != nil {
			return fmt.Errorf("sign cert manage payload failed, %s", err.Error())
		}

		endorsementEntrys[i] = e
	}

	resp, err := client.SendCertManageRequest(payload, endorsementEntrys, -1, syncResult)

	if err != nil {
		return fmt.Errorf("send cert manage request failed, %s", err.Error())
	}

	err = util.CheckProposalRequestResp(resp, false)
	if err != nil {
		return fmt.Errorf("check proposal request resp failed, %s", err.Error())
	}

	output, err := prettyjson.Marshal(resp)
	if err != nil {
		return err
	}
	fmt.Println(string(output))
	return nil
}
