/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	maxbftMode "chainmaker.org/chainmaker-go/module/core/maxbftmode"
	"chainmaker.org/chainmaker-go/module/core/provider"
	syncMode "chainmaker.org/chainmaker-go/module/core/syncmode"
)

func init() {
	provider.RegisterCoreEngineProvider(syncMode.ConsensusTypeSOLO, syncMode.NilSOLOProvider)
	provider.RegisterCoreEngineProvider(syncMode.ConsensusTypeRAFT, syncMode.NilRAFTProvider)
	provider.RegisterCoreEngineProvider(syncMode.ConsensusTypeTBFT, syncMode.NilTBFTProvider)
	provider.RegisterCoreEngineProvider(syncMode.ConsensusTypeDPOS, syncMode.NilDPOSProvider)
	provider.RegisterCoreEngineProvider(maxbftMode.ConsensusTypeMAXBFT, maxbftMode.NilTMAXBFTProvider)
}
