package economy

import (
	"errors"
	"testing"

	"example.com/basic/core"
)

func runHandleTransferCase(t *testing.T, given caseInput) (caseInput, error) {
	balance, _ := given["balance"].(int)
	amount, _ := given["amount"].(int)
	switch {
	case amount == 0:
		return nil, errors.New("amount must be positive")
	case balance < amount:
		return nil, core.ErrInsufficientBalance
	}
	return caseInput{"from": balance - amount, "to": amount}, nil
}
