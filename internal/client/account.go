package client

import (
	"fmt"
	"net/http"
)

// AccountProfile is a safe subset of GET /account/identify2.
type AccountProfile struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
	Mobile    string `json:"mobile"`
	UserName  string `json:"userName"`
}

// Vehicle is a registered vehicle (ids + plate only in CLI output).
type Vehicle struct {
	VehicleID   int    `json:"vehicleId"`
	VRN         string `json:"vrn"`
	State       string `json:"state"`
	Country     string `json:"country"`
	Description string `json:"description"`
	Default     bool   `json:"default"`
}

// PaymentMethod exposes id + last4 only.
type PaymentMethod struct {
	BillingMethodID int    `json:"billingMethodId"`
	Last4           string `json:"last4"`
	CardType        string `json:"cardType"`
	IsDefault       bool   `json:"isDefault"`
	ExpiryMonth     int    `json:"expiryMonth"`
	ExpiryYear      int    `json:"expiryYear"`
}

func (c *Client) GetAccount() (AccountProfile, error) {
	var wire map[string]any
	if err := c.doJSON(http.MethodGet, PathIdentify2, nil, &wire); err != nil {
		return AccountProfile{}, err
	}
	return mapAccountProfile(wire), nil
}

func mapAccountProfile(wire map[string]any) AccountProfile {
	return AccountProfile{
		FirstName: strField(wire, "firstName", "FirstName"),
		LastName:  strField(wire, "lastName", "LastName"),
		Email:     strField(wire, "email", "Email"),
		Mobile:    strField(wire, "mobile", "Mobile"),
		UserName:  strField(wire, "userName", "UserName"),
	}
}

func (c *Client) ListVehicles() ([]Vehicle, error) {
	var wire map[string]any
	if err := c.doJSON(http.MethodGet, PathVehicles, nil, &wire); err != nil {
		return nil, err
	}
	return mapVehicles(wire), nil
}

func mapVehicles(wire map[string]any) []Vehicle {
	raw := firstSlice(wire, "vehicles", "Vehicles", "results", "Results")
	out := make([]Vehicle, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, Vehicle{
			VehicleID:   intField(m, "vehicleId", "VehicleId"),
			VRN:         strField(m, "vrn", "VRN", "lpn", "Lpn"),
			State:       strField(m, "state", "State"),
			Country:     strField(m, "country", "Country"),
			Description: strField(m, "description", "Description"),
			Default:     boolField(m, "default", "Default"),
		})
	}
	return out
}

func (c *Client) ListPaymentMethods() ([]PaymentMethod, error) {
	var wire map[string]any
	if err := c.doJSON(http.MethodGet, PathPaymentMethods, nil, &wire); err != nil {
		return nil, err
	}
	return mapPaymentMethods(wire), nil
}

func mapPaymentMethods(wire map[string]any) []PaymentMethod {
	raw := firstSlice(wire, "paymentMethods", "PaymentMethods", "billingMethods", "BillingMethods", "results")
	out := make([]PaymentMethod, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, PaymentMethod{
			BillingMethodID: intField(m, "billingMethodId", "BillingMethodId", "id", "Id"),
			Last4:           strField(m, "last4", "Last4", "numberLast4", "NumberLast4"),
			CardType:        strField(m, "cardType", "CardType", "type", "Type"),
			IsDefault:       boolField(m, "isDefault", "IsDefault", "default", "Default"),
			ExpiryMonth:     intField(m, "expiryMonth", "ExpiryMonth"),
			ExpiryYear:      intField(m, "expiryYear", "ExpiryYear"),
		})
	}
	return out
}

func strField(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			return fmt.Sprint(v)
		}
	}
	return ""
}

func intField(m map[string]any, keys ...string) int {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			switch n := v.(type) {
			case float64:
				return int(n)
			case int:
				return n
			case int64:
				return int(n)
			default:
				var i int
				_, _ = fmt.Sscan(fmt.Sprint(v), &i)
				return i
			}
		}
	}
	return 0
}

func boolField(m map[string]any, keys ...string) bool {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			switch b := v.(type) {
			case bool:
				return b
			default:
				return fmt.Sprint(b) == "true"
			}
		}
	}
	return false
}

func firstSlice(m map[string]any, keys ...string) []any {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if arr, ok := v.([]any); ok {
				return arr
			}
		}
	}
	return nil
}
