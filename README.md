# goether

A simple Ethereum wallet implementation and utilities in Golang.

## 项目简介

goether 是一个功能完整的以太坊钱包库，专为 Go 开发者设计。它提供了简洁易用的 API 来处理以太坊相关的操作，包括钱包管理、交易发送、智能合约交互等核心功能。

### 主要特性

- 🔐 **安全的私钥管理**: 支持从十六进制字符串、文件路径、助记词等多种方式创建钱包
- 💰 **完整的交易支持**: 支持 Legacy 交易和 EIP-1559 动态费用交易
- 📝 **智能合约交互**: 提供简单易用的合约调用和执行接口
- 🔏 **消息签名**: 支持标准消息签名和 EIP-712 类型化数据签名
- 🛠️ **实用工具**: 内置常用的以太坊工具函数
- 🌐 **多网络支持**: 支持主网、测试网等多种以太坊网络
- ⚡ **高性能**: 基于 go-ethereum 官方库，性能稳定可靠

### 支持的网络

- Ethereum Mainnet (Chain ID: 1)
- Ethereum Testnets (Goerli, Sepolia 等)
- Layer 2 网络 (Polygon, Arbitrum, Optimism 等)
- 私有网络和本地开发网络

## 安装

### 使用 Go Modules

```shell
go get -u github.com/go-enols/goether
```

### 系统要求

- Go 1.24.2 或更高版本
- 稳定的网络连接（用于连接以太坊节点）

### 依赖项

主要依赖包括：
- `github.com/ethereum/go-ethereum`: 以太坊官方 Go 客户端库
- `github.com/go-enols/ethrpc`: 以太坊 RPC 客户端

完整的依赖列表请查看 [go.mod](./go.mod) 文件。

## 快速开始

### 基本用法

```golang
package main

import (
    "fmt"
    "log"
    
    "github.com/ethereum/go-ethereum/common"
    "github.com/go-enols/goether"
)

func main() {
    // 创建钱包实例
    prvHex := "your_private_key_here"
    rpc := "https://mainnet.infura.io/v3/YOUR_PROJECT_ID"
    
    wallet, err := goether.NewWallet(prvHex, rpc)
    if err != nil {
        log.Fatal(err)
    }
    
    // 获取钱包地址
    fmt.Printf("钱包地址: %s\n", wallet.Address.Hex())
    
    // 获取余额
    balance, err := wallet.GetBalance()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("余额: %s ETH\n", balance.String())
}
```

### 更多示例

详细的使用示例请查看 [example](./example) 目录，包含：
- [钱包发送交易示例](./example/wallet_send_test.go)
- [智能合约交互示例](./example/contract_test.go)
- [从文件加载私钥示例](./example/wallet_from_path_test.go)

## 详细使用说明

### 发送以太坊交易

#### 基本交易发送

```golang
prvHex := "your_private_key"
rpc := "https://mainnet.infura.io/v3/YOUR_PROJECT_ID"

// 创建钱包
testWallet, err := goether.NewWallet(prvHex, rpc)
if err != nil {
    panic(err)
}

// 发送基本交易（自动使用 EIP-1559 动态费用）
txHash, err := testWallet.SendTx(
    common.HexToAddress("0xa06b79E655Db7D7C3B3E7B2ccEEb068c3259d0C9"), // 接收地址
    goether.EthToBN(0.12), // 发送金额（ETH）
    []byte("Hello Ethereum"), // 附加数据
    nil) // 使用默认交易选项

if err != nil {
    panic(err)
}
fmt.Printf("交易哈希: %s\n", txHash)
```

#### 自定义交易参数

