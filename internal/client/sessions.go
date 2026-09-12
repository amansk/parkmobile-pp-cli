package client

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/amansk/parkmobile-pp-cli/internal/exitcode"
)

// ParkingSession is an active or historical parking session.
type ParkingSession struct {
	ID           int    `json:"id"`
	ZoneCode     string `json:"zone_code"`
	SpaceNumber  string `json:"space_number,omitempty"`
	StartLocal   string `json:"start_local,omitempty"`
	StopLocal    string `json:"stop_local,omitempty"`
	TotalPrice   float64 `json:"total_price,omitempty"`
	CanStop      bool   `json:"can_stop"`
	CanExtend    bool   `json:"can_extend"`
	VehiclePlate string `json:"vehicle_plate,omitempty"`
	VehicleState string `json:"vehicle_state,omitempty"`
	OrderID      int    `json:"order_id,omitempty"`
	Status       string `json:"status,omitempty"`
}

// ListSessions returns parking history (active + recent).
func (c *Client) ListSessions(page, pageSize int) ([]ParkingSession, error) {
	q := url.Values{}
	if page > 0 {
		q.Set("page", fmt.Sprintf("%d", page))
	}
	if pageSize > 0 {
		q.Set("pageSize", fmt.Sprintf("%d", pageSize))
	}
	path := PathHistoryV2
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}
	wire, err := c.doJSONMap(http.MethodGet, path, nil)
	if err != nil {
		// Fallback to v1 history path from metadata.
		path = PathHistory
		if enc := q.Encode(); enc != "" {
			path += "?" + enc
		}
		wire, err = c.doJSONMap(http.MethodGet, path, nil)
		if err != nil {
			return nil, err
		}
	}
	return mapSessions(wire), nil
}

// GetSession fetches one session by id from history list or detail endpoint.
func (c *Client) GetSession(id string) (ParkingSession, error) {
	sessions, err := c.ListSessions(0, 100)
	if err != nil {
		return ParkingSession{}, err
	}
	for _, s := range sessions {
		if fmt.Sprint(s.ID) == id || fmt.Sprint(s.OrderID) == id {
			return s, nil
		}
	}
	return ParkingSession{}, exitcode.NotFoundf("session %q not found", id)
}

func mapSessions(wire map[string]any) []ParkingSession {
	raw := firstSlice(wire, "parkingActions", "ParkingActions", "results", "Results", "history", "History")
	out := make([]ParkingSession, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, mapOneSession(m))
	}
	return out
}

func mapOneSession(m map[string]any) ParkingSession {
	s := ParkingSession{
		ID:          intField(m, "id", "Id"),
		SpaceNumber: strField(m, "spaceNumber", "SpaceNumber"),
		StartLocal:  strField(m, "startLocal", "StartLocal"),
		StopLocal:   strField(m, "stopLocal", "StopLocal"),
		CanStop:     boolField(m, "canStop", "CanStop"),
		CanExtend:   boolField(m, "canExtend", "CanExtend"),
		OrderID:     intField(m, "orderId", "OrderId"),
		Status:      strField(m, "parkingActionType", "ParkingActionType"),
	}
	if zone, ok := m["zone"].(map[string]any); ok {
		s.ZoneCode = strField(zone, "signageCode", "SignageCode", "internalZoneCode", "InternalZoneCode")
	}
	if car, ok := m["car"].(map[string]any); ok {
		s.VehiclePlate = strField(car, "vrn", "VRN")
		s.VehicleState = strField(car, "state", "State")
	}
	if pd, ok := m["priceDetail"].(map[string]any); ok {
		s.TotalPrice = floatField(pd, "totalPrice", "TotalPrice")
	} else if pd, ok := m["PriceDetail"].(map[string]any); ok {
		s.TotalPrice = floatField(pd, "totalPrice", "TotalPrice")
	}
	return s
}
