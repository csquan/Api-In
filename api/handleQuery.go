package api

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/coin-manage/types"
	"github.com/ethereum/coin-manage/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const ADDRLEN = 42

const Ok = 0

func checkAddr(addr string) error {
	if addr[:2] != "0x" {
		return errors.New("addr must start with 0x")
	}
	if len(addr) != ADDRLEN {
		return errors.New("addr len wrong ,must 40")
	}
	return nil
}

// 首先查询balance_erc20表，得到地址持有的代币合约地址，然后根据代币合约地址查erc20_info表
func (a *ApiService) getCoinHistory(c *gin.Context) {
	addr := c.Param("contractAddr")
	res := types.HttpRes{}

	data := types.CoinData{}
	err := checkAddr(addr)
	if err != nil {
		logrus.Error(err)
		res.Code = http.StatusBadRequest
		res.Message = err.Error()
		c.SecureJSON(http.StatusBadRequest, res)
		return
	}

	coinInfos, err := a.db.GetCoinInfo(strings.ToLower(addr))
	if err != nil {
		logrus.Error(err)
		res.Code = http.StatusInternalServerError
		res.Message = err.Error()
		c.SecureJSON(http.StatusInternalServerError, res)
		return
	}

	if len(coinInfos) > 0 {
		data.InitCoinSupply = coinInfos[0].Tokens_Cnt
		coinInfos = coinInfos[1:]
	}

	//coinInfos是按照blockNum排序的，所以开始第一个元素一定是初始供应量
	for _, info := range coinInfos {
		if data.AddCoinHistory != "" {
			data.AddCoinHistory = data.AddCoinHistory + "," + info.Tokens_Cnt
		} else {
			data.AddCoinHistory = info.Tokens_Cnt
		}
	}

	b, err := json.Marshal(data)
	if err != nil {
		res.Code = http.StatusInternalServerError
		res.Message = err.Error()
		c.SecureJSON(http.StatusInternalServerError, res)
		return
	}

	res.Code = Ok
	res.Message = "success"
	res.Data = string(b)
	c.SecureJSON(http.StatusOK, res)
}

// 首先查询balance_erc20表，得到地址持有的代币合约地址，然后根据代币合约地址查erc20_info表
func (a *ApiService) getCoinBalance(c *gin.Context) {
	accountAddr := c.Param("accountAddr")
	contractAddr := c.Param("contractAddr")

	res := types.HttpRes{}

	err := checkAddr(accountAddr)
	if err != nil {
		logrus.Error(err)
		res.Code = http.StatusBadRequest
		res.Message = err.Error()
		c.SecureJSON(http.StatusBadRequest, res)
		return
	}

	err = checkAddr(contractAddr)
	if err != nil {
		logrus.Error(err)
		res.Code = http.StatusBadRequest
		res.Message = err.Error()
		c.SecureJSON(http.StatusBadRequest, res)
		return
	}
	balance, err := a.db.GetCoinBalance(strings.ToLower(accountAddr), strings.ToLower(contractAddr))
	if err != nil {
		logrus.Error(err)
		res.Code = http.StatusInternalServerError
		res.Message = err.Error()
		c.SecureJSON(http.StatusInternalServerError, res)
		return
	}

	coinInfo, err := a.db.QuerySpecifyCoinInfo(strings.ToLower(contractAddr))
	if err != nil {
		logrus.Error(err)
	}

	if balance != "" {
		balance = HandleAmountDecimals(balance, coinInfo.Decimals)
	} else {
		balance = "0"
	}

	res.Code = Ok
	res.Message = "success"
	res.Data = balance
	c.SecureJSON(http.StatusOK, res)
}

// 首先查询balance_erc20表，得到地址持有的代币合约地址，然后根据代币合约地址查erc20_info表
func (a *ApiService) getAllCoinAllCount(c *gin.Context) {
	addr := c.Param("accountAddr")
	res := types.HttpRes{}

	err := checkAddr(addr)
	if err != nil {
		logrus.Error(err)
		res.Code = http.StatusBadRequest
		res.Message = err.Error()
		c.SecureJSON(http.StatusBadRequest, res)
		return
	}

	count, err := a.db.QueryAllCoinAllHolders(strings.ToLower(addr))
	if err != nil {
		logrus.Error(err)
		res.Code = http.StatusInternalServerError
		res.Message = err.Error()
		c.SecureJSON(http.StatusInternalServerError, res)
		return
	}

	res.Code = Ok
	res.Message = "success"
	res.Data = fmt.Sprintf("%d", count)
	c.SecureJSON(http.StatusOK, res)
}

