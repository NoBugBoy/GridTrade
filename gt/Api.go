package gt

import (
	req "github.com/NoBugBoy/httpgo/http"
	"grid_trader/enum"
	"log"
	"net/http"
	"strings"
)

const host = "https://api.binance.com"

// GetTicker :param symbol: 交易对
//:return: 返回的数据如下:
//{
//'symbol': 'BTCUSDT', 'bidPrice': '9168.50000000', 'bidQty': '1.27689900',
//'askPrice': '9168.51000000', 'askQty': '0.93307800'
//}
func (bt *BinanceTrader)GetTicker(symbol string) (string,error) {
	return bt.httpClient.
		Method(http.MethodGet).
		Header("X-MBX-APIKEY",bt.apikey).
		Url(host + "/api/v3/ticker/bookTicker").
		Retry(3).
		Timeout(90).
		Params(req.Query{
			"symbol":symbol,
	    }).
		Go().
	    Body()
}
//:param symbol: 交易对名称
//:param order_side: 买或者卖， BUY or SELL
//:param order_type: 订单类型 LIMIT or other order type.
//:param quantity: 数量
//:param price: 价格.
//:param client_order_id: 用户的订单ID
//:param time_inforce:
//:param stop_price:
//:return:

func (bt *BinanceTrader) placeOrder(symbol ,orderSide ,orderType ,clientOrderId ,timeInForce string,quantity ,price ,stopPrice float64) (string,error) {
	if clientOrderId == "" {
		clientOrderId = bt.getClientOrderId()
	}
	query := make(req.Query)
	query["symbol"] = symbol
	query["side"] = orderSide
	query["type"] = orderType
	query["quantity"] = quantity
	query["recvWindow"] = 10_000
	query["timestamp"] = bt.currentTime()
	query["newClientOrderId"] = clientOrderId
	if orderType == enum.LIMIT {
		query["timeInForce"] = "GTC"
	}
	if orderType == enum.MARKET {
		price = 0
	}else{
        query["price"] = price
    }
	if orderType == enum.STOP {
		if stopPrice <= 0 {
			panic("stopPrice must greater than 0")
		}
		query["stopPrice"] = stopPrice
	}

	param := req.BuildGetParam(query)
	sign := bt.sign(bt.secret,strings.Replace(param,"?","",1))
	param = param + "&signature=" + sign
	return bt.httpClient.
		Method(http.MethodPost).
		//Header("Content-Type", "application/json").
	    Header("X-MBX-APIKEY",bt.apikey).
		Url(host + "/api/v3/order"+param).
		Timeout(30).
		Go().
		Body()

}

func (bt *BinanceTrader) cancelOrder (symbol string,clientOrderId string) bool {
	query := req.Query{}
	query["symbol"] = symbol
	query["timestamp"] = bt.currentTime()
	query["origClientOrderId"] = clientOrderId
	param := req.BuildGetParam(query)
	sign := bt.sign(bt.secret,strings.Replace(param,"?","",1))
	param = param + "&signature=" + sign
	value,err :=bt.httpClient.
		Method(http.MethodDelete).
		//Header("Content-Type", "application/json").
		Header("X-MBX-APIKEY",bt.apikey).
		Url(host + "/api/v3/order"+param).
		Timeout(30).
		Go().
		Body()
	if err != nil{
		return false
	}
	log.Println(value)
	return true
}

func (bt *BinanceTrader) getOrder(symbol string,clientOrderId string) string {
	query := req.Query{}
	query["symbol"] = symbol
	query["timestamp"] = bt.currentTime()
	query["origClientOrderId"] = clientOrderId
	param := req.BuildGetParam(query)
	value,err :=bt.httpClient.
		Method(http.MethodGet).
		//Header("Content-Type", "application/json").
		Header("X-MBX-APIKEY",bt.apikey).
		Url(host + "/api/v3/order"+param).
		Timeout(30).
		Go().
		Body()
	if err != nil{
		panic(err)
	}
	return value

}