```golang
// 手动设置交易参数
nonce := int(1)
gasLimit := int(21000)

txHash, err := testWallet.SendTx(
    common.HexToAddress("0xa06b79E655Db7D7C3B3E7B2ccEEb068c3259d0C9"),
    goether.EthToBN(0.12),
    []byte("Custom transaction"),
    //也可以使用nil自动配置
    &goether.TxOpts{
        Nonce:    &nonce,                    // 手动设置 nonce
        GasLimit: &gasLimit,               // 设置 Gas 限制
        GasPrice: goether.GweiToBN(20),    // Legacy 交易的 Gas 价格
    })
```

#### 发送 Legacy 交易

```golang
// 强制使用 Legacy 交易类型
txHash, err := testWallet.SendLegacyTx(
    common.HexToAddress("0xa06b79E655Db7D7C3B3E7B2ccEEb068c3259d0C9"),
    goether.EthToBN(0.1),
    []byte("Legacy transaction"),
    //也可以使用nil自动配置
    &goether.TxOpts{
        GasLimit: &gasLimit,
        GasPrice: goether.GweiToBN(15),
    })
```

### 智能合约交互

#### 创建合约实例

```golang
// ERC20 代币合约 ABI（简化版）
abi := `[
    {
        "constant": true,
        "inputs": [{"name": "", "type": "address"}],
        "name": "balanceOf",
        "outputs": [{"name": "", "type": "uint256"}],
        "payable": false,
        "stateMutability": "view",
        "type": "function"
    },
    {
        "constant": false,
        "inputs": [
            {"name": "dst", "type": "address"},
            {"name": "wad", "type": "uint256"}
        ],
        "name": "transfer",
        "outputs": [{"name": "", "type": "bool"}],
        "payable": false,
        "stateMutability": "nonpayable",
        "type": "function"
    }
]`

contractAddr := common.HexToAddress("0x123456789")
prvHex := "your_private_key"
rpc := "https://mainnet.infura.io/v3/YOUR_PROJECT_ID"

// 创建钱包和合约实例
testWallet, err := goether.NewWallet(prvHex, rpc)
if err != nil {
    panic(err)
}

testContract, err := goether.NewContract(contractAddr, abi, rpc, testWallet)
if err != nil {
    panic(err)
}
```

#### 调用只读方法

```golang
// 查询 ERC20 代币余额
balance, err := testContract.CallMethod(
    "balanceOf",                                                    // 方法名
    "latest",                                                       // 区块标签
    common.HexToAddress("0x123456789")) // 参数

if err != nil {
    panic(err)
}
fmt.Printf("代币余额: %s\n", balance)
```

#### 执行状态改变方法

```golang
// 转账 ERC20 代币
nonce := int(1)
gasLimit := int(60000)

txHash, err := testContract.ExecMethod(
    "transfer",                                                      // 方法名
    &goether.TxOpts{                                                // 交易选项
        Nonce:    &nonce,
        GasLimit: &gasLimit,
        GasPrice: goether.GweiToBN(20),
    },
    common.HexToAddress("0x123456879"), // 接收地址
    big.NewInt(100))                                                 // 转账数量

if err != nil {
    panic(err)
}
fmt.Printf("交易哈希: %s\n", txHash)
```

#### 编码和解码合约数据

```golang
// 编码方法调用数据
data, err := testContract.EncodeData("transfer", 
    common.HexToAddress("0x123456879"),
    big.NewInt(100))

// 编码为十六进制字符串
dataHex, err := testContract.EncodeDataHex("transfer",
    common.HexToAddress("0x123456879"),
    big.NewInt(100))

// 解码返回数据
result, err := testContract.DecodeData("balanceOf", responseData)

// 解码事件日志
event, err := testContract.DecodeEvent("Transfer", logData)
```

## API 文档

### Signer 模块

以太坊账户签名器，用于签名消息和交易。

#### 创建签名器

```golang
// 从私钥创建
signer, err := goether.NewSigner("your_private_key_hex")

// 从文件加载私钥
signer, err := goether.NewSignerFromPath("/path/to/private_key.txt")

// 从助记词创建（如果支持）
signer, err := goether.NewSignerFromMnemonic("your mnemonic phrase")
```

