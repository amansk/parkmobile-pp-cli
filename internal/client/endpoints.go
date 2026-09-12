package client

// DefaultBaseURL is the Phonixx consumer ServiceStack API (verified public metadata Sep 2026).
// Not developer.parkmobile.io (partner/commercial).
const DefaultBaseURL = "https://parkmobile.us/ParkmobileApi"

// Verified REST paths from https://parkmobile.us/ParkmobileApi/csv/metadata
const (
	PathLocations        = "/locations"
	PathIdentify2        = "/account/identify2"
	PathVehicles         = "/account/vehicles"
	PathPaymentMethods   = "/account/paymentmethods"
	PathZoneV4           = "/v4/parking/zone/%s"
	PathZoneV3           = "/v3/parking/zone/%s"
	PathZoneBySignage    = "/parking/zones/%s"
	PathZoneByInternal   = "/parking/zone/%s"
	PathPriceV3          = "/v3/parking/price"
	PathHistoryV2        = "/v2/parking/history"
	PathHistory          = "/parking/history"
	PathActiveV3         = "/v3/parking/active"
	PathExtensionV3      = "/v3/extension/active"
	PathActiveStop       = "/parking/active/%s"
)
