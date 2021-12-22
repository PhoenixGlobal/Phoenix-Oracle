// SPDX-License-Identifier: MIT
pragma solidity ^0.7.0;

import "./AccessControllerInterface.sol";
import "./PhbTokenInterface.sol";
import "./Owned.sol";

/**
 * @notice tracks administration of oracle-reward and gas-reimbursement parameters.

 * @dev
 * If you read or change this, be sure to read or adjust the comments. They
 * track the units of the values under consideration, and are crucial to
 * the readability of the operations it specifies.

 * @notice
 * Trust Model:

 * Nothing in this contract prevents a billing admin from setting insane
 * values for the billing parameters in setBilling. Oracles
 * participating in this contract should regularly check that the
 * parameters make sense. Similarly, the outstanding obligations of this
 * contract to the oracles can exceed the funds held by the contract.
 * Oracles participating in this contract should regularly check that it
 * holds sufficient funds and stop interacting with it if funding runs
 * out.

 * This still leaves oracles with some risk due to TOCTOU issues.
 * However, since the sums involved are pretty small (Ethereum
 * transactions aren't that expensive in the end) and an oracle would
 * likely stop participating in a contract it repeatedly lost money on,
 * this risk is deemed acceptable. Oracles should also regularly
 * withdraw any funds in the contract to prevent issues where the
 * contract becomes underfunded at a later time, and different oracles
 * are competing for the left-over funds.

 * Finally, note that any change to the set of oracles or to the billing
 * parameters will trigger payout of all oracles first (using the old
 * parameters), a billing admin cannot take away funds that are already
 * marked for payment.
*/
contract OffchainAggregatorBilling is Owned {

  // Maximum number of oracles the offchain reporting protocol is designed for
  uint256 constant internal maxNumOracles = 31;

  // Parameters for oracle payments
  struct Billing {

    // Highest compensated gas price, in ETH-gwei uints
    uint32 maximumGasPrice;

    // If gas price is less (in ETH-gwei units), transmitter gets half the savings
    uint32 reasonableGasPrice;

    // Pay transmitter back this much PHB per unit eth spent on gas
    // (1e-6PHB/ETH units)
    uint32 microPhbPerEth;

    // Fixed PHB reward for each observer, in PHB-gwei units
    uint32 phbGweiPerObservation;

    // Fixed reward for transmitter, in phbGweiPerObservation units
    uint32 phbGweiPerTransmission;
  }
  Billing internal s_billing;

  // We assume that the token contract is correct. This contract is not written
  // to handle misbehaving ERC20 tokens!
  PhbTokenInterface internal s_phbToken;

  AccessControllerInterface internal s_billingAccessController;

  // ith element is number of observation rewards due to ith process, plus one.
  // This is expected to saturate after an oracle has submitted 65,535
  // observations, or about 65535/(3*24*20) = 45 days, given a transmission
  // every 3 minutes.
  //
  // This is always one greater than the actual value, so that when the value is
  // reset to zero, we don't end up with a zero value in storage (which would
  // result in a higher gas cost, the next time the value is incremented.)
  // Calculations using this variable need to take that offset into account.
  uint16[maxNumOracles] internal s_oracleObservationsCounts;

  // Addresses at which oracles want to receive payments, by transmitter address
  mapping (address /* transmitter */ => address /* payment address */)
    internal
    s_payees;

  // Payee addresses which must be approved by the owner
  mapping (address /* transmitter */ => address /* payment address */)
    internal
    s_proposedPayees;

  // PHB-wei-denominated reimbursements for gas used by transmitters.
  //
  // This is always one greater than the actual value, so that when the value is
  // reset to zero, we don't end up with a zero value in storage (which would
  // result in a higher gas cost, the next time the value is incremented.)
  // Calculations using this variable need to take that offset into account.
  //
  // Argument for overflow safety:
  // We have the following maximum intermediate values:
  // - 2**40 additions to this variable (epochAndRound is a uint40)
  // - 2**32 gas price in ethgwei/gas
  // - 1e9 ethwei/ethgwei
  // - 2**32 gas since the block gas limit is at ~20 million
  // - 2**32 (microphb/eth)
  // And we have 2**40 * 2**32 * 1e9 * 2**32 * 2**32 < 2**166
  // (we also divide in some places, but that only makes the value smaller)
  // We can thus safely use uint256 intermediate values for the computation
  // updating this variable.
  uint256[maxNumOracles] internal s_gasReimbursementsPhbWei;

  // Used for s_oracles[a].role, where a is an address, to track the purpose
  // of the address, or to indicate that the address is unset.
  enum Role {
    // No oracle role has been set for address a
    Unset,
    // Signing address for the s_oracles[a].index'th oracle. I.e., report
    // signatures from this oracle should ecrecover back to address a.
    Signer,
    // Transmission address for the s_oracles[a].index'th oracle. I.e., if a
    // report is received by OffchainAggregator.transmit in which msg.sender is
    // a, it is attributed to the s_oracles[a].index'th oracle.
    Transmitter
  }

  struct Oracle {
    uint8 index; // Index of oracle in s_signers/s_transmitters
    Role role;   // Role of the address which mapped to this struct
  }

  mapping (address /* signer OR transmitter address */ => Oracle)
    internal s_oracles;

  // s_signers contains the signing address of each oracle
  address[] internal s_signers;

  // s_transmitters contains the transmission address of each oracle,
  // i.e. the address the oracle actually sends transactions to the contract from
  address[] internal s_transmitters;

  uint256 constant private  maxUint16 = (1 << 16) - 1;
  uint256 constant internal maxUint128 = (1 << 128) - 1;

  constructor(
    uint32 _maximumGasPrice,
    uint32 _reasonableGasPrice,
    uint32 _microPhbPerEth,
    uint32 _phbGweiPerObservation,
    uint32 _phbGweiPerTransmission,
    PhbTokenInterface _phb,
    AccessControllerInterface _billingAccessController
  )
  {
    setBillingInternal(_maximumGasPrice, _reasonableGasPrice, _microPhbPerEth,
      _phbGweiPerObservation, _phbGweiPerTransmission);
    s_phbToken = _phb;
    emit PhbTokenSet(PhbTokenInterface(address(0)), _phb);
    setBillingAccessControllerInternal(_billingAccessController);
    uint16[maxNumOracles] memory counts; // See s_oracleObservationsCounts docstring
    uint256[maxNumOracles] memory gas; // see s_gasReimbursementsPhbWei docstring
    for (uint8 i = 0; i < maxNumOracles; i++) {
      counts[i] = 1;
      gas[i] = 1;
    }
    s_oracleObservationsCounts = counts;
    s_gasReimbursementsPhbWei = gas;
  }

  /*
   * @notice emitted when the PHB token contract is set
   * @param _oldPhbToken the address of the old PHB token contract
   * @param _newPhbToken the address of the new PHB token contract
   */
  event PhbTokenSet(
    PhbTokenInterface indexed _oldPhbToken,
    PhbTokenInterface indexed _newPhbToken
  );

  /*
   * @notice sets the PHB token contract used for paying oracles
   * @param _phbToken the address of the PHB token contract
   * @param _recipient remaining funds from the previous token contract are transfered
   * here
   * @dev this function will return early (without an error) without changing any state
   * if _phbToken equals getPhbToken().
   * @dev this will trigger a payout so that a malicious owner cannot take from oracles
   * what is already owed to them.
   * @dev we assume that the token contract is correct. This contract is not written
   * to handle misbehaving ERC20 tokens!
   */
  function setPhbToken(
    PhbTokenInterface _phbToken,
    address _recipient
  ) external
    onlyOwner()
  {
    PhbTokenInterface oldPhbToken = s_phbToken;
    if (_phbToken == oldPhbToken) {
      // No change, nothing to be done
      return;
    }
    // call balanceOf as a sanity check on whether we're talking to a token
    // contract
    _phbToken.balanceOf(address(this));
    // we break CEI here, but that's okay because we're dealing with a correct
    // token contract (by assumption).
    payOracles();
    uint256 remainingBalance = oldPhbToken.balanceOf(address(this));
    require(oldPhbToken.transfer(_recipient, remainingBalance), "transfer remaining funds failed");
    s_phbToken = _phbToken;
    emit PhbTokenSet(oldPhbToken, _phbToken);
  }

  /*
   * @notice gets the PHB token contract used for paying oracles
   * @return phbToken the address of the PHB token contract
   */
  function getPhbToken()
    external
    view
    returns(PhbTokenInterface phbToken)
  {
    return s_phbToken;
  }

  /**
   * @notice emitted when billing parameters are set
   * @param maximumGasPrice highest gas price for which transmitter will be compensated
   * @param reasonableGasPrice transmitter will receive reward for gas prices under this value
   * @param microPhbPerEth reimbursement per ETH of gas cost, in 1e-6PHB units
   * @param phbGweiPerObservation reward to oracle for contributing an observation to a successfully transmitted report, in 1e-9PHB units
   * @param phbGweiPerTransmission reward to transmitter of a successful report, in 1e-9PHB units
   */
  event BillingSet(
    uint32 maximumGasPrice,
    uint32 reasonableGasPrice,
    uint32 microPhbPerEth,
    uint32 phbGweiPerObservation,
    uint32 phbGweiPerTransmission
  );

  function setBillingInternal(
    uint32 _maximumGasPrice,
    uint32 _reasonableGasPrice,
    uint32 _microPhbPerEth,
    uint32 _phbGweiPerObservation,
    uint32 _phbGweiPerTransmission
  )
    internal
  {
    s_billing = Billing(_maximumGasPrice, _reasonableGasPrice, _microPhbPerEth,
      _phbGweiPerObservation, _phbGweiPerTransmission);
    emit BillingSet(_maximumGasPrice, _reasonableGasPrice, _microPhbPerEth,
      _phbGweiPerObservation, _phbGweiPerTransmission);
  }

  /**
   * @notice sets billing parameters
   * @param _maximumGasPrice highest gas price for which transmitter will be compensated
   * @param _reasonableGasPrice transmitter will receive reward for gas prices under this value
   * @param _microPhbPerEth reimbursement per ETH of gas cost, in 1e-6PHB units
   * @param _phbGweiPerObservation reward to oracle for contributing an observation to a successfully transmitted report, in 1e-9PHB units
   * @param _phbGweiPerTransmission reward to transmitter of a successful report, in 1e-9PHB units
   * @dev access control provided by billingAccessController
   */
  function setBilling(
    uint32 _maximumGasPrice,
    uint32 _reasonableGasPrice,
    uint32 _microPhbPerEth,
    uint32 _phbGweiPerObservation,
    uint32 _phbGweiPerTransmission
  )
    external
  {
    AccessControllerInterface access = s_billingAccessController;
    require(msg.sender == owner || access.hasAccess(msg.sender, msg.data),
      "Only owner&billingAdmin can call");
    payOracles();
    setBillingInternal(_maximumGasPrice, _reasonableGasPrice, _microPhbPerEth,
      _phbGweiPerObservation, _phbGweiPerTransmission);
  }

  /**
   * @notice gets billing parameters
   * @param maximumGasPrice highest gas price for which transmitter will be compensated
   * @param reasonableGasPrice transmitter will receive reward for gas prices under this value
   * @param microPhbPerEth reimbursement per ETH of gas cost, in 1e-6PHB units
   * @param phbGweiPerObservation reward to oracle for contributing an observation to a successfully transmitted report, in 1e-9PHB units
   * @param phbGweiPerTransmission reward to transmitter of a successful report, in 1e-9PHB units
   */
  function getBilling()
    external
    view
    returns (
      uint32 maximumGasPrice,
      uint32 reasonableGasPrice,
      uint32 microPhbPerEth,
      uint32 phbGweiPerObservation,
      uint32 phbGweiPerTransmission
    )
  {
    Billing memory billing = s_billing;
    return (
      billing.maximumGasPrice,
      billing.reasonableGasPrice,
      billing.microPhbPerEth,
      billing.phbGweiPerObservation,
      billing.phbGweiPerTransmission
    );
  }

  /**
   * @notice emitted when a new access-control contract is set
   * @param old the address prior to the current setting
   * @param current the address of the new access-control contract
   */
  event BillingAccessControllerSet(AccessControllerInterface old, AccessControllerInterface current);

  function setBillingAccessControllerInternal(AccessControllerInterface _billingAccessController)
    internal
  {
    AccessControllerInterface oldController = s_billingAccessController;
    if (_billingAccessController != oldController) {
      s_billingAccessController = _billingAccessController;
      emit BillingAccessControllerSet(
        oldController,
        _billingAccessController
      );
    }
  }

  /**
   * @notice sets billingAccessController
   * @param _billingAccessController new billingAccessController contract address
   * @dev only owner can call this
   */
  function setBillingAccessController(AccessControllerInterface _billingAccessController)
    external
    onlyOwner
  {
    setBillingAccessControllerInternal(_billingAccessController);
  }

  /**
   * @notice gets billingAccessController
   * @return address of billingAccessController contract
   */
  function billingAccessController()
    external
    view
    returns (AccessControllerInterface)
  {
    return s_billingAccessController;
  }

  /**
   * @notice withdraws an oracle's payment from the contract
   * @param _transmitter the transmitter address of the oracle
   * @dev must be called by oracle's payee address
   */
  function withdrawPayment(address _transmitter)
    external
  {
    require(msg.sender == s_payees[_transmitter], "Only payee can withdraw");
    payOracle(_transmitter);
  }

  /**
   * @notice query an oracle's payment amount
   * @param _transmitter the transmitter address of the oracle
   */
  function owedPayment(address _transmitter)
    public
    view
    returns (uint256)
  {
    Oracle memory oracle = s_oracles[_transmitter];
    if (oracle.role == Role.Unset) { return 0; }
    Billing memory billing = s_billing;
    uint256 phbWeiAmount =
      uint256(s_oracleObservationsCounts[oracle.index] - 1) *
      uint256(billing.phbGweiPerObservation) *
      (1 gwei);
    phbWeiAmount += s_gasReimbursementsPhbWei[oracle.index] - 1;
    return phbWeiAmount;
  }

  /**
   * @notice emitted when an oracle has been paid PHB
   * @param transmitter address from which the oracle sends reports to the transmit method
   * @param payee address to which the payment is sent
   * @param amount amount of PHB sent
   * @param phbToken address of the PHB token contract
   */
  event OraclePaid(
    address indexed transmitter,
    address indexed payee,
    uint256 amount,
    PhbTokenInterface indexed phbToken
  );

  // payOracle pays out _transmitter's balance to the corresponding payee, and zeros it out
  function payOracle(address _transmitter)
    internal
  {
    Oracle memory oracle = s_oracles[_transmitter];
    uint256 phbWeiAmount = owedPayment(_transmitter);
    if (phbWeiAmount > 0) {
      address payee = s_payees[_transmitter];
      // Poses no re-entrancy issues, because PHB.transfer does not yield
      // control flow.
      require(s_phbToken.transfer(payee, phbWeiAmount), "insufficient funds");
      s_oracleObservationsCounts[oracle.index] = 1; // "zero" the counts. see var's docstring
      s_gasReimbursementsPhbWei[oracle.index] = 1; // "zero" the counts. see var's docstring
      emit OraclePaid(_transmitter, payee, phbWeiAmount, s_phbToken);
    }
  }

  // payOracles pays out all transmitters, and zeros out their balances.
  //
  // It's much more gas-efficient to do this as a single operation, to avoid
  // hitting storage too much.
  function payOracles()
    internal
  {
    Billing memory billing = s_billing;
    PhbTokenInterface phbToken = s_phbToken;
    uint16[maxNumOracles] memory observationsCounts = s_oracleObservationsCounts;
    uint256[maxNumOracles] memory gasReimbursementsPhbWei =
      s_gasReimbursementsPhbWei;
    address[] memory transmitters = s_transmitters;
    for (uint transmitteridx = 0; transmitteridx < transmitters.length; transmitteridx++) {
      uint256 reimbursementAmountPhbWei = gasReimbursementsPhbWei[transmitteridx] - 1;
      uint256 obsCount = observationsCounts[transmitteridx] - 1;
      uint256 phbWeiAmount =
        obsCount * uint256(billing.phbGweiPerObservation) * (1 gwei) + reimbursementAmountPhbWei;
      if (phbWeiAmount > 0) {
          address payee = s_payees[transmitters[transmitteridx]];
          // Poses no re-entrancy issues, because PHB.transfer does not yield
          // control flow.
          require(phbToken.transfer(payee, phbWeiAmount), "insufficient funds");
          observationsCounts[transmitteridx] = 1;       // "zero" the counts.
          gasReimbursementsPhbWei[transmitteridx] = 1; // "zero" the counts.
          emit OraclePaid(transmitters[transmitteridx], payee, phbWeiAmount, phbToken);
        }
    }
    // "Zero" the accounting storage variables
    s_oracleObservationsCounts = observationsCounts;
    s_gasReimbursementsPhbWei = gasReimbursementsPhbWei;
  }

  function oracleRewards(
    bytes memory observers,
    uint16[maxNumOracles] memory observations
  )
    internal
    pure
    returns (uint16[maxNumOracles] memory)
  {
    // reward each observer-participant with the observer reward
    for (uint obsIdx = 0; obsIdx < observers.length; obsIdx++) {
      uint8 observer = uint8(observers[obsIdx]);
      observations[observer] = saturatingAddUint16(observations[observer], 1);
    }
    return observations;
  }

  // This value needs to change if maxNumOracles is increased, or the accounting
  // calculations at the bottom of reimburseAndRewardOracles change.
  //
  // To recalculate it, run the profiler as described in
  // ../../profile/README.md, and add up the gas-usage values reported for the
  // lines in reimburseAndRewardOracles following the "gasLeft = gasleft()"
  // line. E.g., you will see output like this:
  //
  //      7        uint256 gasLeft = gasleft();
  //     29        uint256 gasCostEthWei = transmitterGasCostEthWei(
  //      9          uint256(initialGas),
  //      3          gasPrice,
  //      3          callDataGasCost,
  //      3          gasLeft
  //      .
  //      .
  //      .
  //     59        uint256 gasCostPhbWei = (gasCostEthWei * billing.microPhbPerEth)/ 1e6;
  //      .
  //      .
  //      .
  //   5047        s_gasReimbursementsPhbWei[txOracle.index] =
  //    856          s_gasReimbursementsPhbWei[txOracle.index] + gasCostPhbWei +
  //     26          uint256(billing.phbGweiPerTransmission) * (1 gwei);
  //
  // If those were the only lines to be accounted for, you would add up
  // 29+9+3+3+3+59+5047+856+26=6035.
  uint256 internal constant accountingGasCost = 6035;

  // Uncomment the following declaration to compute the remaining gas cost after
  // above gasleft(). (This must exist in a base class to OffchainAggregator, so
  // it can't go in TestOffchainAggregator.)
  //
  // uint256 public gasUsedInAccounting;

  // Gas price at which the transmitter should be reimbursed, in ETH-gwei/gas
  function impliedGasPrice(
    uint256 txGasPrice,         // ETH-gwei/gas units
    uint256 reasonableGasPrice, // ETH-gwei/gas units
    uint256 maximumGasPrice     // ETH-gwei/gas units
  )
    internal
    pure
    returns (uint256)
  {
    // Reward the transmitter for choosing an efficient gas price: if they manage
    // to come in lower than considered reasonable, give them half the savings.
    //
    // The following calculations are all in units of gwei/gas, i.e. 1e-9ETH/gas
    uint256 gasPrice = txGasPrice;
    if (txGasPrice < reasonableGasPrice) {
      // Give transmitter half the savings for coming in under the reasonable gas price
      gasPrice += (reasonableGasPrice - txGasPrice) / 2;
    }
    // Don't reimburse a gas price higher than maximumGasPrice
    return min(gasPrice, maximumGasPrice);
  }

  // gas reimbursement due the transmitter, in ETH-wei
  //
  // If this function is changed, accountingGasCost needs to change, too. See
  // its docstring
  function transmitterGasCostEthWei(
    uint256 initialGas,
    uint256 gasPrice, // ETH-gwei/gas units
    uint256 callDataCost, // gas units
    uint256 gasLeft
  )
    internal
    pure
    returns (uint128 gasCostEthWei)
  {
    require(initialGas >= gasLeft, "gasLeft cannot exceed initialGas");
    uint256 gasUsed = // gas units
      initialGas - gasLeft + // observed gas usage
      callDataCost + accountingGasCost; // estimated gas usage
    // gasUsed is in gas units, gasPrice is in ETH-gwei/gas units; convert to ETH-wei
    uint256 fullGasCostEthWei = gasUsed * gasPrice * (1 gwei);
    assert(fullGasCostEthWei < maxUint128); // the entire ETH supply fits in a uint128...
    return uint128(fullGasCostEthWei);
  }

  /**
   * @notice withdraw any available funds left in the contract, up to _amount, after accounting for the funds due to participants in past reports
   * @param _recipient address to send funds to
   * @param _amount maximum amount to withdraw, denominated in PHB-wei.
   * @dev access control provided by billingAccessController
   */
  function withdrawFunds(address _recipient, uint256 _amount)
    external
  {
    require(msg.sender == owner || s_billingAccessController.hasAccess(msg.sender, msg.data),
      "Only owner&billingAdmin can call");
    uint256 phbDue = totalPHBDue();
    uint256 phbBalance = s_phbToken.balanceOf(address(this));
    require(phbBalance >= phbDue, "insufficient balance");
    require(s_phbToken.transfer(_recipient, min(phbBalance - phbDue, _amount)), "insufficient funds");
  }

  // Total PHB due to participants in past reports.
  function totalPHBDue()
    internal
    view
    returns (uint256 phbDue)
  {
    // Argument for overflow safety: We do all computations in
    // uint256s. The inputs to phbDue are:
    // - the <= 31 observation rewards each of which has less than
    //   64 bits (32 bits for billing.phbGweiPerObservation, 32 bits
    //   for wei/gwei conversion). Hence 69 bits are sufficient for this part.
    // - the <= 31 gas reimbursements, each of which consists of at most 166
    //   bits (see s_gasReimbursementsPhbWei docstring). Hence 171 bits are
    //   sufficient for this part
    // In total, 172 bits are enough.
    uint16[maxNumOracles] memory observationCounts = s_oracleObservationsCounts;
    for (uint i = 0; i < maxNumOracles; i++) {
      phbDue += observationCounts[i] - 1; // Stored value is one greater than actual value
    }
    Billing memory billing = s_billing;
    // Convert phbGweiPerObservation to uint256, or this overflows!
    phbDue *= uint256(billing.phbGweiPerObservation) * (1 gwei);
    address[] memory transmitters = s_transmitters;
    uint256[maxNumOracles] memory gasReimbursementsPhbWei =
      s_gasReimbursementsPhbWei;
    for (uint i = 0; i < transmitters.length; i++) {
      phbDue += uint256(gasReimbursementsPhbWei[i]-1); // Stored value is one greater than actual value
    }
  }

  /**
   * @notice allows oracles to check that sufficient PHB balance is available
   * @return availableBalance PHB available on this contract, after accounting for outstanding obligations. can become negative
   */
  function phbAvailableForPayment()
    external
    view
    returns (int256 availableBalance)
  {
    // there are at most one billion PHB, so this cast is safe
    int256 balance = int256(s_phbToken.balanceOf(address(this)));
    // according to the argument in the definition of totalPHBDue,
    // totalPHBDue is never greater than 2**172, so this cast is safe
    int256 due = int256(totalPHBDue());
    // safe from overflow according to above sizes
    return int256(balance) - int256(due);
  }

  /**
   * @notice number of observations oracle is due to be reimbursed for
   * @param _signerOrTransmitter address used by oracle for signing or transmitting reports
   */
  function oracleObservationCount(address _signerOrTransmitter)
    external
    view
    returns (uint16)
  {
    Oracle memory oracle = s_oracles[_signerOrTransmitter];
    if (oracle.role == Role.Unset) { return 0; }
    return s_oracleObservationsCounts[oracle.index] - 1;
  }


  function reimburseAndRewardOracles(
    uint32 initialGas,
    bytes memory observers
  )
    internal
  {
    Oracle memory txOracle = s_oracles[msg.sender];
    Billing memory billing = s_billing;
    // Reward oracles for providing observations. Oracles are not rewarded
    // for providing signatures, because signing is essentially free.
    s_oracleObservationsCounts =
      oracleRewards(observers, s_oracleObservationsCounts);
    // Reimburse transmitter of the report for gas usage
    require(txOracle.role == Role.Transmitter,
      "sent by undesignated transmitter"
    );
    uint256 gasPrice = impliedGasPrice(
      tx.gasprice / (1 gwei), // convert to ETH-gwei units
      billing.reasonableGasPrice,
      billing.maximumGasPrice
    );
    // The following is only an upper bound, as it ignores the cheaper cost for
    // 0 bytes. Safe from overflow, because calldata just isn't that long.
    uint256 callDataGasCost = 16 * msg.data.length;
    // If any changes are made to subsequent calculations, accountingGasCost
    // needs to change, too.
    uint256 gasLeft = gasleft();
    uint256 gasCostEthWei = transmitterGasCostEthWei(
      uint256(initialGas),
      gasPrice,
      callDataGasCost,
      gasLeft
    );

    // microPhbPerEth is 1e-6PHB/ETH units, gasCostEthWei is 1e-18ETH units
    // (ETH-wei), product is 1e-24PHB-wei units, dividing by 1e6 gives
    // 1e-18PHB units, i.e. PHB-wei units
    // Safe from over/underflow, since all components are non-negative,
    // gasCostEthWei will always fit into uint128 and microPhbPerEth is a
    // uint32 (128+32 < 256!).
    uint256 gasCostPhbWei = (gasCostEthWei * billing.microPhbPerEth)/ 1e6;

    // Safe from overflow, because gasCostPhbWei < 2**160 and
    // billing.phbGweiPerTransmission * (1 gwei) < 2**64 and we increment
    // s_gasReimbursementsPhbWei[txOracle.index] at most 2**40 times.
    s_gasReimbursementsPhbWei[txOracle.index] =
      s_gasReimbursementsPhbWei[txOracle.index] + gasCostPhbWei +
      uint256(billing.phbGweiPerTransmission) * (1 gwei); // convert from phbGwei to phbWei

    // Uncomment next line to compute the remaining gas cost after above gasleft().
    // See OffchainAggregatorBilling.accountingGasCost docstring for more information.
    //
    // gasUsedInAccounting = gasLeft - gasleft();
  }

  /*
   * Payee management
   */

  /**
   * @notice emitted when a transfer of an oracle's payee address has been initiated
   * @param transmitter address from which the oracle sends reports to the transmit method
   * @param current the payeee address for the oracle, prior to this setting
   * @param proposed the proposed new payee address for the oracle
   */
  event PayeeshipTransferRequested(
    address indexed transmitter,
    address indexed current,
    address indexed proposed
  );

  /**
   * @notice emitted when a transfer of an oracle's payee address has been completed
   * @param transmitter address from which the oracle sends reports to the transmit method
   * @param current the payeee address for the oracle, prior to this setting
   */
  event PayeeshipTransferred(
    address indexed transmitter,
    address indexed previous,
    address indexed current
  );

  /**
   * @notice sets the payees for transmitting addresses
   * @param _transmitters addresses oracles use to transmit the reports
   * @param _payees addresses of payees corresponding to list of transmitters
   * @dev must be called by owner
   * @dev cannot be used to change payee addresses, only to initially populate them
   */
  function setPayees(
    address[] calldata _transmitters,
    address[] calldata _payees
  )
    external
    onlyOwner()
  {
    require(_transmitters.length == _payees.length, "transmitters.size != payees.size");

    for (uint i = 0; i < _transmitters.length; i++) {
      address transmitter = _transmitters[i];
      address payee = _payees[i];
      address currentPayee = s_payees[transmitter];
      bool zeroedOut = currentPayee == address(0);
      require(zeroedOut || currentPayee == payee, "payee already set");
      s_payees[transmitter] = payee;

      if (currentPayee != payee) {
        emit PayeeshipTransferred(transmitter, currentPayee, payee);
      }
    }
  }

  /**
   * @notice first step of payeeship transfer (safe transfer pattern)
   * @param _transmitter transmitter address of oracle whose payee is changing
   * @param _proposed new payee address
   * @dev can only be called by payee address
   */
  function transferPayeeship(
    address _transmitter,
    address _proposed
  )
    external
  {
      require(msg.sender == s_payees[_transmitter], "only current payee can update");
      require(msg.sender != _proposed, "cannot transfer to self");

      address previousProposed = s_proposedPayees[_transmitter];
      s_proposedPayees[_transmitter] = _proposed;

      if (previousProposed != _proposed) {
        emit PayeeshipTransferRequested(_transmitter, msg.sender, _proposed);
      }
  }

  /**
   * @notice second step of payeeship transfer (safe transfer pattern)
   * @param _transmitter transmitter address of oracle whose payee is changing
   * @dev can only be called by proposed new payee address
   */
  function acceptPayeeship(
    address _transmitter
  )
    external
  {
    require(msg.sender == s_proposedPayees[_transmitter], "only proposed payees can accept");

    address currentPayee = s_payees[_transmitter];
    s_payees[_transmitter] = msg.sender;
    s_proposedPayees[_transmitter] = address(0);

    emit PayeeshipTransferred(_transmitter, currentPayee, msg.sender);
  }

  /*
   * Helper functions
   */

  function saturatingAddUint16(uint16 _x, uint16 _y)
    internal
    pure
    returns (uint16)
  {
    return uint16(min(uint256(_x)+uint256(_y), maxUint16));
  }

  function min(uint256 a, uint256 b)
    internal
    pure
    returns (uint256)
  {
    if (a < b) { return a; }
    return b;
  }
}