#### 主要方法

- ✅ **NewSigner(prvHex string)**: 从十六进制私钥创建签名器
- ✅ **NewSignerFromPath(prvPath string)**: 从文件路径加载私钥创建签名器
- ✅ **NewSignerFromMnemonic(mnemonic string)**: 从助记词创建签名器
- ✅ **SignTx(...)**: 签名交易
- ✅ **SignMsg(message []byte)**: 签名消息
- ✅ **SignTypedData(typedData)**: 签名 EIP-712 类型化数据
- ✅ **GetPublicKey()**: 获取公钥字节数组
- ✅ **GetPublicKeyHex()**: 获取公钥十六进制字符串
- ✅ **GetPrivateKey()**: 获取私钥对象
- ✅ **Decrypt(data []byte)**: 解密数据

### Wallet 模块

连接以太坊网络，执行状态改变操作。

#### 创建钱包

```golang
// 基本创建
wallet, err := goether.NewWallet(privateKey, rpcURL)

// 指定链 ID
chainID := big.NewInt(1) // 主网
wallet, err := goether.NewWallet(privateKey, rpcURL, chainID)

// 使用自定义 RPC 客户端
client := ethrpc.New(rpcURL)
wallet, err := goether.NewWallet(privateKey, "", client)

// 从现有钱包复制配置
newWallet, err := goether.NewWallet(newPrivateKey, "", existingWallet)
```

#### 主要方法

- ✅ **SendTx(to, amount, data, opts)**: 发送交易（默认 EIP-1559）
- ✅ **SendLegacyTx(to, amount, data, opts)**: 发送 Legacy 交易
- ✅ **GetAddress()**: 获取钱包地址
- ✅ **GetBalance()**: 获取 ETH 余额
- ✅ **GetNonce()**: 获取当前 nonce
- ✅ **GetPendingNonce()**: 获取待处理 nonce
- ✅ **InitTxOpts(...)**: 初始化交易选项

#### TxOpts 交易选项

```golang
type TxOpts struct {
    Nonce     *int      // 交易序号
    GasLimit  *int      // Gas 限制
    GasPrice  *big.Int  // Gas 价格（Legacy 交易）
    GasTipCap *big.Int  // 矿工小费（EIP-1559）
    GasFeeCap *big.Int  // 最大费用（EIP-1559）
}

// 计算 Legacy 交易费用
fee, err := opts.GetOldFee()

// 计算 EIP-1559 交易费用
fee, err := opts.GetNewFee()
```

### Contract 模块

创建合约实例，用于调用和执行合约方法。

#### 主要方法

- ✅ **NewContract(address, abi, rpc, wallet)**: 创建合约实例
- ✅ **CallMethod(method, tag, args...)**: 调用只读方法
- ✅ **ExecMethod(method, opts, args...)**: 执行状态改变方法
- ✅ **EncodeData(method, args...)**: 编码方法调用数据
- ✅ **EncodeDataHex(method, args...)**: 编码为十六进制字符串
- ✅ **DecodeData(method, data)**: 解码返回数据
- ✅ **DecodeDataHex(method, dataHex)**: 解码十六进制数据
- ✅ **DecodeEvent(event, data)**: 解码事件数据
- ✅ **DecodeEventHex(event, dataHex)**: 解码十六进制事件数据

### Utils 工具函数

常用的以太坊工具函数。

- ✅ **EthToBN(amount float64)**: 将 ETH 数量转换为 big.Int（wei）
- ✅ **GweiToBN(amount float64)**: 将 Gwei 数量转换为 big.Int（wei）
- ✅ **EIP712Hash(typedData)**: 计算 EIP-712 类型化数据哈希
- ✅ **Ecrecover(hash, signature)**: 从签名恢复公钥和地址
- ✅ **Encrypt(data, publicKey)**: 使用公钥加密数据

