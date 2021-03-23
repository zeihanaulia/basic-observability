package main

type RegisterPayment struct {
	UberTraceID string   `json:"uber_trace_id,omitempty"`
	SONumber    string   `json:"so_number"`
	TrxID       string   `json:"trx_id"`
	Customer    Customer `json:"customer"`
	Order       []Order  `json:"order"`
}

type Customer struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Address string `json:"address"`
}

type Order struct {
	StoreID int    `json:"store_id"`
	Items   []Item `json:"items"`
}

type Item struct {
	ProductID int     `json:"product_id"`
	SKU       string  `json:"sku"`
	Name      string  `json:"name"`
	Uom       string  `json:"uom"`
	Price     float64 `json:"price"`
	Quantity  int     `json:"quantity"`
}
