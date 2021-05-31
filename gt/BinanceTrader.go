package gt

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	req "github.com/NoBugBoy/httpgo/http"
	"grid_trader/enum"
	"log"
	"math"
	"reflect"
	"sort"
	"strconv"
	"sync"
	"time"
)
var bt *BinanceTrader
func init() {
	//初始化交易对象
	httpReq := &req.Req{}
	bt = &BinanceTrader{}
	bt.lock = sync.Mutex{}
	bt.httpClient = httpReq
	bt.orderCount = 1_000_000
}

// New 初始化参数
func New(symbol,secret,apikey string,gapPercent,quantity,minPrice,minQty float64,maxOrder int) *BinanceTrader {
	bt.symbol = symbol
	bt.gapPercent = gapPercent
	bt.quantity = quantity
	bt.minPrice = minPrice
	bt.minQty = minQty
	bt.secret = secret
	bt.apikey = apikey
	bt.maxOrders = maxOrder
	log.Println(
		"\n交易对:",symbol,
		"\napiKey:",apikey,
		"\napiSecret:",secret,
		"\n网格交易的价格间隙:",gapPercent,
		"\n每次下单的数量:",quantity,
		"\n价格波动的最小单位:",minPrice,
		"\n最小的下单量:",minQty,
		"\n单边的下单量:",maxOrder,
		)
	return bt
}

type BinanceTrader struct {
	secret string
	apikey string
	buyOrders []*Order
	sellOrders []*Order
	deleteOrder []*Order
	httpClient *req.Req
	symbol string
	quantity float64
	minQty float64
	maxOrders int
	gapPercent float64
	minPrice float64
	lock sync.Mutex
	orderCount int
}
type Order struct {
	symbol string
	orderId int
	orderListId int
	clientOrderId string
	transactTime int
	price float64
	origQty float64
	executedQty float64
	cummulativeQuoteQty float64
	status string
	timeInForce string
	types string `json:"type"`
	side string
	fills []Info
}
type Info struct {
	price float64
	qty float64
	commission float64
	commissionAsset string
}

func (bt *BinanceTrader) getBidAskPrice() (float64,float64) {
	if bt.symbol == "" {
		panic(errors.New("symbol is blank"))
	}
	ticker,err := bt.GetTicker(bt.symbol)
	if err != nil{
		panic(err)
	}
	jsons := bt.strToMap(ticker)
	log.Printf("%v",jsons)
	bidPrice, _ := strconv.ParseFloat(jsons["bidPrice"].(string),64)
	askPrice, _ := strconv.ParseFloat(jsons["askPrice"].(string),64)
	return bidPrice,askPrice
}

func Sort(slice []*Order)  {
	sort.SliceStable(slice, func(i, j int) bool {
		if slice[j].price > slice[i].price {
			return true
		}
		return false
	})
}