// 首先查询balance_erc20表，得到地址持有的代币合约地址，然后根据代币合约地址查erc20_info表
func (a *ApiService) getSpecifyCoinInfo(c *gin.Context) {
	addr := c.Param("contractAddr")
	res := types.HttpRes{}

	err := checkAddr(addr)
	if err != nil {
		logrus.Error(err)
		res.Code = http.StatusBadRequest
		res.Message = err.Error()
		c.SecureJSON(http.StatusBadRequest, res)
		return
	}

	info, err := a.db.QuerySpecifyCoinInfo(strings.ToLower(addr))
	if err != nil {
		logrus.Error(err)
		res.Code = http.StatusInternalServerError
		res.Message = err.Error()
		c.SecureJSON(http.StatusInternalServerError, res)
		return
	}
	//处理下 info的精度
	info.Totoal_Supply = HandleAmountDecimals(info.Totoal_Supply, info.Decimals)
	b, err := json.Marshal(info)
	if err != nil {
		logrus.Error(err)
		res.Code = http.StatusInternalServerError
		res.Message = err.Error()
		c.SecureJSON(http.StatusInternalServerError, res)
		return
	}

	res.Code = Ok
	res.Message = "success"
	res.Data = string(b)
	c.SecureJSON(http.StatusOK, res)
}

func HandleAmountDecimals(amount string, decimal string) string {
	decimalInt, err := strconv.ParseInt(decimal, 10, 64)
	if err != nil {
		logrus.Error(err)
	}
	str := ""
	if decimalInt >= 8 {
		pos := decimalInt - 8
		endpos := len(amount) - int(pos)

		str = amount[:endpos-8] + "." + amount[endpos-8:endpos]
	} else {
		splitpos := len(amount) - int(decimalInt)
		str = amount[:splitpos] + "." + amount[splitpos:len(amount)]
	}
	return str
}

// 首先查询balance_erc20表，得到地址持有的代币合约地址，然后根据代币合约地址查erc20_info表
func (a *ApiService) getCoinInfos(c *gin.Context) {
	addr := c.Param("accountAddr")
	res := types.HttpRes{}

	err := checkAddr(addr)
	if err != nil {
		logrus.Error(err)
		res.Code = http.StatusBadRequest
		res.Message = err.Error()
		c.SecureJSON(http.StatusBadRequest, res)
		return
	}

	baseInfos, err := a.db.QueryCoinInfos(strings.ToLower(addr))
	if err != nil {
		logrus.Error(err)
		res.Code = http.StatusInternalServerError
		res.Message = err.Error()
		c.SecureJSON(http.StatusInternalServerError, res)
		return
	}

	coinInfos := make([]*types.CoinInfo, 0)

	height, err := a.db.GetBlockHeight()
	if err != nil {
		logrus.Error(err)
		res.Code = http.StatusInternalServerError
		res.Message = err.Error()
		c.SecureJSON(http.StatusInternalServerError, res)
		return
	}

	//这里拼装每个代币的持币地址数 状态
	for _, info := range baseInfos {
		coinInfo := types.CoinInfo{
			BaseInfo: *info,
			Status:   1, //正常交易
		}

		instance, _ := util.PrepareTx(a.config, info.Addr)

		blackRange, err := instance.BlackBlocks(nil)
		if err != nil && err.Error() != "abi: attempting to unmarshall an empty string while arguments are expected" {
			logrus.Error(err)
			continue
		}
		for _, rangeValue := range blackRange {
			if height >= int(rangeValue.BeginBlock.Int64()) || height <= int(rangeValue.EndBlock.Int64()) {
				coinInfo.Status = 0 //暂停交易
			}
		}

		count, err := a.db.QueryCoinHolderCount(strings.ToLower(info.Addr))
		if err != nil {
			logrus.Error(err)
		}
		coinInfo.HolderCount = count

		coinInfo.BaseInfo.Totoal_Supply = HandleAmountDecimals(coinInfo.BaseInfo.Totoal_Supply, coinInfo.BaseInfo.Decimals)
		coinInfos = append(coinInfos, &coinInfo)
	}

	b, err := json.Marshal(coinInfos)
	if err != nil {
		logrus.Error(err)
		res.Code = http.StatusInternalServerError
		res.Message = err.Error()
		c.SecureJSON(http.StatusInternalServerError, res)
		return
	}

	res.Code = Ok
	res.Message = "success"
	res.Data = string(b)
	c.SecureJSON(http.StatusOK, res)
}

func (a *ApiService) getCoinHoldersCount(c *gin.Context) {
	addr := c.Param("contractAddr")
	res := types.HttpRes{}

	err := checkAddr(addr)

	if err != nil {
		logrus.Error(err)
		res.Code = http.StatusBadRequest
		res.Message = err.Error()
		c.SecureJSON(http.StatusBadRequest, res)
		return
	}

	holderInfos, err := a.db.QueryCoinHolderCount(strings.ToLower(addr))
	if err != nil {
		logrus.Error(err)
		res.Code = http.StatusInternalServerError
		res.Message = err.Error()
		c.SecureJSON(http.StatusInternalServerError, res)
		return
	}

	b, err := json.Marshal(holderInfos)
	if err != nil {
		logrus.Error(err)
		res.Code = http.StatusInternalServerError
		res.Message = err.Error()
		c.SecureJSON(http.StatusInternalServerError, res)
		return
	}
	res.Code = Ok
	res.Message = "success"
	res.Data = string(b)
	c.SecureJSON(http.StatusOK, res)
}

