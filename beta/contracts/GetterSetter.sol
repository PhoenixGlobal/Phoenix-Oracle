pragma solidity ^0.5.0;

contract GetterSetter {
    bytes32 public value;

    function setValue(bytes32 _value) public {
        value = _value;
    }
}
