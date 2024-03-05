// Copyright (C) BABEC. All rights reserved.
// Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package query

import (
	"fmt"

	"chainmaker.org/chainmaker-go/tools/cmc/util"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
	"github.com/hokaccha/go-prettyjson"
	"github.com/spf13/cobra"
)

// newQueryTxOnChainCMD `query tx` command implementation
func newQueryTxOnChainCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tx [txid]",
		Short: "query on-chain tx by txid",
		Long:  "query on-chain tx by txid",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			//// 1.Chain Client
			cc, err := sdk.NewChainClient(
				sdk.WithConfPath(sdkConfPath),
				sdk.WithChainClientChainId(chainId),
			)
			if err != nil {
				return err
			}
			defer cc.Stop()
			if err := util.DealChainClientCertHash(cc, enableCertHash); err != nil {
				return err
			}

			//// 2.Query tx on-chain
			truncateLength := 0
			if truncateValue {
				truncateLength = 1000

			}
			txInfo, err := cc.GetTxByTxIdTruncate(args[0], withRWSet, truncateLength, "truncate")
			if err != nil {
				return err
			}

			output, err := prettyjson.Marshal(txInfo)
			if err != nil {
				return err
			}
			fmt.Println(string(output))
			return nil
		},
	}
	util.AttachFlags(cmd, flags, []string{
		flagEnableCertHash, flagWithRWSet, flagTruncateValue, flagChainId, flagSdkConfPath,
	})
	return cmd
}
