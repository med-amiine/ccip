package oraclecreator

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-ccip/pkg/consts"

	"github.com/smartcontractkit/libocr/commontypes"
	libocr3 "github.com/smartcontractkit/libocr/offchainreporting2plus"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/ocr3types"
	ocrtypes "github.com/smartcontractkit/libocr/offchainreporting2plus/types"

	commitocr3 "github.com/smartcontractkit/chainlink-ccip/commit"
	execocr3 "github.com/smartcontractkit/chainlink-ccip/execute"
	ccipreaderpkg "github.com/smartcontractkit/chainlink-ccip/pkg/reader"
	cciptypes "github.com/smartcontractkit/chainlink-common/pkg/types/ccipocr3"

	"github.com/smartcontractkit/chainlink-common/pkg/types"
	"github.com/smartcontractkit/chainlink/v2/core/chains/legacyevm"
	"github.com/smartcontractkit/chainlink/v2/core/logger"
	evmconfigs "github.com/smartcontractkit/chainlink/v2/core/services/ccipcapability/configs/evm"
	"github.com/smartcontractkit/chainlink/v2/core/services/ccipcapability/ocrimpls"
	cctypes "github.com/smartcontractkit/chainlink/v2/core/services/ccipcapability/types"
	"github.com/smartcontractkit/chainlink/v2/core/services/job"
	"github.com/smartcontractkit/chainlink/v2/core/services/keystore/chaintype"
	"github.com/smartcontractkit/chainlink/v2/core/services/keystore/keys/ocr2key"
	"github.com/smartcontractkit/chainlink/v2/core/services/ocr3/plugins/ccipevm"
	"github.com/smartcontractkit/chainlink/v2/core/services/ocrcommon"
	"github.com/smartcontractkit/chainlink/v2/core/services/relay"
	"github.com/smartcontractkit/chainlink/v2/core/services/relay/evm"
	evmrelaytypes "github.com/smartcontractkit/chainlink/v2/core/services/relay/evm/types"
	"github.com/smartcontractkit/chainlink/v2/core/services/synchronization"
	"github.com/smartcontractkit/chainlink/v2/core/services/telemetry"
)

var _ cctypes.OracleCreator = &inprocessOracleCreator{}

// inprocessOracleCreator creates oracles that reference plugins running
// in the same process as the chainlink node, i.e not LOOPPs.
type inprocessOracleCreator struct {
	ocrKeyBundles         map[string]ocr2key.KeyBundle
	transmitters          map[types.RelayID][]string
	chains                legacyevm.LegacyChainContainer
	peerWrapper           *ocrcommon.SingletonPeerWrapper
	externalJobID         uuid.UUID
	jobID                 int32
	isNewlyCreatedJob     bool
	pluginConfig          job.JSONConfig
	db                    ocr3types.Database
	lggr                  logger.Logger
	monitoringEndpointGen telemetry.MonitoringEndpointGenerator
	bootstrapperLocators  []commontypes.BootstrapperLocator
	homeChainReader       ccipreaderpkg.HomeChain
}

func New(
	ocrKeyBundles map[string]ocr2key.KeyBundle,
	transmitters map[types.RelayID][]string,
	chains legacyevm.LegacyChainContainer,
	peerWrapper *ocrcommon.SingletonPeerWrapper,
	externalJobID uuid.UUID,
	jobID int32,
	isNewlyCreatedJob bool,
	pluginConfig job.JSONConfig,
	db ocr3types.Database,
	lggr logger.Logger,
	monitoringEndpointGen telemetry.MonitoringEndpointGenerator,
	bootstrapperLocators []commontypes.BootstrapperLocator,
	homeChainReader ccipreaderpkg.HomeChain,
) cctypes.OracleCreator {
	return &inprocessOracleCreator{
		ocrKeyBundles:         ocrKeyBundles,
		transmitters:          transmitters,
		chains:                chains,
		peerWrapper:           peerWrapper,
		externalJobID:         externalJobID,
		jobID:                 jobID,
		isNewlyCreatedJob:     isNewlyCreatedJob,
		pluginConfig:          pluginConfig,
		db:                    db,
		lggr:                  lggr,
		monitoringEndpointGen: monitoringEndpointGen,
		bootstrapperLocators:  bootstrapperLocators,
		homeChainReader:       homeChainReader,
	}
}

