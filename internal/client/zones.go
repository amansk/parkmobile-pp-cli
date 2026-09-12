package client

import (
	"fmt"
	"net/http"
	"net/url"
)

// ZoneInfo is a safe subset of zone metadata for CLI output.
type ZoneInfo struct {
	ZoneCode         string  `json:"zone_code"`
	InternalZoneCode string  `json:"internal_zone_code,omitempty"`
	SignageCode      string  `json:"signage_code,omitempty"`
	LocationName     string  `json:"location_name,omitempty"`
	Street           string  `json:"street,omitempty"`
	City             string  `json:"city,omitempty"`
	State            string  `json:"state,omitempty"`
	Latitude         float64 `json:"latitude,omitempty"`
	Longitude        float64 `json:"longitude,omitempty"`
	IsStoppable      *bool   `json:"is_stoppable,omitempty"`
	MaxDurationMins  int     `json:"max_duration_minutes,omitempty"`
	IsParkingAllowed bool    `json:"is_parking_allowed,omitempty"`
	NotAllowedReason string  `json:"parking_not_allowed_reason,omitempty"`
	TimeBlocks       []TimeBlock `json:"time_blocks,omitempty"`
	Raw              map[string]any `json:"-"`
}

type TimeBlock struct {
	Name         string  `json:"name"`
	Unit         string  `json:"unit"`
	MinQuantity  int     `json:"min_quantity"`
	MaxQuantity  int     `json:"max_quantity"`
	TotalPrice   float64 `json:"total_price,omitempty"`
	TimeBlockID  int     `json:"timeblock_id,omitempty"`
}

// GetZone fetches zone info by signage/internal zone code.
// Tries v4 then v3 then legacy paths (verified metadata; auth required).
func (c *Client) GetZone(zoneCode string) (ZoneInfo, error) {
	paths := []string{
		fmt.Sprintf(PathZoneV4, url.PathEscape(zoneCode)),
		fmt.Sprintf(PathZoneV3, url.PathEscape(zoneCode)),
		fmt.Sprintf(PathZoneBySignage, url.PathEscape(zoneCode)),
		fmt.Sprintf(PathZoneByInternal, url.PathEscape(zoneCode)),
	}
	var lastErr error
	for _, path := range paths {
		wire, err := c.doJSONMap(http.MethodGet, path, nil)
		if err != nil {
			lastErr = err
			continue
		}
		if len(wire) == 0 {
			continue
		}
		return mapZoneInfo(zoneCode, wire), nil
	}
	if lastErr != nil {
		return ZoneInfo{}, lastErr
	}
	return ZoneInfo{}, fmt.Errorf("zone %q not found", zoneCode)
}

func mapZoneInfo(code string, wire map[string]any) ZoneInfo {
	z := ZoneInfo{
		ZoneCode: code,
		Raw:      wire,
	}
	z.InternalZoneCode = strField(wire, "internalZoneCode", "InternalZoneCode")
	z.SignageCode = strField(wire, "signageCode", "SignageCode")
	z.LocationName = strField(wire, "locationName", "LocationName", "name", "Name")

	if zi, ok := wire["zoneInfo"].(map[string]any); ok {
		z.Street = strField(zi, "street", "Street")
		z.City = strField(zi, "city", "City")
		z.State = strField(zi, "state", "State")
		z.Latitude = floatField(zi, "latitude", "Latitude")
		z.Longitude = floatField(zi, "longitude", "Longitude")
	}
	if pi, ok := wire["parkInfo"].(map[string]any); ok {
		z.IsParkingAllowed = boolField(pi, "isParkingAllowed", "IsParkingAllowed")
		z.NotAllowedReason = strField(pi, "parkingNotAllowedReason", "ParkingNotAllowedReason")
		if mt, ok := pi["maxParkingTime"].(map[string]any); ok {
			z.MaxDurationMins = intField(mt, "totalMinutes", "TotalMinutes")
		}
		if blocks, ok := pi["timeBlocks"].([]any); ok {
			for _, b := range blocks {
				bm, ok := b.(map[string]any)
				if !ok {
					continue
				}
				tb := TimeBlock{
					Name:        strField(bm, "name", "Name"),
					Unit:        strField(bm, "timeBlockUnit", "TimeBlockUnit"),
					MinQuantity: intField(bm, "minimumValue", "MinimumValue"),
					MaxQuantity: intField(bm, "maximumValue", "MaximumValue"),
					TimeBlockID: intField(bm, "timeblockId", "TimeblockId"),
				}
				if price, ok := bm["priceInfo"].(map[string]any); ok {
					if p, ok := price["price"].(map[string]any); ok {
						tb.TotalPrice = floatField(p, "totalPrice", "TotalPrice")
					}
				}
				z.TimeBlocks = append(z.TimeBlocks, tb)
			}
		}
	}
	// Top-level zone object from Parkingactions schema embeds CanStop on sessions; zones may expose via parkInfo.
	if v, ok := wire["canStop"].(bool); ok {
		z.IsStoppable = &v
	}
	return z
}

func floatField(m map[string]any, keys ...string) float64 {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			switch n := v.(type) {
			case float64:
				return n
			case int:
				return float64(n)
			default:
				var f float64
				_, _ = fmt.Sscan(fmt.Sprint(v), &f)
				return f
			}
		}
	}
	return 0
}

// PriceQuote is a parking price preview (GET /v3/parking/price — unverified query params without live session).
type PriceQuote struct {
	ZoneCode         string  `json:"zone_code"`
	DurationMinutes  int     `json:"duration_minutes"`
	TotalPrice       float64 `json:"total_price"`
	ParkingPrice     float64 `json:"parking_price"`
	ServiceFee       float64 `json:"service_fee"`
	IsParkingAllowed bool    `json:"is_parking_allowed"`
	NotAllowedReason string  `json:"not_allowed_reason,omitempty"`
	OrderToken       string  `json:"order_token,omitempty"`
}

// GetPriceQuote requests a price quote for zone parking.
// Query param shape is unverified — derived from ParkingPriceInfoRequest CSV metadata.
func (c *Client) GetPriceQuote(zoneCode string, durationMinutes int, orderToken string) (PriceQuote, error) {
	q := url.Values{}
	q.Set("zoneCode", zoneCode)
	if durationMinutes > 0 {
		q.Set("durationInMinutes", fmt.Sprintf("%d", durationMinutes))
	}
	if orderToken != "" {
		q.Set("orderToken", orderToken)
	}
	path := PathPriceV3 + "?" + q.Encode()
	wire, err := c.doJSONMap(http.MethodGet, path, nil)
	if err != nil {
		return PriceQuote{}, err
	}
	out := PriceQuote{ZoneCode: zoneCode, DurationMinutes: durationMinutes}
	if p, ok := wire["price"].(map[string]any); ok {
		out.TotalPrice = floatField(p, "totalPrice", "TotalPrice")
		out.ParkingPrice = floatField(p, "parkingPrice", "ParkingPrice")
		out.ServiceFee = floatField(p, "serviceFee", "ServiceFee")
	}
	out.IsParkingAllowed = boolField(wire, "isParkingAllowed", "IsParkingAllowed")
	out.NotAllowedReason = strField(wire, "parkingNotAllowedReason", "ParkingNotAllowedReason")
	out.OrderToken = strField(wire, "orderToken", "OrderToken")
	return out, nil
}
