package contract

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
	"time"

	"github.com/MetaBloxIO/metablox-foundation-services/models"
	"github.com/MetaBloxIO/metablox-foundation-services/registry"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const deployedContract = "0xf880b97Be7c402Cc441895bF397c3f865BfE1Cb2"
const network = "wss://ws.s0.b.hmny.io"

var client *ethclient.Client
var instance *registry.Registry
var contractAddress common.Address

func Init() error {
	var err error
	client, err = ethclient.Dial(network)
	if err != nil {
		return err
	}
	contractAddress = common.HexToAddress(deployedContract)
	instance, err = registry.NewRegistry(contractAddress, client)
	if err != nil {
		return err
	}

	return nil
}

func createSignatureFromMessage(messageBytes []byte, privateKey *ecdsa.PrivateKey) ([32]byte, [32]byte, uint8, error) {
	messageHash := crypto.Keccak256Hash(messageBytes)

	comboHash := crypto.Keccak256Hash([]byte("\x19Ethereum Signed Message:\n32"), messageHash.Bytes())
	signature, err := crypto.Sign(comboHash[:], privateKey)
	if err != nil {
		return [32]byte{}, [32]byte{}, 0, err
	}
	var r [32]byte
	var s [32]byte
	var v uint8

	copy(r[:], signature[:32])
	copy(s[:], signature[32:64])
	v = signature[64] + 27 //have to increment this manually as the smart contract expects v to be 27 or 28, while the crypto package generates it as 0 or 1

	return r, s, v, nil
}

func generateAuth(privateKey *ecdsa.PrivateKey) (*bind.TransactOpts, error) {
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(1666700000))
	if err != nil {
		return nil, err
	}
	authNonce, err := client.PendingNonceAt(context.Background(), crypto.PubkeyToAddress(privateKey.PublicKey))
	if err != nil {
		return nil, err
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return nil, err
	}
	auth.Nonce = big.NewInt(int64(authNonce))
	auth.Value = big.NewInt(0)
	auth.GasLimit = uint64(300000)
	auth.GasPrice = gasPrice
	return auth, nil
}

func RegisterVCIssuer(credentialKey, did string, privateKey *ecdsa.PrivateKey) error {
	didAccount, err := instance.Dids(nil, did)
	if err != nil {
		return err
	}

	nonce, err := instance.Nonce(nil, didAccount)
	if err != nil {
		return err
	}

	var messageBytes []byte

	messageBytes = bytes.Join([][]byte{messageBytes, []byte(credentialKey), []byte(did), []byte(nonce.String()), []byte("createVc")}, nil)
	r, s, v, err := createSignatureFromMessage(messageBytes, privateKey)
	if err != nil {
		return err
	}

	auth, err := generateAuth(privateKey)
	if err != nil {
		return err
	}

	tx, err := instance.CreateVcDef(auth, credentialKey, did, v, r, s)
	if err != nil {
		return err
	}

	fmt.Println("transaction address: ", tx.Hash().Hex())
	return nil
}

func UpdateVCValue(credentialKey, fieldName, newValue string, privateKey *ecdsa.PrivateKey) error {
	ownerDid, err := instance.VcIssuers(nil, credentialKey)
	if err != nil {
		return err
	}

	ownerAccount, err := instance.Dids(nil, ownerDid)
	if err != nil {
		return err
	}

	nonce, err := instance.Nonce(nil, ownerAccount)
	if err != nil {
		return err
	}

	var messageBytes []byte
	var fieldBytes [32]byte

	copy(fieldBytes[:], []byte(fieldName))

	messageBytes = bytes.Join([][]byte{messageBytes, []byte(credentialKey), []byte(nonce.String()), []byte("setVcAttribute"), fieldBytes[:], []byte(newValue)}, nil)
	r, s, v, err := createSignatureFromMessage(messageBytes, privateKey)
	if err != nil {
		return err
	}

	auth, err := generateAuth(privateKey)
	if err != nil {
		return err
	}

	tx, err := instance.SetVcAttributeSigned(auth, credentialKey, v, r, s, fieldBytes, []byte(newValue))
	if err != nil {
		return err
	}

	fmt.Println("transaction address: ", tx.Hash().Hex())
	return nil
}

func ReadVCChangedEvents(credentialKey string) error {
	ownerDid, err := instance.VcIssuers(nil, credentialKey)
	if err != nil {
		return err
	}

	ownerAccount, err := instance.Dids(nil, ownerDid)
	if err != nil {
		return err
	}

	targetBlock, err := instance.Changed(nil, ownerAccount)
	if err != nil || targetBlock == nil {
		return err
	}

	end := new(uint64)
	*end = targetBlock.Uint64()

	filterOpts := &bind.FilterOpts{Context: context.Background(), Start: targetBlock.Uint64() - 50, End: end}
	itr, err := instance.FilterVCSchemaChanged(filterOpts, []string{credentialKey})
	if err != nil {
		return err
	}
	if itr.Error() != nil {
		return itr.Error()
	}

	// Loop over all found events
	for itr.Next() {
		event := itr.Event
		fmt.Println(event.VcName.Hex())
		fmt.Println(event.Name)
		fmt.Println(common.Bytes2Hex(event.Value))
	}

	return nil
}

