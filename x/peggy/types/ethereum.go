package types

import (
	"bytes"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
)

const (
	// PeggyDenomPrefix indicates the prefix for all assests minted by this module
	PeggyDenomPrefix = ModuleName

	// PeggyDenomSeparator is the separator for peggy denoms
	PeggyDenomSeparator = ""

	// ETHContractAddressLen is the length of contract address bytes
	ETHContractAddressLen = 20

	// PeggyDenomLen is the length of the denoms generated by the peggy module
	PeggyDenomLen = len(PeggyDenomPrefix) + len(PeggyDenomSeparator) + ETHContractAddressLen
)

// EthAddrLessThan migrates the Ethereum address less than function
func EthAddrLessThan(e, o string) bool {
	return bytes.Compare([]byte(e)[:], []byte(o)[:]) == -1
}

// ValidateEthAddress validates the ethereum address strings
func ValidateEthAddress(address string) error {
	if address == "" {
		return fmt.Errorf("empty Ethereum address")
	}
	if !common.IsHexAddress(address) {
		return fmt.Errorf("%s is not a valid Ethereum address", address)
	}

	return nil
}

// NewERC20Token returns a new instance of an ERC20
func NewERC20Token(amount uint64, contract common.Address) *ERC20Token {
	return &ERC20Token{Amount: sdk.NewIntFromUint64(amount), Contract: contract.Hex()}
}

func NewSDKIntERC20Token(amount sdk.Int, contract common.Address) *ERC20Token {
	return &ERC20Token{Amount: amount, Contract: contract.Hex()}
}

// PeggyCoin returns the peggy representation of an ERC20 token
func (e *ERC20Token) PeggyCoin() sdk.Coin {
	return sdk.NewCoin(PeggyDenomString(common.HexToAddress(e.Contract)), e.Amount)
}

type PeggyDenom []byte

func (p PeggyDenom) String() string {
	contractAddress, err := p.TokenContract()
	if err != nil {
		// the case of unparseable peggy denom
		return fmt.Sprintf("%x(error: %s)", []byte(p), err.Error())
	}

	return PeggyDenomString(contractAddress)
}

func (p PeggyDenom) TokenContract() (common.Address, error) {
	fullPrefix := []byte(PeggyDenomPrefix + PeggyDenomSeparator)
	if !bytes.HasPrefix(p, fullPrefix) {
		err := errors.Errorf("denom '%x' byte prefix not equal to expected '%x'", []byte(p), fullPrefix)
		return common.Address{}, err
	}

	addressBytes := bytes.TrimPrefix(p, fullPrefix)
	if len(addressBytes) != ETHContractAddressLen {
		err := errors.Errorf("failed to validate Ethereum address bytes: %x", addressBytes)
		return common.Address{}, err
	}

	return common.BytesToAddress(addressBytes), nil
}

func NewPeggyDenom(tokenContract common.Address) PeggyDenom {
	buf := make([]byte, 0, PeggyDenomLen)
	buf = append(buf, PeggyDenomPrefix+PeggyDenomSeparator...)
	buf = append(buf, tokenContract.Bytes()...)

	return PeggyDenom(buf)
}

func NewPeggyDenomFromString(denom string) (PeggyDenom, error) {
	fullPrefix := PeggyDenomPrefix + PeggyDenomSeparator
	if !strings.HasPrefix(denom, fullPrefix) {
		err := errors.Errorf("denom '%s' string prefix not equal to expected '%s'", denom, fullPrefix)
		return nil, err
	}

	addressHex := strings.TrimPrefix(denom, fullPrefix)
	if err := ValidateEthAddress(addressHex); err != nil {
		return nil, err
	}

	peggyDenom := NewPeggyDenom(common.HexToAddress(addressHex))
	return peggyDenom, nil
}

func PeggyDenomString(tokenContract common.Address) string {
	return fmt.Sprintf("%s%s%s", PeggyDenomPrefix, PeggyDenomSeparator, tokenContract.Hex())
}

// ValidateBasic permforms stateless validation
func (e *ERC20Token) ValidateBasic() error {
	if err := ValidateEthAddress(e.Contract); err != nil {
		return sdkerrors.Wrap(err, "ethereum address")
	}

	if !e.PeggyCoin().IsValid() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidCoins, e.PeggyCoin().String())
	}

	if !e.PeggyCoin().IsPositive() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidCoins, e.PeggyCoin().String())
	}

	return nil
}

// Add adds one ERC20 to another
func (e *ERC20Token) Add(o *ERC20Token) (*ERC20Token, error) {
	if string(e.Contract) != string(o.Contract) {
		return nil, errors.New("invalid contract address")
	}

	sum := e.Amount.Add(o.Amount)
	if !sum.IsUint64() {
		return nil, errors.New("invalid amount")
	}

	return NewERC20Token(sum.Uint64(), common.HexToAddress(e.Contract)), nil
}