func (a *ApiService) getCoinHolders(c *gin.Context) {
	contractAddr := c.Param("contractAddr")
	res := types.HttpRes{}

	err := checkAddr(contractAddr)

	if err != nil {
		logrus.Error(err)
		res.Code = http.StatusBadRequest
		res.Message = err.Error()
		c.SecureJSON(http.StatusBadRequest, res)
		return
	}

	infos := make([]*types.Balance_Erc20, 0)

	holderInfos, err := a.db.QueryCoinHolders(strings.ToLower(contractAddr))
	if err != nil {
		logrus.Error(err)
		res.Code = http.StatusInternalServerError
		res.Message = err.Error()
		c.SecureJSON(http.StatusInternalServerError, res)
		return
	}

	info, err := a.db.QuerySpecifyCoinInfo(strings.ToLower(contractAddr))
	if err != nil {
		logrus.Error(err)
		return
	}

	//过滤Addr空地址
	for _, holderInfo := range holderInfos {
		if holderInfo.Balance != "0" {
			//处理下 info的精度
			holderInfo.Balance = HandleAmountDecimals(holderInfo.Balance, info.Decimals)
			infos = append(infos, holderInfo)
		}
	}

	b, err := json.Marshal(infos)
	if err != nil {
		logrus.Error(err)
		res.Code = http.StatusInternalServerError
		res.Message = err.Error()
		c.SecureJSON(http.StatusInternalServerError, res)
		return
	}
	res.Code = Ok
	res.Message = "success"
	res.Data = string(b)
	c.SecureJSON(http.StatusOK, res)
}

func parse(db types.IDB, txhash string) (*types.OpParam, error) {
	op := ""
	//首先在tx_log中找到这笔hash对应的交易，比对op表中的hash，看是哪个动作，取出对应的参数个数和参数格式
	tx_log, err := db.QueryTxlogByHash(txhash)
	if err != nil {
		logrus.Error(err)
	}
	eventHashs, err := db.GetEventHash()
	if err != nil {
		logrus.Error(err)
	}
	if tx_log == nil {
		logrus.Info("tx_log is null and txhash is " + txhash)
		return nil, nil
	}
	opparam := types.OpParam{}

	for _, value := range eventHashs {
		if tx_log.Topic0 == "0x"+value.EventHash { //找到动作,然后依据格式解析参数
			op = value.Op
			opparam.Op = op
			switch op {
			case "AddBlack": //event AddBlack(address account);
				opparam.Addr1 = formatHex(tx_log.Data)
				break
			case "RemoveBlack": // event RemoveBlack(address account);
				opparam.Addr1 = formatHex(tx_log.Data)
				break
			case "AddBlackIn": // event AddBlackIn(address account);
				opparam.Addr1 = formatHex(tx_log.Data)
				break
			case "RemoveBlackIn": // event RemoveBlackIn(address account);
				opparam.Addr1 = formatHex(tx_log.Data)
				break
			case "AddBlackOut": //event AddBlackOut(address account);
				opparam.Addr1 = formatHex(tx_log.Data)
				break
			case "RemoveBlackOut": //event RemoveBlackOut(address account);
				opparam.Addr1 = formatHex(tx_log.Data)
				break
			case "AddBlackBlock": //这里tx_log.Data 含有2个uint128参数- event AddBlackBlock(uint128 _beginBlock, uint128 _endBlock);
				valueStr1 := formatHex(tx_log.Data[:66])
				if valueStr1 == "0x" {
					opparam.Value1 = "0"
				} else {
					value1, err := hexutil.DecodeBig(valueStr1)
					if err != nil {
						logrus.Error(err)
					}
					opparam.Value1 = value1.String()
				}

				valueStr2 := formatHex(tx_log.Data[66:])
				if valueStr2 == "0x" {
					opparam.Value2 = "0"
				} else {
					value2, err := hexutil.DecodeBig(valueStr2)
					if err != nil {
						logrus.Error(err)
					}
					opparam.Value2 = value2.String()
				}
				break
			case "RemoveBlackBlock": //这里tx_log.Data 含有3个uint参数- event RemoveBlackBlock(uint256 i, uint128 _beginBlock, uint128 _endBlock);
				valueStr1 := formatHex(tx_log.Data[:66])
				if valueStr1 == "0x" {
					opparam.Value1 = "0"
				} else {
					value1, err := hexutil.DecodeBig(valueStr1)
					if err != nil {
						logrus.Error(err)
					}
					opparam.Value1 = value1.String()
				}

				valueStr2 := formatHex(tx_log.Data[66:130])
				if valueStr2 == "0x" {
					opparam.Value2 = "0"
				} else {
					value2, err := hexutil.DecodeBig(valueStr2)
					if err != nil {
						logrus.Error(err)
					}
					opparam.Value2 = value2.String()
				}

				valueStr3 := formatHex(tx_log.Data[130:])
				if valueStr3 == "0x" {
					opparam.Value3 = "0"
				} else {
					value3, err := hexutil.DecodeBig(valueStr3)
					if err != nil {
						logrus.Error(err)
					}
					opparam.Value3 = value3.String()
				}
				break
			case "Frozen": //这里tx_log.Data 含有后2个uint128参数- event Frozen(address indexed account, uint256 frozen, uint256 waitFrozen);
				param1 := common.HexToAddress(tx_log.Topic1)
				opparam.Addr1 = param1.Hex()

				valueStr1 := formatHex(tx_log.Data[:66])
				if valueStr1 == "0x" {
					opparam.Value1 = "0"
				} else {
					value1, err := hexutil.DecodeBig(valueStr1)
					if err != nil {
						logrus.Error(err)
					}
					opparam.Value1 = value1.String()
				}

				valueStr2 := formatHex(tx_log.Data[66:])
				if valueStr2 == "0x" {
					opparam.Value2 = "0"
				} else {
					value2, err := hexutil.DecodeBig(valueStr2)
					if err != nil {
						logrus.Error(err)
					}
					opparam.Value2 = value2.String()
				}
				break
			case "Transfer": // event Transfer(address indexed from, address indexed to, uint256 value);
				param1 := common.HexToAddress(tx_log.Topic1)
				opparam.Addr1 = param1.String()

				param2 := common.HexToAddress(tx_log.Topic2)
				opparam.Addr2 = param2.String()

				valueStr := formatHex(tx_log.Data)
				value, err := hexutil.DecodeBig(valueStr)
				if err != nil {
					logrus.Error(err)
				}
				opparam.Value1 = value.String()
				break
			case "UnFrozen": //这里tx_log.Data 含有后2个uint128参数- event Frozen(address indexed account, uint256 frozen, uint256 waitFrozen);
				param1 := common.HexToAddress(tx_log.Topic1)
				opparam.Addr1 = param1.Hex()

				valueStr1 := formatHex(tx_log.Data[:66])
				if valueStr1 == "0x" {
					opparam.Value1 = "0"
				} else {
					value1, err := hexutil.DecodeBig(valueStr1)
					if err != nil {
						logrus.Error(err)
					}
					opparam.Value1 = value1.String()
				}

				valueStr2 := formatHex(tx_log.Data[66:])
				if valueStr2 == "0x" {
					opparam.Value2 = "0"
				} else {
					value2, err := hexutil.DecodeBig(valueStr2)
					if err != nil {
						logrus.Error(err)
					}
					opparam.Value2 = value2.String()
				}

				break
			case "Paused": //event Paused(address account);
				param1 := common.HexToAddress(tx_log.Data)
				opparam.Addr1 = param1.String()
				break
			case "Unpaused": //event Unpaused(address account);
				param1 := common.HexToAddress(tx_log.Data)
				opparam.Addr1 = param1.String()
				break
			}
		}
	}

	return &opparam, nil
}

