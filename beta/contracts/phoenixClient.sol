pragma solidity ^0.5.0;

import "./zeppelin/Ownable.sol";

contract phoenixClient is Ownable{

    struct Callback {
        address addr;
        bytes4 fid;
    }

    uint private nonce;
    mapping(uint => Callback) private callbacks;

    event Request(
        uint indexed nonce,
        address indexed to,
        bytes4 indexed fid
    );

    function requestData(address _callbackAddress, bytes4 _callbackFID) public {
        Callback memory cb = Callback(_callbackAddress, _callbackFID);
        callbacks[nonce] = cb;
        emit Request(nonce, cb.addr, cb.fid);
        nonce += 1;
    }

    function fulfillData(uint256 _nonce, bytes32 _data)
    public
    onlyOwner
    hasNonce(_nonce)
    {
        Callback memory cb = callbacks[_nonce];
        (bool success, ) = cb.addr.call(abi.encodePacked(cb.fid, _data));
        require(success);
        delete callbacks[_nonce];
    }

    modifier hasNonce(uint256 _nonce) {
        require(callbacks[_nonce].addr != address(0));
        _;
    }
}