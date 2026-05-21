package main

import (
	"encoding/json"
	"fmt"
)

type TriggerBridgeResult struct {
	ConfirmationID string  `json:"confirmationId"`
	DrawdownUSD    float64 `json:"drawdownUSD"`
	FeeUSD         float64 `json:"feeUSD"`
}

func main() {
	payload := []byte(`{"confirmationId":"123","drawdownUSD":100,"feeUSD":10}`)
	
	var res *TriggerBridgeResult
	res = &TriggerBridgeResult{}
	
	// pass res, not &res
	err := json.Unmarshal(payload, res)
	fmt.Printf("Error: %v\n", err)
	fmt.Printf("Result: %+v\n", res)
}