func formatHex(hexstr string) string {
	res := strings.TrimLeft(hexstr[2:], "0")
	return "0x" + res
}

func (a *ApiService) hasBurnAmount(c *gin.Context) {
	accountAddr := c.Param("accountAddr")
	contractAddr := c.Param("contractAddr")
	res := types.HttpRes{}

	err := checkAddr(accountAddr)
	if err != nil {
		logrus.Error(err)
		res.Code = http.StatusBadRequest
		res.Message = err.Error()
		c.SecureJSON(http.StatusBadRequest, res)
		return
	}
	err = checkAddr(contractAddr)
	if err != nil {
		logrus.Error(err)
		res.Code = http.StatusBadRequest
		res.Message = err.Error()
		c.SecureJSON(http.StatusBadRequest, res)
		return
	}

	Txs, err := a.db.QueryBurnTxs(strings.ToLower(accountAddr), strings.ToLower(contractAddr))
	if err != nil {
		logrus.Error(err)
		res.Code = http.StatusInternalServerError
		res.Message = err.Error()
		c.SecureJSON(http.StatusInternalServerError, res)
		return
	}
	var sum int64
	for _, tx := range Txs {
		parseInt, err := strconv.ParseInt(tx.Value, 10, 64)
		if err != nil {
			logrus.Error(err)
			res.Code = http.StatusBadRequest
			res.Message = err.Error()
			c.SecureJSON(http.StatusBadRequest, res)
			return
		}
		sum += parseInt
	}

	res.Code = Ok
	res.Message = "success"
	res.Data = fmt.Sprintf("%d", sum)
	c.SecureJSON(http.StatusOK, res)
}

func copyStruct(paramDest *types.OpParam, paramSrc *types.OpParam) {
	paramDest.Op = paramSrc.Op
	paramDest.Value1 = paramSrc.Value1
	paramDest.Value2 = paramSrc.Value2
	paramDest.Value3 = paramSrc.Value3
	paramDest.Addr1 = paramSrc.Addr1
	paramDest.Addr2 = paramSrc.Addr2
}

