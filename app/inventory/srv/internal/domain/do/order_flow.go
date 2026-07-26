package do

type OrderFlow struct {
	SellDetail  *StockSellDetailDO
	Adjustments []InventoryAdjustmentDO
	Inventories []InventoryDO
}
