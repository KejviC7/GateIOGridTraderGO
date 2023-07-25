package main

import (
	"context"
	"fmt"

	"github.com/gateio/gateapi-go/v6"
)

var (
	apiKey    = ""
	secretKey = ""
)

/*************** INITIALIZE CLIENT ****************************/

exchange := binance.NewFuturesClient(apiKey, secretKey)

/* ************* Global variables *****************************/

var LEVERAGE int
var BUY_ORDERS []int64
var SELL_ORDERS []int64

// var CLOSED_ORDER []string
var CLOSED_ORDERS_IDS []int64
var STARTING_BALANCE = 1000.0
var CURRENT_BALANCE = 0.0
var STOP_BALANCE = 800.0
var TAKE_PROFIT_BALANCE = 1200.0
var SYMBOL = "ETC_USDT"
var FIAT_ASSET = "BUSD"
var CONTRACT_SIZE int64 = 1
var GRID_LINES int64 = 5
var THRESHHOLD_POSITION int64 = CONTRACT_SIZE * GRID_LINES * 2.0

// Gridbot Settings
var NUM_BUY_GRID_LINES = GRID_LINES
var NUM_SELL_GRID_LINES = GRID_LINES
var GRID_SIZE = 5.0

//var CHECK_ORDERS_FREQUENCY = 2
//var CLOSED_ORDER_STATUS = "closed"

/* *************************************************************/
var futuresClient = gateapi.NewAPIClient(gateapi.NewConfiguration())

// client.ChangeBasePath(config.BaseUrl)
var ctx = context.WithValue(context.Background(), gateapi.ContextGateAPIV4, gateapi.GateAPIV4{
	Key:    apiKey,
	Secret: secretKey,
})

/* *************** HELPER FUNCTIONS *******************/

func main() {
	
		futuresOrder := gateapi.FuturesOrder{Contract: SYMBOL, Size: CONTRACT_SIZE * 2, Price: "36", Tif: "gtc"}
		response, _, err := futuresClient.FuturesApi.CreateFuturesOrder(ctx, "usdt", futuresOrder)
		if err != nil {
			if e, ok := err.(gateapi.GateAPIError); ok {
				fmt.Printf("gate api error: %s\n", e.Error())
			} else {
				fmt.Printf("generic error: %s\n", err.Error())
			}
		} else {
			fmt.Println(response)
		}
	
	
		res, _, err := futuresClient.FuturesApi.CancelFuturesOrders(ctx, "usdt", SYMBOL, nil)
		if err != nil {
			if e, ok := err.(gateapi.GateAPIError); ok {
				fmt.Printf("gate api error: %s\n", e.Error())
			} else {
				fmt.Printf("generic error: %s\n", err.Error())
			}
		} else {
			fmt.Println(res)
		}
	
	result, _, err := futuresClient.FuturesApi.GetPosition(ctx, "usdt", SYMBOL)
	if err != nil {
		if e, ok := err.(gateapi.GateAPIError); ok {
			fmt.Printf("gate api error: %s\n", e.Error())
		} else {
			fmt.Printf("generic error: %s\n", err.Error())
		}
	} else {
		fmt.Println(result)
	}

	//_, _, err := futuresClient.FuturesApi.
	fmt.Println(futuresClient)
	fmt.Println("Testing")
	
		fmt.Println(" \n======== STARTING GRIDBOT ========")
		fmt.Printf(" \n======== STARTING BALANCE: $%f ========", STARTING_BALANCE)
		CURRENT_BALANCE = get_current_balance()
		fmt.Printf(" ======== CURRENT BALANCE: $%f ========", CURRENT_BALANCE)
		fmt.Println(" \n======== Cancelling all existing orders! ========")
		cancel_all_existing_orders()
		close_all_positions()
		fmt.Println(" \n======== Proceeding to the Main Logic! ========")

		loop_run := 1
		for loop_run > 0 {
			threshold_checker()
			check_take_profit()
			check_stop_condition()
			check_buy_orders()
			check_sell_orders()
			fmt.Println(" \n======== Checking for Open Limit Buy Orders! ======== ")
			check_open_buy_orders()
			fmt.Println(" \n======== Checking for Open Limit Sell Orders! ======== ")
			check_open_sell_orders()
			clear_order_list()
		}
	
}