func (a *ApiService) getTxHistory(c *gin.Context) {
	accountAddr := c.Param("accountAddr")
	contractAddr := c.Param("contractAddr")
	res := types.HttpRes{}

	err := checkAddr(accountAddr)
	if err != nil {
		logrus.Error(err)
		res.Code = http.StatusBadRequest
		res.Message = err.Error()
		c.SecureJSON(http.StatusBadRequest, res)
		return
	}
	err = checkAddr(contractAddr)
	if err != nil {
		logrus.Error(err)
		res.Code = http.StatusBadRequest
		res.Message = err.Error()
		c.SecureJSON(http.StatusBadRequest, res)
		return
	}

	TxInfos, err := a.db.QueryTxHistory(strings.ToLower(accountAddr), strings.ToLower(contractAddr))
	if err != nil {
		logrus.Error(err)
		res.Code = http.StatusInternalServerError
		res.Message = err.Error()
		c.SecureJSON(http.StatusInternalServerError, res)
		return
	}

	//Erc20TxInfos, err := a.db.QueryTxErc20History(strings.ToLower(addr))
	//if err != nil {
	//	res.Code = http.StatusInternalServerError
	//	res.Message = err.Error()
	//	c.SecureJSON(http.StatusInternalServerError, res)
	//	return
	//}

	//动作-TxInfos从input中解析，Erc20TxInfos是属于内部交易，动作为转账
	txArray := make([]types.TxRes, 0)

	for _, tx := range TxInfos {
		txRes := types.TxRes{}

		parseUInt, err := strconv.ParseUint(tx.Value, 10, 64)
		if err != nil {
			logrus.Error(err)
			continue
		}
		txRes.Amount = parseUInt

		txRes.Hash = tx.Hash
		txRes.TxGeneral = tx
		opparam := types.OpParam{}

		if tx.IsContractCreate == "1" {
			opparam.Op = "ContractCreate"
		} else {
			if tx.IsContract == "1" { //需要解析input
				param, err := parse(a.db, tx.Hash)
				if err != nil {
					logrus.Error(err)
					res.Code = http.StatusInternalServerError
					res.Message = err.Error()
					c.SecureJSON(http.StatusInternalServerError, res)
					return
				}
				if param != nil {
					copyStruct(&opparam, param)
				}

				if opparam.Op == "Transfer" {

					coinInfo, err := a.db.QuerySpecifyCoinInfo(strings.ToLower(tx.To))
					if err != nil {
						logrus.Error(err)
					}

					decimalInt, err := strconv.ParseInt(coinInfo.Decimals, 10, 64)
					if err != nil {
						logrus.Error(err)
						res.Code = http.StatusBadRequest
						res.Message = err.Error()
						c.SecureJSON(http.StatusBadRequest, res)
						return
					}

					if tx.From == accountAddr {
						opparam.Op = "TransferOut"

						if len(opparam.Value1) > int(decimalInt) {
							opparam.Value1 = HandleAmountDecimals(opparam.Value1, coinInfo.Decimals)
						}

						if tx.To == "" {
							opparam.Op = "Destroy"
						}
					} else {
						opparam.Op = "TransferIn"

						if len(opparam.Value1) > int(decimalInt) {
							opparam.Value1 = HandleAmountDecimals(opparam.Value1, coinInfo.Decimals)
						}
						if tx.From == "" {
							opparam.Op = "Increase"
						}
					}
				}
			} else {
				if tx.From == accountAddr {
					opparam.Op = "TransferOut"

					if tx.To == "" {
						opparam.Op = "Destroy"
					}
				} else {
					opparam.Op = "TransferIn"

					if tx.From == "" {
						opparam.Op = "Increase"
					}
				}
			}
		}
		txRes.OpParams = &opparam
		txArray = append(txArray, txRes)
	}

	b, err := json.Marshal(txArray)

	res.Code = Ok
	res.Message = "success"
	res.Data = json.RawMessage(b)
	c.JSON(http.StatusOK, res)
}

func (a *ApiService) getBlockHeight(c *gin.Context) {

	res := types.HttpRes{}

	count, err := a.db.GetBlockHeight()
	if err != nil {
		logrus.Error(err)
		res.Code = http.StatusInternalServerError
		res.Message = err.Error()
		c.SecureJSON(http.StatusInternalServerError, res)
		return
	}

	res.Code = Ok
	res.Message = "success"
	res.Data = fmt.Sprintf("%d", count)

	c.SecureJSON(http.StatusOK, res)
}

func (a *ApiService) GetTask(c *gin.Context) {
	buf := make([]byte, 1024)
	n, _ := c.Request.Body.Read(buf)
	data1 := string(buf[0:n])
	res := types.HttpRes{}

	isValid := gjson.Valid(data1)
	if isValid == false {
		logrus.Error("Not valid json")
		res.Code = http.StatusBadRequest
		res.Message = "Not valid json"
		c.SecureJSON(http.StatusBadRequest, res)
		return
	}

	contractAddr := gjson.Get(data1, "contractAddr")
	accountAddr := gjson.Get(data1, "accountAddr")
	uuid := gjson.Get(data1, "uuid")

	err := checkAddr(contractAddr.String())
	if err != nil {
		logrus.Error(err)
		res.Code = http.StatusBadRequest
		res.Message = err.Error()
		c.SecureJSON(http.StatusBadRequest, res)
		return
	}

	err = checkAddr(accountAddr.String())
	if err != nil {
		logrus.Error(err)
		res.Code = http.StatusBadRequest
		res.Message = err.Error()
		c.SecureJSON(http.StatusBadRequest, res)
		return
	}

	cli := resty.New()

	data := types.TxData{
		RequestID: strconv.Itoa(int(time.Now().Unix())),
		UUID:      uuid.String(),
		From:      accountAddr.String(),
	}

	var result types.HttpRes
	resp, err := cli.R().SetBody(data).SetResult(&result).Post(a.config.TxState.EndPoint + "/tx/get")
	if err != nil {
		logrus.Error(err)
	}
	if resp.StatusCode() != http.StatusOK {
		logrus.Error(resp)
	}
	if result.Code != 0 {
		logrus.Error(result)
	}

	if err != nil {
		logrus.Error(err)
	}

	res.Code = Ok
	res.Message = "success"
	res.Data = result.Message

	c.SecureJSON(http.StatusOK, res)
}

