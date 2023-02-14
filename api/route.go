package api

import (
	"fmt"
	"github.com/ethereum/coin-manage/config"
	"github.com/ethereum/coin-manage/db"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
)

type ApiService struct {
	conns  map[string]*db.Mysql
	config *config.Config
}

func NewApiService(conns map[string]*db.Mysql, cfg *config.Config) *ApiService {
	return &ApiService{
		conns:  conns,
		config: cfg,
	}
}

func (a *ApiService) Run() {
	r := gin.Default()

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowCredentials = true
	corsConfig.AllowOrigins = []string{"*"}
	r.Use(func(ctx *gin.Context) {
		method := ctx.Request.Method
		ctx.Header("Access-Control-Allow-Origin", "*")
		ctx.Header("Access-Control-Allow-Headers", "*")
		// ctx.Header("Access-Control-Allow-Headers", "Content-Type,addr,GoogleAuth,AccessToken,X-CSRF-Token,Authorization,Token,token,auth,x-token")
		ctx.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		ctx.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Content-Type")
		ctx.Header("Access-Control-Allow-Credentials", "true")
		if method == "OPTIONS" {
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}
		ctx.Next()
	})

	//查询指定的代币信息
	r.GET("/token/detail/:contractAddr/:chainName", a.getSpecifyCoinInfo)
	//查询代币详情列表
	r.GET("/token/list/:accountAddr/:chainName", a.getCoinInfos)
	//查询账户的所有代币的持币地址总数
	r.GET("/account/count/:accountAddr/:chainName", a.getAllCoinAllCount)
	//查询代币的持有者信息
	r.GET("/token/holderInfos/:contractAddr/:chainName", a.getCoinHolders)
	//查询账户的代币余额
	r.GET("/account/tokenBalance/:accountAddr/:contractAddr/:chainName", a.getCoinBalance)
	//查询代币的持有者数量
	r.GET("/token/holderCount/:contractAddr/:chainName", a.getCoinHoldersCount)
	//查询账户的交易记录
	r.GET("/account/txHistory/:accountAddr/:contractAddr/:beginTime/:endTime/:chainName", a.getTxHistory)
	//查询账户下指定代笔的燃烧数量
	r.GET("/account/burnAmount/:accountAddr/:contractAddr/:chainName", a.hasBurnAmount)
	//查询链的高度
	r.GET("/chain/height/:chainName", a.getBlockHeight)
	//查询的代币的初始发行和增发历史
	r.GET("/token/history/:contractAddr/:chainName", a.getCoinHistory)

	//创建一个多签任务
	r.POST("/multiSign/create", a.addmultisign)

	//写合约
	//禁止账户交易-加入黑名单
	r.POST("/account/addBlack", a.addBlack)
	//允许账户交易-移出黑名单
	r.POST("/account/removeBlack", a.removeBlack)
	//禁止账户转入交易
	r.POST("/account/addBlackIn", a.addBlackIn)
	//允许账户转入交易
	r.POST("/account/removeBlackIn", a.removeBlackIn)
	//禁止账户转出交易
	r.POST("/account/addBlackOut", a.addBlackOut)
	//允许账户转出交易
	r.POST("/account/removeBlackOut", a.removeBlackOut)

	//冻结
	r.POST("/token/frozen", a.frozen)
	//解冻
	r.POST("/token/unfrozen", a.unfrozen)

	//禁止区块高度间交易
	r.POST("/token/addBlackRange", a.addBlackRange)
	//允许区块高度间交易
	r.POST("/token/removeBlackRange", a.removeBlackRange)
	//铸造（增发）
	r.POST("/token/mint", a.mint)
	//燃烧
	r.POST("/token/burn", a.burn)
	//燃烧指定账户下的代币
	r.POST("/token/burnFrom", a.burnFrom)

	//读取合约
	//查询代币的状态
	r.POST("/token/status", a.status)
	//查询代币的模型
	r.POST("/token/model", a.model)

	//查询代币的交易费比例
	r.POST("/token/taxFee", a.GetTaxFee)
	//查询代币的奖励分红比例
	r.POST("/token/bonusFee", a.GetBonusFee)

	//查询禁止交易的区块区间
	r.POST("/blackRange", a.blackRange)

	//查询账户的冻结数量
	r.POST("/account/forzenAmount", a.hasForzenAmount)

	//查询代币的总容量
	r.POST("/token/cap", a.cap)
	//查询代币的闪电费
	r.POST("/token/flashFee", a.getFlashFee)

	//对于写合约，查询写合约的交易状态
	r.POST("/tx/get", a.GetTask)

	logrus.Info("coin-manage run at " + a.config.Server.Port)

	err := r.Run(fmt.Sprintf(":%s", a.config.Server.Port))
	if err != nil {
		logrus.Fatalf("start http server err:%v", err)
	}
}
