package main

import (
	"github.com/go-ini/ini"
	"grid_trader/gt"
	"log"
	"os"
)

func main() {
	cfg, err := ini.Load("config.ini")
	if err != nil {
		log.Println("config read error", err)
		os.Exit(1)
	}
	f := cfg.Section("conf")
	gapPercent,err :=f.Key("gapPercent").Float64()
	if err != nil{
		log.Println("gapPercent not float", err)
		os.Exit(1)
	}
	quantity,err :=f.Key("quantity").Float64()
	if err != nil{
		log.Println("quantity not float", err)
		os.Exit(1)
	}
	minQty,err :=f.Key("minQty").Float64()
	if err != nil{
		log.Println("minQty not float", err)
		os.Exit(1)
	}
	minPrice,err :=f.Key("minPrice").Float64()
	if err != nil{
		log.Println("minPrice not float", err)
		os.Exit(1)
	}
	maxOrder,err :=f.Key("maxOrders").Int()
	if err != nil{
		log.Println("minPrice not int", err)
		os.Exit(1)
	}
	_ = gt.New(
		f.Key("symbol").String(),
		f.Key("secret").String(),
		f.Key("apikey").String(),
		gapPercent,
		quantity,
		minPrice,
		minQty,
		maxOrder,
		)
	//for{
	//	bt.GridTrader()
	//	time.Sleep(1 * time.Second)
	//}


}