func (a *ApiService) getStatus(contractAddr string, accountAddr string) (*types.StatusInfo, error) {
	instance, _ := util.PrepareTx(a.config, contractAddr)

	isBlack, err := instance.BlackOf(nil, common.HexToAddress(accountAddr))
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	isBlackIn, err := instance.BlackInOf(nil, common.HexToAddress(accountAddr))
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	isBlackOut, err := instance.BlackOutOf(nil, common.HexToAddress(accountAddr))
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	nowFrozenAmount, err := instance.FrozenOf(nil, common.HexToAddress(accountAddr))
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	waitFrozenAmount, err := instance.WaitFrozenOf(nil, common.HexToAddress(accountAddr))
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	status := types.StatusInfo{
		IsBlack:          isBlack,
		IsBlackIn:        isBlackIn,
		IsBlackOut:       isBlackOut,
		NowFrozenAmount:  nowFrozenAmount,
		WaitFrozenAmount: waitFrozenAmount,
	}
	return &status, nil
}

func (a *ApiService) cap(c *gin.Context) {
	buf := make([]byte, 1024)
	n, _ := c.Request.Body.Read(buf)
	data1 := string(buf[0:n])
	res := types.HttpRes{}

	isValid := gjson.Valid(data1)
	if isValid == false {
		logrus.Error("Not valid json")
		res.Code = http.StatusBadRequest
		res.Message = "Not valid json"
		c.SecureJSON(http.StatusBadRequest, res)
		return
	}
	contractAddr := gjson.Get(data1, "contractAddr")

	err := checkAddr(contractAddr.String())
	if err != nil {
		logrus.Error(err)
		res.Code = http.StatusBadRequest
		res.Message = err.Error()
		c.SecureJSON(http.StatusBadRequest, res)
		return
	}

	instance, _ := util.PrepareTx(a.config, contractAddr.String())

	capValue, err := instance.Cap(nil)
	if err != nil && err.Error() != "abi: attempting to unmarshall an empty string while arguments are expected" && err.Error() != "execution reverted" {
		logrus.Error(err)
		res.Code = http.StatusInternalServerError
		res.Message = err.Error()
		c.SecureJSON(http.StatusInternalServerError, res)
		return
	}

	coinInfo, err := a.db.QuerySpecifyCoinInfo(strings.ToLower(contractAddr.String()))
	if err != nil {
		logrus.Error(err)
	}

	res.Code = Ok
	res.Message = "success"

	if capValue == nil || capValue.Uint64() == 0 {
		res.Data = "0"
	} else {
		res.Data = HandleAmountDecimals(capValue.String(), coinInfo.Decimals)
	}

	c.SecureJSON(http.StatusOK, res)
}

func (a *ApiService) hasForzenAmount(c *gin.Context) {
	buf := make([]byte, 1024)
	n, _ := c.Request.Body.Read(buf)
	data1 := string(buf[0:n])
	res := types.HttpRes{}

	isValid := gjson.Valid(data1)
	if isValid == false {
		logrus.Error("Not valid json")
		res.Code = http.StatusBadRequest
		res.Message = "Not valid json"
		c.SecureJSON(http.StatusBadRequest, res)
		return
	}
	contractAddr := gjson.Get(data1, "contractAddr")
	accountAddr := gjson.Get(data1, "accountAddr")

	err := checkAddr(contractAddr.String())
	if err != nil {
		logrus.Error(err)
		res.Code = http.StatusBadRequest
		res.Message = err.Error()
		c.SecureJSON(http.StatusBadRequest, res)
		return
	}

	err = checkAddr(accountAddr.String())
	if err != nil {
		logrus.Error(err)
		res.Code = http.StatusBadRequest
		res.Message = err.Error()
		c.SecureJSON(http.StatusBadRequest, res)
		return
	}

	instance, _ := util.PrepareTx(a.config, contractAddr.String())

	FrozenAmount, err := instance.FrozenOf(nil, common.HexToAddress(accountAddr.String()))
	if err != nil && err.Error() != "abi: attempting to unmarshall an empty string while arguments are expected" {
		logrus.Error(err)
		res.Code = http.StatusInternalServerError
		res.Message = err.Error()
		c.SecureJSON(http.StatusInternalServerError, res)
		return
	}

	res.Code = Ok
	res.Message = "success"
	res.Data = FrozenAmount.String()

	c.SecureJSON(http.StatusOK, res)
}