```golang
// 单位转换示例
oneEth := goether.EthToBN(1.0)        // 1 ETH = 1e18 wei
tenGwei := goether.GweiToBN(10.0)     // 10 Gwei = 1e10 wei

// EIP-712 签名
hash, err := goether.EIP712Hash(typedData)

// 签名恢复
pubKey, address, err := goether.Ecrecover(messageHash, signature)
```

## 配置选项

### RPC 客户端配置

```golang
// 自定义 RPC 客户端选项
clientOptions := []func(rpc *ethrpc.EthRPC){
    func(rpc *ethrpc.EthRPC) {
        // 设置超时时间
        rpc.SetTimeout(30 * time.Second)
    },
    func(rpc *ethrpc.EthRPC) {
        // 设置重试次数
        rpc.SetRetryCount(3)
    },
}

wallet, err := goether.NewWallet(privateKey, rpcURL, clientOptions...)
```

### 网络配置

```golang
// 主要以太坊网络的链 ID
var (
    MainnetChainID = big.NewInt(1)     // 以太坊主网
    GoerliChainID  = big.NewInt(5)     // Goerli 测试网
    SepoliaChainID = big.NewInt(11155111) // Sepolia 测试网
    PolygonChainID = big.NewInt(137)   // Polygon 主网
    BSCChainID     = big.NewInt(56)    // BSC 主网
)

// 使用特定网络
wallet, err := goether.NewWallet(privateKey, rpcURL, MainnetChainID)
```

## 错误处理

### 常见错误类型

```golang
// 私钥格式错误
if err != nil {
    if strings.Contains(err.Error(), "invalid hex character") {
        log.Fatal("私钥格式错误，请检查十六进制格式")
    }
}

// RPC 连接错误
if err != nil {
    if strings.Contains(err.Error(), "connection refused") {
        log.Fatal("无法连接到以太坊节点，请检查 RPC URL")
    }
}

// Gas 不足错误
if err != nil {
    if strings.Contains(err.Error(), "insufficient funds") {
        log.Fatal("账户余额不足，无法支付交易费用")
    }
}

// Nonce 错误
if err != nil {
    if strings.Contains(err.Error(), "nonce too low") {
        log.Fatal("Nonce 值过低，请使用更高的 nonce")
    }
}
```

### 最佳实践

1. **私钥安全**
   ```golang
   // ❌ 不要在代码中硬编码私钥
   privateKey := "0x1234567890abcdef..."
   
   // ✅ 从环境变量或配置文件读取
   privateKey := os.Getenv("PRIVATE_KEY")
   if privateKey == "" {
       log.Fatal("请设置 PRIVATE_KEY 环境变量")
   }
   ```

2. **Gas 费用管理**
   ```golang
   // ✅ 在发送交易前估算 Gas
   gasLimit, err := wallet.EstimateGas(to, amount, data)
   if err != nil {
       return err
   }
   
   // ✅ 设置合理的 Gas 价格
   gasPrice, err := wallet.SuggestGasPrice()
   if err != nil {
       return err
   }
   
   opts := &goether.TxOpts{
       GasLimit: &gasLimit,
       GasPrice: gasPrice,
   }
   ```

3. **交易确认**
   ```golang
   // ✅ 等待交易确认
   txHash, err := wallet.SendTx(to, amount, data, opts)
   if err != nil {
       return err
   }
   
   // 等待交易被挖掘
   receipt, err := wallet.WaitForReceipt(txHash, 5*time.Minute)
   if err != nil {
       return err
   }
   
   if receipt.Status == 1 {
       fmt.Println("交易成功")
   } else {
       fmt.Println("交易失败")
   }
   ```

