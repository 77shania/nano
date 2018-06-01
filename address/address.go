package address

import (
	"bytes"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"errors"

	"github.com/frankh/nano/types"
	"github.com/frankh/nano/utils"
	"github.com/golang/crypto/blake2b"
	// We've forked golang's ed25519 implementation
	// to use blake2b instead of sha3
	"github.com/frankh/crypto/ed25519"
)

// nano uses a non-standard base32 character set.
const EncodeNano = "13456789abcdefghijkmnopqrstuwxyz"

var NanoEncoding = base32.NewEncoding(EncodeNano)

func ValidateAddress(account types.Account) bool {
	_, err := AddressToPub(account)

	return err == nil
}

func AddressToPub(account types.Account) (public_key []byte, err error) {
	address := string(account)

	if address[:4] == "xrb_" {
		address = address[4:]
	} else if address[:5] == "nano_" {
		address = address[5:]
	} else {
		return nil, errors.New("Invalid address format")
	}
	// A valid nano address is 64 bytes long
	// First 5 are simply a hard-coded string nano_ for ease of use
	// The following 52 characters form the address, and the final
	// 8 are a checksum.
	// They are base 32 encoded with a custom encoding.
	if len(address) == 60 {
		// The nano address string is 260bits which doesn't fall on a
		// byte boundary. pad with zeros to 280bits.
		// (zeros are encoded as 1 in nano's 32bit alphabet)
		key_b32nano := "1111" + address[0:52]
		input_checksum := address[52:]

		key_bytes, err := NanoEncoding.DecodeString(key_b32nano)
		if err != nil {
			return nil, err
		}
		// strip off upper 24 bits (3 bytes). 20 padding was added by us,
		// 4 is unused as account is 256 bits.
		key_bytes = key_bytes[3:]

		// nano checksum is calculated by hashing the key and reversing the bytes
		valid := NanoEncoding.EncodeToString(GetAddressChecksum(key_bytes)) == input_checksum
		if valid {
			return key_bytes, nil
		} else {
			return nil, errors.New("Invalid address checksum")
		}
	}

	return nil, errors.New("Invalid address format")
}

func GetAddressChecksum(pub ed25519.PublicKey) []byte {
	hash, err := blake2b.New(5, nil)
	if err != nil {
		panic("Unable to create hash")
	}

	hash.Write(pub)
	return utils.Reversed(hash.Sum(nil))
}

func PubKeyToAddress(pub ed25519.PublicKey) types.Account {
	// Pubkey is 256bits, base32 must be multiple of 5 bits
	// to encode properly.
	// Pad the start with 0's and strip them off after base32 encoding
	padded := append([]byte{0, 0, 0}, pub...)
	address := NanoEncoding.EncodeToString(padded)[4:]
	checksum := NanoEncoding.EncodeToString(GetAddressChecksum(pub))

	return types.Account("nano_" + address + checksum)
}

func KeypairFromPrivateKey(private_key string) (ed25519.PublicKey, ed25519.PrivateKey) {
	private_bytes, _ := hex.DecodeString(private_key)
	pub, priv, _ := ed25519.GenerateKey(bytes.NewReader(private_bytes))

	return pub, priv
}

func KeypairFromSeed(seed string, index uint32) (ed25519.PublicKey, ed25519.PrivateKey) {
	// This seems to be the standard way of producing wallets.

	// We hash together the seed with an address index and use
	// that as the private key. Whenever you "add" an address
	// to your wallet the wallet software increases the index
	// and generates a new address.
	hash, err := blake2b.New(32, nil)
	if err != nil {
		panic("Unable to create hash")
	}

	seed_data, err := hex.DecodeString(seed)
	if err != nil {
		panic("Invalid seed")
	}

	bs := make([]byte, 4)
	binary.BigEndian.PutUint32(bs, index)

	hash.Write(seed_data)
	hash.Write(bs)

	seed_bytes := hash.Sum(nil)
	pub, priv, err := ed25519.GenerateKey(bytes.NewReader(seed_bytes))

	if err != nil {
		panic("Unable to generate ed25519 key")
	}

	return pub, priv
}

func GenerateKey() (ed25519.PublicKey, ed25519.PrivateKey) {
	pubkey, privkey, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic("Unable to generate ed25519 key")
	}

	return pubkey, privkey
}
