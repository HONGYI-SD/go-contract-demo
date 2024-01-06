package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	RPC_URL = "your rpc url"
	//私钥需要将0x前缀去掉
	PRIKEY   = "your private key"
	CONTRACT = "you contract address"
)

func callContract(client *ethclient.Client) (string, error) {
	contractAddr := common.HexToAddress(CONTRACT)
	contractAbi, err := abi.JSON(strings.NewReader(CONTRACT_ABI))
	if err != nil {
		return "", err
	}
	data, err := contractAbi.Pack("retrieve")
	if err != nil {
		return "", err
	}
	msg := ethereum.CallMsg{
		To:   &contractAddr,
		Data: data,
	}

	ctxTimeout, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	retData, err := client.CallContract(ctxTimeout, msg, nil)
	if err != nil {
		return "", err
	}
	unpackData, _ := contractAbi.Unpack("retrieve", retData)
	a := *abi.ConvertType(unpackData[0], new(big.Int)).(*big.Int)
	return a.String(), nil
}

func initSomeinfo(client *ethclient.Client) (*bind.BoundContract, common.Address, *bind.TransactOpts, error) {
	contractAddr := common.HexToAddress(CONTRACT)
	contractAbi, err := abi.JSON(strings.NewReader(CONTRACT_ABI))
	if err != nil {
		return nil, [20]byte{}, nil, fmt.Errorf(": %w", err)
	}

	//1. 创建BoundContract
	contract := bind.NewBoundContract(contractAddr, contractAbi, client, client, nil)
	//2. 获取chainID
	chainId, err := client.ChainID(context.Background())
	if err != nil {
		return nil, [20]byte{}, nil, fmt.Errorf(": %w", err)
	}
	//3.转换成ecdsa格式的私钥
	privateKey, err := crypto.HexToECDSA(PRIKEY)
	if err != nil {
		return nil, [20]byte{}, nil, fmt.Errorf("privateKey HexToECDSA failed: %w", err)
	}
	//4. ecdsa格式私钥-》ecdsa格式公钥-》钱包地址
	publicKey := privateKey.Public()
	publickKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, [20]byte{}, nil, fmt.Errorf(": %w", err)
	}
	fromAddr := crypto.PubkeyToAddress(*publickKeyECDSA)

	//5. 创建签名器
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainId)
	if err != nil {
		return nil, [20]byte{}, nil, fmt.Errorf(": %w", err)
	}
	return contract, fromAddr, auth, nil
}
func callContractEx(client *ethclient.Client, contract *bind.BoundContract) (string, error) {
	results := new([]interface{})
	err := contract.Call(&bind.CallOpts{}, results, "retrieve")
	if err != nil {
		return "", fmt.Errorf("call failed: %w", err)
	}
	a := *abi.ConvertType((*results)[0], new(big.Int)).(*big.Int)
	return a.String(), nil
}
func setContract(client *ethclient.Client, contract *bind.BoundContract, fromAddr common.Address, auth *bind.TransactOpts) (string, error) {
	transInfo, err := contract.Transact(&bind.TransactOpts{
		Context: context.Background(),
		From:    fromAddr,
		Signer:  auth.Signer,
	}, "store", new(big.Int).SetUint64(666))
	if err != nil {
		return "", fmt.Errorf(": %w", err)
	}

	fmt.Println("store ok, txHash:", transInfo.Hash().String())
	return transInfo.Hash().String(), nil
}
func main() {
	// connect to etherum goerli network
	client, err := ethclient.Dial(RPC_URL)
	if err != nil {
		fmt.Println("dial etherum network failed:", err)
		return
	}
	defer client.Close()
	fmt.Println("dial success")

	contract, fromAddr, auth, err := initSomeinfo(client)
	if err != nil {
		fmt.Println("initSomeinfo failed:", err)
		return
	}
	fmt.Println("initSomeinfo success")

	//调用合约的store接口，写入数据100
	txhash, err := setContract(client, contract, fromAddr, auth)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("call contract set function success")

	//根据交易hash查询receipt信息，网络确认需要一点时间，所以修改合约数据后最好等待一段时间再查
	time.Sleep(time.Second * 15)
	receipt, err := client.TransactionReceipt(context.Background(), common.HexToHash(txhash))
	if err != nil {
		fmt.Println("get transaction receipt failed:", err)
		return
	}
	fmt.Println("get receipt success: ", receipt.Status)

	// 第一种方法：通过ethClient的CallContract读取合约内容
	retval1, err := callContract(client)
	if err != nil {
		fmt.Println("callContract failed:", err)
		return
	}
	fmt.Println("1st way call contract success:", retval1)
	// 第二种方法：通过boundContract的call方法读取合约内容
	retval2, err := callContractEx(client, contract)
	if err != nil {
		fmt.Println("callContractEX failed")
		return
	}
	fmt.Println("2nd way call contract success:", retval2)
}

const CONTRACT_ABI = `
[
	{
		"inputs": [],
		"name": "retrieve",
		"outputs": [
			{
				"internalType": "uint256",
				"name": "",
				"type": "uint256"
			}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "uint256",
				"name": "num",
				"type": "uint256"
			}
		],
		"name": "store",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	}
]`