func CreateVC(vc *models.VerifiableCredential, did string, privateKey *ecdsa.PrivateKey) error {
	/*	fromAddress := crypto.PubkeyToAddress(foundationPrivateKey.PublicKey)	//todo: uncomment once smart contract is ready
		nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
		if err != nil {
			log.Fatal(err)
		}

		gasPrice, err := client.SuggestGasPrice(context.Background())
		if err != nil {
			log.Fatal(err)
		}
		auth := bind.NewKeyedTransactor(foundationPrivateKey)
		auth.Nonce = big.NewInt(int64(nonce))
		auth.Value = big.NewInt(0)     // in wei
		auth.GasLimit = uint64(300000) // in units
		auth.GasPrice = gasPrice
		_, err = instance.UploadVC(auth, vcBytes)
		if err != nil {
			return err
		}*/

	return nil
}

func RenewVC(vcBytes [32]byte) error {
	/*	fromAddress := crypto.PubkeyToAddress(foundationPrivateKey.PublicKey)	//todo: uncomment once smart contract is ready
		nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
		if err != nil {
			log.Fatal(err)
		}

		gasPrice, err := client.SuggestGasPrice(context.Background())
		if err != nil {
			log.Fatal(err)
		}
		auth := bind.NewKeyedTransactor(foundationPrivateKey)
		auth.Nonce = big.NewInt(int64(nonce))
		auth.Value = big.NewInt(0)     // in wei
		auth.GasLimit = uint64(300000) // in units
		auth.GasPrice = gasPrice
		_, err = instance.RenewVC(auth, vcBytes)
		if err != nil {
			return err
		}*/

	return nil
}

func RevokeVC(vcBytes [32]byte) error {
	/*	fromAddress := crypto.PubkeyToAddress(foundationPrivateKey.PublicKey)	//todo: uncomment once smart contract is ready
		nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
		if err != nil {
			log.Fatal(err)
		}

		gasPrice, err := client.SuggestGasPrice(context.Background())
		if err != nil {
			log.Fatal(err)
		}
		auth := bind.NewKeyedTransactor(foundationPrivateKey)
		auth.Nonce = big.NewInt(int64(nonce))
		auth.Value = big.NewInt(0)     // in wei
		auth.GasLimit = uint64(300000) // in units
		auth.GasPrice = gasPrice
		_, err = instance.RevokeVC(auth, vcBytes)
		if err != nil {
			return err
		}*/

	return nil
}

func UploadDocument(document *models.DIDDocument, did string, privateKey *ecdsa.PrivateKey) error {
	fmt.Println(did)
	ownerAccount, _ := instance.Dids(nil, did)
	fmt.Println(ownerAccount.Hex())
	pubAddress := crypto.PubkeyToAddress(privateKey.PublicKey)
	nonce, err := instance.Nonce(nil, pubAddress)
	if err != nil {
		return err
	}
	var messageBytes []byte

	messageBytes = bytes.Join([][]byte{messageBytes, []byte(did), pubAddress.Bytes(), []byte(nonce.String()) /*nonceBytes[:]*/, []byte("register")}, nil)
	r, s, v, err := createSignatureFromMessage(messageBytes, privateKey)
	if err != nil {
		return err
	}

	auth, err := generateAuth(privateKey)
	if err != nil {
		return err
	}

	tx, err := instance.RegisterDid(auth, did, pubAddress, v, r, s)
	if err != nil {
		return err
	}

	fmt.Println("transaction address: ", tx.Hash().Hex())
	return nil
}

func GetDocument(targetDID string) (*models.DIDDocument, [32]byte, error) {
	address, err := instance.Dids(nil, targetDID)
	if err != nil {
		return nil, [32]byte{0}, err
	}

	document := new(models.DIDDocument)

	document.ID = "did:metablox:" + targetDID
	document.Context = make([]string, 0)
	document.Context = append(document.Context, models.ContextSecp256k1)
	document.Context = append(document.Context, models.ContextDID)
	document.Created = time.Now().Format(time.RFC3339) //todo: need to get this from contract
	document.Updated = document.Created                //todo: need to get this from contract
	document.Version = 1                               //todo: need to get this from contract

	VM := models.VerificationMethod{}
	VM.ID = document.ID + "#verification"
	VM.BlockchainAccountId = "eip155:1666600000:" + address.Hex()
	VM.Controller = document.ID
	VM.MethodType = models.Secp256k1Key

	document.VerificationMethod = append(document.VerificationMethod, VM)
	document.Authentication = VM.ID

	placeholderHash := [32]byte{94, 241, 27, 134, 190, 223, 112, 91, 189, 49, 221, 31, 228, 35, 189, 213, 251, 60, 60, 210, 162, 45, 151, 3, 31, 78, 41, 239, 41, 75, 198, 139}
	return document, placeholderHash, nil
}

func RegisterDID(register *models.RegisterDID, key *ecdsa.PrivateKey) (*types.Transaction, error) {
	userAddress := common.HexToAddress(register.Account)
	// todo: verify signature first better here

	auth, err := generateAuth(key)
	if err != nil {
		return nil, err
	}

	var r [32]byte
	copy(r[:], []byte(register.SigR)[:32])
	var s [32]byte
	copy(s[:], []byte(register.SigS)[:32])

	//pubAddress := crypto.PubkeyToAddress(key.PublicKey)
	//
	//msg := ethereum.CallMsg{
	//	From:  pubAddress,
	//	To:    &contractAddress,
	//	Value: big.NewInt(0),
	//}
	//
	//// EstimateGas
	//_, err = client.EstimateGas(context.Background(), msg)
	//if err != nil {
	//	return err
	//}

	tx, err := instance.RegisterDid(auth, register.Did, userAddress, register.SigV, r, s)
	if err != nil {
		return nil, err
	}

	fmt.Println("transaction hash: ", tx.Hash().Hex())
	return tx, nil
}
