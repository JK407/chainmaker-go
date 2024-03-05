// Copyright (C) BABEC. All rights reserved.
// Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"errors"
	"fmt"

	"chainmaker.org/chainmaker/pb-go/v2/common"
)

// CheckProposalRequestResp check *common.TxResponse is success
func CheckProposalRequestResp(resp *common.TxResponse, needContractResult bool) error {
	if resp.Code != common.TxStatusCode_SUCCESS {
		if resp.Message == "" {
			if resp.ContractResult != nil && resp.ContractResult.Code != 0 && resp.ContractResult.Message != "" {
				return errors.New(resp.ContractResult.Message)
			}
			return errors.New(resp.Code.String())
		}
		return errors.New(resp.Message)
	}

	if needContractResult && resp.ContractResult == nil {
		return fmt.Errorf("contract result is nil")
	}

	if resp.ContractResult != nil && resp.ContractResult.Code != 0 {
		return errors.New(resp.ContractResult.Message)
	}

	return nil
}