func check_open_buy_orders() {

	for i := 0; i < len(BUY_ORDERS); i++ {
		fmt.Printf(" \n======== Checking Limit Buy Order %d ======== ", BUY_ORDERS[i])
		order, err := futuresClient.NewGetOrderService().Symbol(SYMBOL).OrderID(BUY_ORDERS[i]).Do(context.Background())

		if err != nil {
			fmt.Println(err)
		}

		if order.Status == "FILLED" {

			CLOSED_ORDERS_IDS = append(CLOSED_ORDERS_IDS, order.OrderID)
			fmt.Printf(" \n======== Limit Buy Order was executed at %s ======== ", order.Price)
			_, new_ask_price := fetch_latest_prices()
			new_sell_price := new_ask_price + GRID_SIZE
			fmt.Printf(" \n************** Creating New Limit Sell Order at %f ***************** ", new_sell_price)
			new_sell_order, err := futuresClient.NewCreateOrderService().Symbol(SYMBOL).
				Side("SELL").Type("LIMIT").
				TimeInForce("GTC").Quantity(fmt.Sprintf("%f", CONTRACT_SIZE)).
				Price(fmt.Sprintf("%f", new_sell_price)).Do(context.Background())

			if err != nil {
				fmt.Println(err)
			}
			SELL_ORDERS = append(SELL_ORDERS, new_sell_order.OrderID)

		}

	}
}

func check_open_sell_orders() {

	for i := 0; i < len(SELL_ORDERS); i++ {
		fmt.Printf(" \n======== Checking Limit Sell Order %d ======== ", SELL_ORDERS[i])
		order, err := futuresClient.NewGetOrderService().Symbol(SYMBOL).OrderID(SELL_ORDERS[i]).Do(context.Background())

		if err != nil {
			fmt.Println(err)
		}

		if order.Status == "FILLED" {

			CLOSED_ORDERS_IDS = append(CLOSED_ORDERS_IDS, order.OrderID)
			fmt.Printf(" \n======== Limit Sell Order was executed at %s ======== ", order.Price)
			new_bid_price, _ := fetch_latest_prices()
			new_buy_price := new_bid_price - GRID_SIZE
			fmt.Printf(" \n************** Creating New Limit Buy Order at %f ***************** ", new_buy_price)
			new_buy_order, err := futuresClient.NewCreateOrderService().Symbol(SYMBOL).
				Side("BUY").Type("LIMIT").
				TimeInForce("GTC").Quantity(fmt.Sprintf("%f", CONTRACT_SIZE)).
				Price(fmt.Sprintf("%f", new_buy_price)).Do(context.Background())

			if err != nil {
				fmt.Println(err)
			}
			BUY_ORDERS = append(BUY_ORDERS, new_buy_order.OrderID)

		}

	}
}

func cancel_all_existing_orders() {
	err := futuresClient.NewCancelAllOpenOrdersService().Symbol(SYMBOL).Do(context.Background())
	if err != nil {
		fmt.Println(err)
	}

}

func create_buy_orders() {
	_, ask_price := fetch_latest_prices()
	for i := 0.0; i < GRID_LINES; i++ {
		//bid_price, ask_price := fetch_latest_prices()
		price := ask_price - (GRID_SIZE * (i + 1))
		fmt.Printf(" \n======== Submitting market limit buy order at $%f ======== ", price)
		order, err := futuresClient.NewCreateOrderService().Symbol(SYMBOL).
			Side("BUY").Type("LIMIT").
			TimeInForce("GTC").Quantity(fmt.Sprintf("%f", CONTRACT_SIZE)).
			Price(fmt.Sprintf("%f", price)).Do(context.Background())
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(order)
		BUY_ORDERS = append(BUY_ORDERS, order.OrderID)
	}

}

func create_sell_orders() {
	bid_price, _ := fetch_latest_prices()
	for i := 0.0; i < GRID_LINES; i++ {
		//bid_price, ask_price := fetch_latest_prices()
		price := bid_price + (GRID_SIZE * (i + 1))
		fmt.Printf(" \n======== Submitting market limit sell order at $%f ======== ", price)
		order, err := futuresClient.NewCreateOrderService().Symbol(SYMBOL).
			Side("SELL").Type("LIMIT").
			TimeInForce("GTC").Quantity(fmt.Sprintf("%f", CONTRACT_SIZE)).
			Price(fmt.Sprintf("%f", price)).Do(context.Background())
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(order)
		SELL_ORDERS = append(SELL_ORDERS, order.OrderID)
	}

}

func fetch_latest_prices() (float64, float64) {
	order_data, err := futuresClient.NewDepthService().Symbol(SYMBOL).Limit(5).Do(context.Background())
	if err != nil {
		fmt.Println(err)
	}

	bid_price, _ := strconv.ParseFloat(order_data.Bids[0].Price, 64)
	ask_price, _ := strconv.ParseFloat(order_data.Asks[0].Price, 64)

	return bid_price, ask_price
}

func check_buy_orders() {
	if len(BUY_ORDERS) == 0 {
		fmt.Println(" \n======== There are no buy orders currently. Creating the Buy Orders ======== ")
		create_buy_orders()
	} else {
		fmt.Println(" \n======== Buy orders exist. Continue! ======== ")
	}

}

func check_sell_orders() {
	if len(SELL_ORDERS) == 0 {
		fmt.Println(" \n======== There are no buy orders currently. Creating the Buy Orders ======== ")
		create_sell_orders()
	} else {
		fmt.Println(" \n======== Sell orders exist. Continue! ======== ")
	}

}