// CreateBootstrapOracle implements types.OracleCreator.
func (i *inprocessOracleCreator) CreateBootstrapOracle(config cctypes.OCR3ConfigWithMeta) (cctypes.CCIPOracle, error) {
	// Assuming that the chain selector is referring to an evm chain for now.
	// TODO: add an api that returns chain family.
	chainID, err := chainsel.ChainIdFromSelector(uint64(config.Config.ChainSelector))
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID from selector: %w", err)
	}

	destChainFamily := chaintype.EVM
	destRelayID := types.NewRelayID(string(destChainFamily), fmt.Sprintf("%d", chainID))

	bootstrapperArgs := libocr3.BootstrapperArgs{
		BootstrapperFactory:   i.peerWrapper.Peer2,
		V2Bootstrappers:       i.bootstrapperLocators,
		ContractConfigTracker: ocrimpls.NewConfigTracker(config),
		Database:              i.db,
		LocalConfig:           defaultLocalConfig(),
		Logger: ocrcommon.NewOCRWrapper(
			i.lggr.
				Named("CCIPBootstrap").
				Named(destRelayID.String()).
				Named(config.Config.ChainSelector.String()).
				Named(hexutil.Encode(config.Config.OfframpAddress)),
			false, /* traceLogging */
			func(ctx context.Context, msg string) {}),
		MonitoringEndpoint: i.monitoringEndpointGen.GenMonitoringEndpoint(
			string(destChainFamily),
			destRelayID.ChainID,
			hexutil.Encode(config.Config.OfframpAddress),
			synchronization.OCR3CCIPBootstrap,
		),
		OffchainConfigDigester: ocrimpls.NewConfigDigester(config.ConfigDigest),
	}
	bootstrapper, err := libocr3.NewBootstrapper(bootstrapperArgs)
	if err != nil {
		return nil, err
	}
	return bootstrapper, nil
}