4. **错误重试机制**
   ```golang
   // ✅ 实现重试机制
   func sendTxWithRetry(wallet *goether.Wallet, to common.Address, amount *big.Int, data []byte, opts *goether.TxOpts, maxRetries int) (string, error) {
       for i := 0; i < maxRetries; i++ {
           txHash, err := wallet.SendTx(to, amount, data, opts)
           if err == nil {
               return txHash, nil
           }
           
           // 如果是 nonce 错误，更新 nonce 后重试
           if strings.Contains(err.Error(), "nonce") {
               nonce, err := wallet.GetPendingNonce()
               if err != nil {
                   continue
               }
               opts.Nonce = &nonce
           }
           
           time.Sleep(time.Second * time.Duration(i+1))
       }
       return "", fmt.Errorf("重试 %d 次后仍然失败", maxRetries)
   }
   ```

## 常见问题 (FAQ)

### Q: 如何选择合适的 Gas 价格？

A: 可以通过以下方式获取合适的 Gas 价格：

```golang
// 获取网络建议的 Gas 价格
gasPrice, err := wallet.SuggestGasPrice()

// 或者查看当前网络状况，手动设置
// 主网通常 10-50 Gwei，拥堵时可能更高
gasPrice := goether.GweiToBN(20) // 20 Gwei
```

### Q: EIP-1559 和 Legacy 交易有什么区别？

A: 
- **Legacy 交易**: 使用固定的 `gasPrice`，费用 = gasPrice × gasUsed
- **EIP-1559 交易**: 使用 `baseFee + tip`，更灵活的费用机制

```golang
// Legacy 交易
opts := &goether.TxOpts{
    GasPrice: goether.GweiToBN(20),
}

// EIP-1559 交易
opts := &goether.TxOpts{
    GasTipCap: goether.GweiToBN(2),  // 矿工小费
    GasFeeCap: goether.GweiToBN(30), // 最大费用
}
```

### Q: 如何处理交易卡住的情况？

A: 可以通过提高 Gas 价格来加速交易：

```golang
// 使用相同的 nonce 但更高的 Gas 价格发送新交易
higherGasPrice := goether.GweiToBN(50) // 提高 Gas 价格
newOpts := &goether.TxOpts{
    Nonce:    opts.Nonce,    // 使用相同的 nonce
    GasPrice: higherGasPrice, // 更高的 Gas 价格
}

// 发送加速交易
txHash, err := wallet.SendTx(to, amount, data, newOpts)
```

### Q: 如何安全地存储私钥？

A: 推荐的私钥存储方式：

1. **环境变量**（开发环境）
2. **加密的配置文件**（生产环境）
3. **硬件钱包**（高安全要求）
4. **密钥管理服务**（企业级应用）

```golang
// 从环境变量读取
privateKey := os.Getenv("PRIVATE_KEY")

// 从加密文件读取
encryptedKey, err := ioutil.ReadFile("encrypted_key.json")
if err != nil {
    return err
}
privateKey, err := decrypt(encryptedKey, password)
```

## 贡献指南

我们欢迎社区贡献！请遵循以下步骤：

1. Fork 本仓库
2. 创建功能分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

### 开发环境设置

```bash
# 克隆仓库
git clone https://github.com/go-enols/goether.git
cd goether

# 安装依赖
go mod download

# 运行测试
go test ./...

# 运行示例
go test ./example -v
```

## 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 支持

如果您在使用过程中遇到问题，可以通过以下方式获取帮助：

- 📖 查看 [文档](./README.md)
- 🐛 提交 [Issue](https://github.com/go-enols/goether/issues)
- 💬 参与 [Discussions](https://github.com/go-enols/goether/discussions)
- 📧 发送邮件到维护者

## 相关项目

- [go-ethereum](https://github.com/ethereum/go-ethereum) - 官方以太坊 Go 客户端
- [ethrpc](https://github.com/go-enols/ethrpc) - 以太坊 RPC 客户端库

---

**免责声明**: 本库仅供学习和开发使用。在生产环境中使用前，请确保充分测试并遵循安全最佳实践。处理加密货币时请格外小心，确保私钥安全。