func clear_order_list() {

	var new_buy_orders []int64
	var new_sell_orders []int64

	for i := 0; i < len(CLOSED_ORDERS_IDS); i++ {
		for j := 0; j < len(BUY_ORDERS); j++ {
			if BUY_ORDERS[j] == CLOSED_ORDERS_IDS[i] {
				continue
			} else {
				new_buy_orders = append(new_buy_orders, BUY_ORDERS[j])
			}
		}
	}

	for i := 0; i < len(CLOSED_ORDERS_IDS); i++ {
		for j := 0; j < len(SELL_ORDERS); j++ {
			if SELL_ORDERS[j] == CLOSED_ORDERS_IDS[i] {
				continue
			} else {
				new_sell_orders = append(new_sell_orders, SELL_ORDERS[j])
			}
		}
	}

	copy(BUY_ORDERS, new_buy_orders)
	copy(SELL_ORDERS, new_sell_orders)

}

func get_current_balance() float64 {

	current_bal, err := futuresClient.NewGetAccountService().Do(context.Background())
	var fiat_balance float64
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(current_bal.TotalWalletBalance)
	//fmt.Println(current_bal.Positions)
	for _, p := range current_bal.Assets {
		if p.Asset == FIAT_ASSET {

			fiat_balance, _ = strconv.ParseFloat(p.WalletBalance, 64)
			//fmt.Printf("The current balance for %s is %s", p.Asset, p.WalletBalance)

		}
	}

	return fiat_balance
}

func check_take_profit() {
	if CURRENT_BALANCE > TAKE_PROFIT_BALANCE {

		fmt.Println(" \n======== TAKE PROFIT REACHED! Closing all Positions and Open Orders")
		cancel_all_existing_orders()
		close_all_positions()
		fmt.Println(" \n======== THE GRID BOT WILL RESTART SOON ========")
		return

	} else {

		fmt.Println(" \n======== TAKE PROFIT CONDITION NOT MET YET. GRIDBOT STILL RUNNING ========")
		return
	}
}

func check_stop_condition() {
	if CURRENT_BALANCE < STOP_BALANCE {

		fmt.Println(" \n======== STOP LOSS REACHED! Closing all Positions and Open Orders")
		cancel_all_existing_orders()
		close_all_positions()
		fmt.Println(" \n======== SHUTTING DOWN THE BOT ========")
		os.Exit(3)
		return

	} else {

		fmt.Println(" \n======== STOP CONDITION NOT MET YET. GRIDBOT STILL RUNNING  ========")
		return
	}
}

func close_all_positions() {

	pos_side, pos_size := fetch_position()
	bid_price, ask_price := fetch_latest_prices()

	if pos_side == "LONG" {

		price := ask_price - 5
		_, err := futuresClient.NewCreateOrderService().Symbol(SYMBOL).
			Side("SELL").Type("LIMIT").
			TimeInForce("GTC").Quantity(fmt.Sprintf("%f", math.Abs(pos_size))).
			Price(fmt.Sprintf("%f", price)).Do(context.Background())

		if err != nil {
			fmt.Println(err)
			return
		}

	} else if pos_side == "SHORT" {

		price := bid_price + 5
		_, err := futuresClient.NewCreateOrderService().Symbol(SYMBOL).
			Side("BUY").Type("LIMIT").
			TimeInForce("GTC").Quantity(fmt.Sprintf("%f", math.Abs(pos_size))).
			Price(fmt.Sprintf("%f", price)).Do(context.Background())

		if err != nil {
			fmt.Println(err)
			return
		}

	} else {
		fmt.Println(" ======== No position side was found! Look into it. ======== ")
	}

}

func fetch_position() (string, float64) {

	var pos_side string
	var pos_size float64

	data, err := futuresClient.NewGetAccountService().Do(context.Background())
	if err != nil {
		fmt.Println(err)
	}

	for _, p := range data.Positions {
		if p.Symbol == SYMBOL {
			pos_size, _ = strconv.ParseFloat(p.PositionAmt, 64)
			if pos_size > 0 {
				//fmt.Println("Current Position Side is Long")
				pos_side = "LONG"
			} else if pos_size < 0 {
				pos_side = "SHORT"
				//fmt.Println("Current Position Side is Short")
			} else {
				fmt.Println("There are no current positions.")
			}
		}
	}

	return pos_side, pos_size
}

func threshold_checker() {
	/*
			We check if the current open position is not oversized.
		    An oversized position can lead to significant losses if the trend continues.
		    Therefore, it is important to have some threshhold in place in order to close/reduce the position and refresh the orders.
	*/

	// Get current position
	pos_side, pos_size := fetch_position()

	if pos_size > THRESHHOLD_POSITION {
		fmt.Printf("\n========== Grid Bot is currently in an oversized %s position. Closing the Position and refreshing the orders. ===========", pos_side)
		cancel_all_existing_orders()
		close_all_positions()
	}

}

