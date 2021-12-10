package networking

import (
	"fmt"
)

type NetworkingStack uint8

const (
	_ NetworkingStack = iota
	NetworkingStackV1
	NetworkingStackV2
	NetworkingStackV1V2
)

func (n NetworkingStack) needsv2() bool {
	return n == NetworkingStackV2 || n == NetworkingStackV1V2
}

func (n NetworkingStack) needsv1() bool {
	return n == NetworkingStackV1 || n == NetworkingStackV1V2
}

func (n NetworkingStack) MarshalText() (text []byte, err error) {
	switch n {
	case NetworkingStackV1:
		return []byte("V1"), nil
	case NetworkingStackV2:
		return []byte("V2"), nil
	case NetworkingStackV1V2:
		return []byte("V1V2"), nil
	}
	return nil, fmt.Errorf("unknown NetworkingStack %v", n)
}

func (n *NetworkingStack) UnmarshalText(text []byte) error {
	switch string(text) {
	case "V1":
		*n = NetworkingStackV1
	case "V2":
		*n = NetworkingStackV2
	case "V1V2":
		*n = NetworkingStackV1V2
	default:
		return fmt.Errorf("cannot unmarshal %s as NetworkingStack", text)
	}
	return nil
}