func (a *ApiService) blackRange(c *gin.Context) {
	buf := make([]byte, 1024)
	n, _ := c.Request.Body.Read(buf)
	data1 := string(buf[0:n])
	res := types.HttpRes{}

	isValid := gjson.Valid(data1)
	if isValid == false {
		logrus.Error("Not valid json")
		res.Code = http.StatusBadRequest
		res.Message = "Not valid json"
		c.SecureJSON(http.StatusBadRequest, res)
		return
	}
	contractAddr := gjson.Get(data1, "contractAddr")

	err := checkAddr(contractAddr.String())
	if err != nil {
		logrus.Error(err)
		res.Code = http.StatusBadRequest
		res.Message = err.Error()
		c.SecureJSON(http.StatusBadRequest, res)
		return
	}

	instance, _ := util.PrepareTx(a.config, contractAddr.String())

	blackRange, err := instance.BlackBlocks(nil)
	if err != nil && err.Error() != "abi: attempting to unmarshall an empty string while arguments are expected" {
		logrus.Error(err)
		res.Code = http.StatusInternalServerError
		res.Message = err.Error()
		c.SecureJSON(http.StatusInternalServerError, res)
		return
	}

	b, err := json.Marshal(blackRange)
	if err != nil {
		logrus.Error(err)
		res.Code = http.StatusInternalServerError
		res.Message = err.Error()
		c.SecureJSON(http.StatusInternalServerError, res)
		return
	}
	res.Code = Ok
	res.Message = "success"
	res.Data = string(b)

	c.SecureJSON(http.StatusOK, res)
}

func (a *ApiService) status(c *gin.Context) {
	buf := make([]byte, 1024)
	n, _ := c.Request.Body.Read(buf)
	data1 := string(buf[0:n])
	res := types.HttpRes{}

	isValid := gjson.Valid(data1)
	if isValid == false {
		logrus.Error("Not valid json")
		res.Code = http.StatusBadRequest
		res.Message = "Not valid json"
		c.SecureJSON(http.StatusBadRequest, res)
		return
	}
	contractAddr := gjson.Get(data1, "contractAddr")
	accountAddr := gjson.Get(data1, "accountAddr")

	err := checkAddr(accountAddr.String())
	if err != nil {
		logrus.Error(err)
		res.Code = http.StatusBadRequest
		res.Message = err.Error()
		c.SecureJSON(http.StatusBadRequest, res)
		return
	}

	instance, _ := util.PrepareTx(a.config, contractAddr.String())

	isBlack, err := instance.BlackOf(nil, common.HexToAddress(accountAddr.String()))
	if err != nil && err.Error() != "abi: attempting to unmarshall an empty string while arguments are expected" {
		logrus.Error(err)
		res.Code = http.StatusInternalServerError
		res.Message = err.Error()
		c.SecureJSON(http.StatusInternalServerError, res)
		return
	}
	isBlackIn, err := instance.BlackInOf(nil, common.HexToAddress(accountAddr.String()))
	if err != nil && err.Error() != "abi: attempting to unmarshall an empty string while arguments are expected" {
		logrus.Error(err)
		res.Code = http.StatusInternalServerError
		res.Message = err.Error()
		c.SecureJSON(http.StatusInternalServerError, res)
		return
	}
	isBlackOut, err := instance.BlackOutOf(nil, common.HexToAddress(accountAddr.String()))
	if err != nil && err.Error() != "abi: attempting to unmarshall an empty string while arguments are expected" {
		logrus.Error(err)
		res.Code = http.StatusInternalServerError
		res.Message = err.Error()
		c.SecureJSON(http.StatusInternalServerError, res)
		return
	}
	nowFrozenAmount, err := instance.FrozenOf(nil, common.HexToAddress(accountAddr.String()))
	if err != nil && err.Error() != "abi: attempting to unmarshall an empty string while arguments are expected" {
		logrus.Error(err)
		res.Code = http.StatusInternalServerError
		res.Message = err.Error()
		c.SecureJSON(http.StatusInternalServerError, res)
		return
	}
	waitFrozenAmount, err := instance.WaitFrozenOf(nil, common.HexToAddress(accountAddr.String()))
	if err != nil && err.Error() != "abi: attempting to unmarshall an empty string while arguments are expected" {
		logrus.Error(err)
		res.Code = http.StatusInternalServerError
		res.Message = err.Error()
		c.SecureJSON(http.StatusInternalServerError, res)
		return
	}

	status := types.StatusInfo{
		IsBlack:          isBlack,
		IsBlackIn:        isBlackIn,
		IsBlackOut:       isBlackOut,
		NowFrozenAmount:  nowFrozenAmount,
		WaitFrozenAmount: waitFrozenAmount,
	}

	b, err := json.Marshal(status)
	if err != nil {
		logrus.Error(err)
		res.Code = http.StatusInternalServerError
		res.Message = err.Error()
		c.SecureJSON(http.StatusInternalServerError, res)
		return
	}

	res.Code = Ok
	res.Message = "success"
	res.Data = string(b)

	c.SecureJSON(http.StatusOK, res)
}

