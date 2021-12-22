// provides tools for wrapping solidity contracts with
// golang packages, using abigen.
package gethwrappers

//go:generate ./generation/compile_contracts.sh

//go:generate go run ./generation/generate/wrap.go ../../contracts/solc/v0.6/FluxAggregator.abi ../../contracts/solc/v0.6/FluxAggregator.bin FluxAggregator flux_aggregator_wrapper
//go:generate go run ./generation/generate/wrap.go ../../contracts/solc/v0.6/VRFCoordinator.abi ../../contracts/solc/v0.6/VRFCoordinator.bin VRFCoordinator solidity_vrf_coordinator_interface
//go:generate go run ./generation/generate/wrap.go ../../contracts/solc/v0.6/VRFConsumer.abi ../../contracts/solc/v0.6/VRFConsumer.bin VRFConsumer solidity_vrf_consumer_interface
//go:generate go run ./generation/generate/wrap.go ../../contracts/solc/v0.6/Flags.abi ../../contracts/solc/v0.6/Flags.bin Flags flags_wrapper
//go:generate go run ./generation/generate/wrap.go ../../contracts/solc/v0.6/Oracle.abi ../../contracts/solc/v0.6/Oracle.bin Oracle oracle_wrapper
//go:generate go run ./generation/generate/wrap.go ../../contracts/solc/v0.7/Consumer.abi ../../contracts/solc/v0.7/Consumer.bin Consumer consumer_wrapper
//go:generate go run ./generation/generate/wrap.go ../../contracts/solc/v0.7/MultiWordConsumer.abi ../../contracts/solc/v0.7/MultiWordConsumer.bin MultiWordConsumer multiwordconsumer_wrapper
//go:generate go run ./generation/generate/wrap.go ../../contracts/solc/v0.7/Operator.abi ../../contracts/solc/v0.7/Operator.bin Operator operator_wrapper
//go:generate go run ./generation/generate/wrap.go OffchainAggregator/OffchainAggregator.abi - OffchainAggregator offchain_aggregator_wrapper

// v0.8 VRFConsumer

// VRF V2
//go:generate go run ./generation/generate/wrap.go ../../contracts/solc/v0.8/VRFCoordinatorV2.abi ../../contracts/solc/v0.8/VRFCoordinatorV2.bin VRFCoordinatorV2 vrf_coordinator_v2
//go:generate go run ./generation/generate/wrap.go ../../contracts/solc/v0.8/VRFConsumerV2.abi ../../contracts/solc/v0.8/VRFConsumerV2.bin VRFConsumerV2 vrf_consumer_v2
//go:generate go run ./generation/generate/wrap.go ../../contracts/solc/v0.8/VRFMaliciousConsumerV2.abi ../../contracts/solc/v0.8/VRFMaliciousConsumerV2.bin VRFMaliciousConsumerV2 vrf_malicious_consumer_v2