func (bt *BinanceTrader) GridTrader(){
	bidPrice,askPrice :=bt.getBidAskPrice()
	log.Println("bidPrice=",bidPrice," askPrice=",askPrice)
	quantity := round(bt.quantity,bt.minQty)
	bt.quantity = quantity
	Sort(bt.buyOrders)
	Sort(bt.sellOrders)
	buyDeleteOrders := make([]*Order,1)
	selldeleteOrders := make([]*Order,1)

	//买单逻辑,检查成交的情况.
	for _, order := range bt.buyOrders {
		val := bt.getOrder(order.symbol,order.clientOrderId)
		order := &Order{}
		err := json.Unmarshal([]byte(val), order)
		if err != nil{
			panic(err)
		}
		if order.status == enum.CANCELED {
			buyDeleteOrders = append(buyDeleteOrders, order)
			log.Println("buy order status was canceled:",order.status)
		}else if order.status == enum.FILLED {
			//买单成交 挂卖单
			log.Println("买单成交时间:",time.Now(),"价格:",order.price,"数量:",order.origQty)
			sellPrice := round(order.price * (1 + bt.gapPercent),bt.minPrice)
			if 0 < sellPrice && sellPrice < askPrice {
				sellPrice = round(askPrice,bt.minPrice)
			}
			sellOrder,err := bt.placeOrder(bt.symbol,enum.SELL,enum.LIMIT,"","",bt.quantity,sellPrice,0)
			if err != nil{
				panic(err)
			}
			sellPlaceOrder := &Order{}
			err = json.Unmarshal([]byte(sellOrder), sellPlaceOrder)
			if err != nil{
				panic(err)
			}
			buyDeleteOrders = append(buyDeleteOrders, order)
			bt.sellOrders = append(bt.sellOrders,sellPlaceOrder)

			buyPrice := round(order.price * (1 - bt.gapPercent),bt.minPrice)
			if buyPrice > bidPrice && bidPrice > 0 {
				buyPrice = round(bidPrice,bt.minPrice)
			}
			buyOrder,err := bt.placeOrder(bt.symbol,enum.BUY,enum.LIMIT,"","",bt.quantity,buyPrice,0)
			if err != nil{
				panic(err)
			}
			buyPlaceOrder := &Order{}
			err = json.Unmarshal([]byte(buyOrder), buyPlaceOrder)
			if err != nil{
				panic(err)
			}
			bt.buyOrders = append(bt.buyOrders, buyPlaceOrder)
		}else if order.status == enum.NEW {
			log.Println("buy order status is: New")
		}else{
			log.Println("buy order status is not above options:",order.status)
		}
	}
	for _, order := range buyDeleteOrders {
		for i, buyOrder := range bt.buyOrders {
			if reflect.DeepEqual(order,buyOrder){
				bt.buyOrders = append(bt.buyOrders[:i], bt.buyOrders[i+1:]...)
			}
		}
	}
	//卖单逻辑,检查成交的情况.
	for _, order := range bt.sellOrders {
		val := bt.getOrder(order.symbol,order.clientOrderId)
		order := &Order{}
		err := json.Unmarshal([]byte(val), order)
		if err != nil{
			panic(err)
		}
		if order.status == enum.CANCELED {
			selldeleteOrders = append(selldeleteOrders, order)
			log.Println("sell order status was canceled:",order.status)
		}else if order.status == enum.FILLED {
			//卖单成交 挂买单
			log.Println("卖单成交时间:",time.Now(),"价格:",order.price,"数量:",order.origQty)
			buyPrice := round(order.price * (1 - bt.gapPercent),bt.minPrice)
			if buyPrice > bidPrice && bidPrice > 0 {
				buyPrice = round(bidPrice,bt.minPrice)
			}
			buyOrder,err := bt.placeOrder(bt.symbol,enum.BUY,enum.LIMIT,"","",bt.quantity,buyPrice,0)
			if err != nil{
				panic(err)
			}
			buyPlaceOrder := &Order{}
			err = json.Unmarshal([]byte(buyOrder), buyPlaceOrder)
			if err != nil{
				panic(err)
			}
			bt.buyOrders = append(bt.buyOrders, buyPlaceOrder)
			selldeleteOrders = append(selldeleteOrders, order)

			sellPrice := round(order.price * (1 + bt.gapPercent),bt.minPrice)
			if 0 < sellPrice && sellPrice < askPrice {
				sellPrice = round(askPrice,bt.minPrice)
			}
			sellOrder,err := bt.placeOrder(bt.symbol,enum.SELL,enum.LIMIT,"","",bt.quantity,sellPrice,0)
			if err != nil{
				panic(err)
			}
			sellPlaceOrder := &Order{}
			err = json.Unmarshal([]byte(sellOrder), sellPlaceOrder)
			if err != nil{
				panic(err)
			}
			bt.sellOrders = append(bt.sellOrders,sellPlaceOrder)
		}else if order.status == enum.NEW {
			log.Println("buy order status is: New")
		}else{
			log.Println("buy order status is not above options:",order.status)
		}

	}
	for _, order := range selldeleteOrders {
		for i, sellOrder := range bt.sellOrders {
			if reflect.DeepEqual(order,sellOrder){
				bt.sellOrders = append(bt.sellOrders[:i], bt.sellOrders[i+1:]...)
			}
		}
	}


	//无买单
	if len(bt.buyOrders) <= 0 {
		if bidPrice > 0 {
			price := round(bidPrice * (1 - bt.gapPercent),bt.minPrice)
			buyOrder,err := bt.placeOrder(bt.symbol,enum.BUY,enum.LIMIT,"","",quantity,price,0)
			if err != nil{
				panic(err)
			}
			order := &Order{}
			err = json.Unmarshal([]byte(buyOrder), order)
			if err != nil{
				panic(err)
			}
			if order.clientOrderId != "" && order.price >= 0{
				log.Printf("买入[买单] %v",order)
				bt.buyOrders = append(bt.buyOrders, order)
			}
		}
	}else if len(bt.buyOrders) > bt.maxOrders {
		bt.deleteOrder = append(bt.deleteOrder, bt.buyOrders[0])
		if bt.cancelOrder(bt.buyOrders[0].symbol,bt.buyOrders[0].clientOrderId) {
			bt.deleteOrder = append(bt.deleteOrder[:1], bt.deleteOrder[2:]...)
		}
	}
	//无卖单
	if len(bt.sellOrders) <= 0{
		if askPrice > 0 {
			price := round(askPrice * (1 + bt.gapPercent),bt.minPrice)
			sellOrder,err := bt.placeOrder(bt.symbol,enum.SELL,enum.LIMIT,"","",quantity,price,0)
			if err != nil{
				panic(err)
			}
			order := &Order{}
			err = json.Unmarshal([]byte(sellOrder), order)
			if err != nil{
				panic(err)
			}
			if order.clientOrderId != "" && order.price >= 0{
				log.Printf("买入[卖单] %v",order)
				bt.sellOrders = append(bt.sellOrders, order)
			}
		}
	}else if len(bt.sellOrders) > bt.maxOrders {
		bt.deleteOrder = append(bt.deleteOrder, bt.sellOrders[0])
		if bt.cancelOrder(bt.sellOrders[0].symbol,bt.sellOrders[0].clientOrderId) {
			bt.deleteOrder = append(bt.deleteOrder[:1], bt.deleteOrder[2:]...)
		}
	}
}
func (bt *BinanceTrader) strToMap(key string) map[string]interface{} {
	var mapResult map[string]interface{}
	err := json.Unmarshal([]byte(key), &mapResult)
	if err != nil{
		panic(err)
	}
	return mapResult
}

func (bt *BinanceTrader) getClientOrderId() string {
	bt.lock.Lock()
	defer bt.lock.Unlock()
	bt.orderCount += 1
	return "x-GRIDREADERGO" + strconv.Itoa(bt.currentTime()) + strconv.Itoa(bt.orderCount)
}

func (bt *BinanceTrader) currentTime() int {
	return int(time.Now().Unix() * 1000)
}

func (bt *BinanceTrader) sign (secret string, data string) string {
		h := hmac.New(sha256.New, []byte(secret))
		h.Write([]byte(data))
		return hex.EncodeToString(h.Sum(nil))
}

func round(f1, f2 float64) float64 {
	return math.Round(f1 / f2) * f2
}