func (a *ApiService) model(c *gin.Context) {
	buf := make([]byte, 1024)
	n, _ := c.Request.Body.Read(buf)
	data1 := string(buf[0:n])
	res := types.HttpRes{}

	isValid := gjson.Valid(data1)
	if isValid == false {
		logrus.Error("Not valid json")
		res.Code = http.StatusBadRequest
		res.Message = "Not valid json"
		c.SecureJSON(http.StatusBadRequest, res)
		return
	}
	contractAddr := gjson.Get(data1, "contractAddr")

	instance, _ := util.PrepareTx(a.config, contractAddr.String())

	modelValue, err := instance.Model(nil)
	if err != nil && err.Error() != "no contract code at given address" && err.Error() != "abi: attempting to unmarshall an empty string while arguments are expected" && err.Error() != "execution reverted" {
		logrus.Error(err)
		res.Code = http.StatusInternalServerError
		res.Message = err.Error()
		c.SecureJSON(http.StatusInternalServerError, res)
		return
	}

	res.Code = Ok
	res.Message = "success"
	res.Data = fmt.Sprintf("%d", modelValue)

	c.SecureJSON(http.StatusOK, res)
}

func (a *ApiService) GetTaxFee(c *gin.Context) {
	buf := make([]byte, 1024)
	n, _ := c.Request.Body.Read(buf)
	data1 := string(buf[0:n])
	res := types.HttpRes{}

	isValid := gjson.Valid(data1)
	if isValid == false {
		logrus.Error("Not valid json")
		res.Code = http.StatusBadRequest
		res.Message = "Not valid json"
		c.SecureJSON(http.StatusBadRequest, res)
		return
	}
	contractAddr := gjson.Get(data1, "contractAddr")

	instance, _ := util.PrepareTx(a.config, contractAddr.String())

	taxFee, err := instance.GetTaxFee(nil)
	if err != nil && err.Error() != "no contract code at given address" && err.Error() != "abi: attempting to unmarshall an empty string while arguments are expected" && err.Error() != "execution reverted" {
		logrus.Error(err)
		res.Code = http.StatusInternalServerError
		res.Message = err.Error()
		c.SecureJSON(http.StatusInternalServerError, res)
		return
	}

	res.Code = Ok
	res.Message = "success"
	if taxFee == nil && taxFee.Uint64() == 0 {
		res.Data = "-1"
	} else {
		res.Data = taxFee.String()
	}

	c.SecureJSON(http.StatusOK, res)
}

func (a *ApiService) getFlashFee(c *gin.Context) {
	buf := make([]byte, 1024)
	n, _ := c.Request.Body.Read(buf)
	data1 := string(buf[0:n])
	res := types.HttpRes{}

	isValid := gjson.Valid(data1)
	if isValid == false {
		logrus.Error("Not valid json")
		res.Code = http.StatusBadRequest
		res.Message = "Not valid json"
		c.SecureJSON(http.StatusBadRequest, res)
		return
	}
	contractAddr := gjson.Get(data1, "contractAddr")

	instance, _ := util.PrepareTx(a.config, contractAddr.String())

	fee, err := instance.Fee(nil)
	if err != nil && err.Error() != "no contract code at given address" && err.Error() != "abi: attempting to unmarshall an empty string while arguments are expected" && err.Error() != "execution reverted" {
		logrus.Error(err)
		res.Code = http.StatusInternalServerError
		res.Message = err.Error()
		c.SecureJSON(http.StatusInternalServerError, res)
		return
	}
	if fee == nil {
		res.Data = "-1"
	} else {
		res.Data = fee.String()
	}
	res.Code = Ok
	res.Message = "success"

	c.SecureJSON(http.StatusOK, res)
}

func (a *ApiService) GetBonusFee(c *gin.Context) {
	buf := make([]byte, 1024)
	n, _ := c.Request.Body.Read(buf)
	data1 := string(buf[0:n])
	res := types.HttpRes{}

	isValid := gjson.Valid(data1)
	if isValid == false {
		logrus.Error("Not valid json")
		res.Code = http.StatusBadRequest
		res.Message = "Not valid json"
		c.SecureJSON(http.StatusBadRequest, res)
		return
	}
	contractAddr := gjson.Get(data1, "contractAddr")

	instance, _ := util.PrepareTx(a.config, contractAddr.String())

	bonusFee, err := instance.GetBonusFee(nil)
	if err != nil && err.Error() != "no contract code at given address" && err.Error() != "abi: attempting to unmarshall an empty string while arguments are expected" && err.Error() != "execution reverted" {
		logrus.Error(err)
		res.Code = http.StatusInternalServerError
		res.Message = err.Error()
		c.SecureJSON(http.StatusInternalServerError, res)
		return
	}
	if bonusFee == nil {
		res.Data = "-1"
	} else {
		res.Data = bonusFee.String()
	}
	res.Code = Ok
	res.Message = "success"

	c.SecureJSON(http.StatusOK, res)
}