// CreatePluginOracle implements types.OracleCreator.
func (i *inprocessOracleCreator) CreatePluginOracle(pluginType cctypes.PluginType, config cctypes.OCR3ConfigWithMeta) (cctypes.CCIPOracle, error) {
	// Assuming that the chain selector is referring to an evm chain for now.
	// TODO: add an api that returns chain family.
	destChainID, err := chainsel.ChainIdFromSelector(uint64(config.Config.ChainSelector))
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID from selector %d: %w", config.Config.ChainSelector, err)
	}
	destChainFamily := relay.NetworkEVM
	destRelayID := types.NewRelayID(destChainFamily, fmt.Sprintf("%d", destChainID))

	// this is so that we can use the msg hasher and report encoder from that dest chain relayer's provider.
	contractReaders := make(map[cciptypes.ChainSelector]types.ContractReader)
	chainWriters := make(map[cciptypes.ChainSelector]types.ChainWriter)
	for _, chain := range i.chains.Slice() {
		var chainReaderConfig evmrelaytypes.ChainReaderConfig
		if chain.ID().Uint64() == destChainID {
			chainReaderConfig = evmconfigs.DestReaderConfig()
		} else {
			chainReaderConfig = evmconfigs.SourceReaderConfig()
		}
		cr, err2 := evm.NewChainReaderService(
			context.Background(),
			i.lggr.
				Named("EVMChainReaderService").
				Named(chain.ID().String()).
				Named(pluginType.String()),
			chain.LogPoller(),
			chain.Client(),
			chainReaderConfig,
		)
		if err2 != nil {
			return nil, fmt.Errorf("failed to create contract reader for chain %s: %w", chain.ID(), err2)
		}

		if chain.ID().Uint64() == destChainID {
			// bind the chain reader to the dest chain's offramp.
			offrampAddressHex := common.BytesToAddress(config.Config.OfframpAddress).Hex()
			err3 := cr.Bind(context.Background(), []types.BoundContract{
				{
					Address: offrampAddressHex,
					Name:    consts.ContractNameOffRamp,
				},
			})
			if err3 != nil {
				return nil, fmt.Errorf("failed to bind chain reader for dest chain %s's offramp at %s: %w", chain.ID(), offrampAddressHex, err3)
			}
		}

		// TODO: figure out shutdown.
		// maybe from the plugin directly?
		err2 = cr.Start(context.Background())
		if err2 != nil {
			return nil, fmt.Errorf("failed to start contract reader for chain %s: %w", chain.ID(), err2)
		}

		// Even though we only write to the dest chain, we need to create chain writers for all chains
		// we know about in order to post gas prices on the dest.
		var fromAddress common.Address
		transmitter, ok := i.transmitters[types.NewRelayID(relay.NetworkEVM, chain.ID().String())]
		if ok {
			fromAddress = common.HexToAddress(transmitter[0])
		}
		cw, err2 := evm.NewChainWriterService(
			i.lggr.Named("EVMChainWriterService").
				Named(chain.ID().String()).
				Named(pluginType.String()),
			chain.Client(),
			chain.TxManager(),
			chain.GasEstimator(),
			evmconfigs.ChainWriterConfigRaw(fromAddress, chain.Config().EVM().GasEstimator().PriceMaxKey(fromAddress)),
		)
		if err2 != nil {
			return nil, fmt.Errorf("failed to create chain writer for chain %s: %w", chain.ID(), err2)
		}

		// TODO: figure out shutdown.
		// maybe from the plugin directly?
		err2 = cw.Start(context.Background())
		if err2 != nil {
			return nil, fmt.Errorf("failed to start chain writer for chain %s: %w", chain.ID(), err2)
		}

		chainSelector, ok := chainsel.EvmChainIdToChainSelector()[chain.ID().Uint64()]
		if !ok {
			return nil, fmt.Errorf("failed to get chain selector from chain ID %s", chain.ID())
		}

		contractReaders[cciptypes.ChainSelector(chainSelector)] = cr
		chainWriters[cciptypes.ChainSelector(chainSelector)] = cw
	}

	// build the onchain keyring. it will be the signing key for the destination chain family.
	keybundle, ok := i.ocrKeyBundles[destChainFamily]
	if !ok {
		return nil, fmt.Errorf("no OCR key bundle found for chain family %s, forgot to create one?", destChainFamily)
	}
	onchainKeyring := ocrimpls.NewOnchainKeyring[[]byte](keybundle, i.lggr)

	// build the contract transmitter
	// assume that we are using the first account in the keybundle as the from account
	// and that we are able to transmit to the dest chain.
	// TODO: revisit this in the future, since not all oracles will be able to transmit to the dest chain.
	destChainWriter, ok := chainWriters[config.Config.ChainSelector]
	if !ok {
		return nil, fmt.Errorf("no chain writer found for dest chain selector %d, can't create contract transmitter",
			config.Config.ChainSelector)
	}
	destFromAccounts, ok := i.transmitters[destRelayID]
	if !ok {
		return nil, fmt.Errorf("no transmitter found for dest relay ID %s, can't create contract transmitter", destRelayID)
	}

	//TODO: Extract the correct transmitter address from the destsFromAccount
	var factory ocr3types.ReportingPluginFactory[[]byte]
	var transmitter ocr3types.ContractTransmitter[[]byte]
	if config.Config.PluginType == uint8(cctypes.PluginTypeCCIPCommit) {
		factory = commitocr3.NewPluginFactory(
			i.lggr.
				Named("CCIPCommitPlugin").
				Named(destRelayID.String()).
				Named(hexutil.Encode(config.Config.OfframpAddress)),
			ccipreaderpkg.OCR3ConfigWithMeta(config),
			ccipevm.NewCommitPluginCodecV1(),
			ccipevm.NewMessageHasherV1(),
			i.homeChainReader,
			contractReaders,
			chainWriters,
		)
		transmitter = ocrimpls.NewCommitContractTransmitter[[]byte](destChainWriter,
			ocrtypes.Account(destFromAccounts[0]),
			hexutil.Encode(config.Config.OfframpAddress), // TODO: this works for evm only, how about non-evm?
		)
	} else if config.Config.PluginType == uint8(cctypes.PluginTypeCCIPExec) {
		factory = execocr3.NewPluginFactory(
			i.lggr.
				Named("CCIPExecPlugin").
				Named(destRelayID.String()).
				Named(hexutil.Encode(config.Config.OfframpAddress)),
			ccipreaderpkg.OCR3ConfigWithMeta(config),
			ccipevm.NewExecutePluginCodecV1(),
			ccipevm.NewMessageHasherV1(),
			i.homeChainReader,
			contractReaders,
			chainWriters,
		)
		transmitter = ocrimpls.NewExecContractTransmitter[[]byte](destChainWriter,
			ocrtypes.Account(destFromAccounts[0]),
			hexutil.Encode(config.Config.OfframpAddress), // TODO: this works for evm only, how about non-evm?
		)
	} else {
		return nil, fmt.Errorf("unsupported plugin type %d", config.Config.PluginType)
	}

	oracleArgs := libocr3.OCR3OracleArgs[[]byte]{
		BinaryNetworkEndpointFactory: i.peerWrapper.Peer2,
		Database:                     i.db,
		V2Bootstrappers:              i.bootstrapperLocators,
		ContractConfigTracker:        ocrimpls.NewConfigTracker(config),
		ContractTransmitter:          transmitter,
		LocalConfig:                  defaultLocalConfig(),
		Logger: ocrcommon.NewOCRWrapper(
			i.lggr.
				Named(fmt.Sprintf("CCIP%sOCR3", pluginType.String())).
				Named(destRelayID.String()).
				Named(hexutil.Encode(config.Config.OfframpAddress)),
			false,
			func(ctx context.Context, msg string) {}),
		MetricsRegisterer: prometheus.WrapRegistererWith(map[string]string{"name": fmt.Sprintf("commit-%d", config.Config.ChainSelector)}, prometheus.DefaultRegisterer),
		MonitoringEndpoint: i.monitoringEndpointGen.GenMonitoringEndpoint(
			destChainFamily,
			destRelayID.ChainID,
			string(config.Config.OfframpAddress),
			synchronization.OCR3CCIPCommit,
		),
		OffchainConfigDigester: ocrimpls.NewConfigDigester(config.ConfigDigest),
		OffchainKeyring:        keybundle,
		OnchainKeyring:         onchainKeyring,
		ReportingPluginFactory: factory,
	}
	oracle, err := libocr3.NewOracle(oracleArgs)
	if err != nil {
		return nil, err
	}
	return oracle, nil
}

func defaultLocalConfig() ocrtypes.LocalConfig {
	return ocrtypes.LocalConfig{
		BlockchainTimeout: 10 * time.Second,
		// Config tracking is handled by the launcher, since we're doing blue-green
		// deployments we're not going to be using OCR's built-in config switching,
		// which always shuts down the previous instance.
		ContractConfigConfirmations:        1,
		SkipContractConfigConfirmations:    true,
		ContractConfigTrackerPollInterval:  10 * time.Second,
		ContractTransmitterTransmitTimeout: 10 * time.Second,
		DatabaseTimeout:                    10 * time.Second,
		MinOCR2MaxDurationQuery:            1 * time.Second,
		DevelopmentMode:                    "false",
	}
}
