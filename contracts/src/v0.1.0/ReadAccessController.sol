// SPDX-License-Identifier: MIT
pragma solidity ^0.6.0;

import "./WriteAccessController.sol";

/**
 * @title ReadAccessController
 * @notice Gives access to:
 * - any externally owned account (note that offchain actors can always read
 * any contract storage regardless of onchain access control measures, so this
 * does not weaken the access control while improving usability)
 * - accounts explicitly added to an access list
 * @dev ReadAccessController is not suitable for access controlling writes
 * since it grants any externally owned account access! See
 * WriteAccessController for that.
 */
contract ReadAccessController is WriteAccessController {

  /**
   * @notice Returns the access of an address
   * @param _user The address to query
   */
  function hasAccess(
    address _user,
    bytes memory _calldata
  )
    public
    view
    virtual
    override
    returns (bool)
  {
    return super.hasAccess(_user, _calldata) || _user == tx.origin;
  }

}
