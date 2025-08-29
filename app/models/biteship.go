package models

// Define the request model
type CourierRequest struct {
	OriginAreaID      string `json:"origin_area_id"`
	DestinationAreaID string `json:"destination_area_id"`
	Couriers          string `json:"couriers"`
	Items             []Item `json:"items"`
}

type Item struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Value       int    `json:"value"`
	Length      int    `json:"length"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Weight      int    `json:"weight"`
	Quantity    int    `json:"quantity"`
}

// Define the response model
type CourierResponse struct {
	Success     bool      `json:"success"`
	Object      string    `json:"object"`
	Message     string    `json:"message"`
	Code        int       `json:"code"`
	Pricing     []Pricing `json:"pricing"`
	Origin      Location  `json:"origin"`
	Destination Location  `json:"destination"`
}

type CourierResponseInstant struct {
	Success     bool             `json:"success"`
	Object      string           `json:"object"`
	Message     string           `json:"message"`
	Code        int              `json:"code"`
	Pricing     []PricingInstant `json:"pricing"`
	Origin      Location         `json:"origin"`
	Destination Location         `json:"destination"`
}

type Pricing struct {
	CourierName        string `json:"courier_name"`
	CourierServiceName string `json:"courier_service_name"`
	Duration           string `json:"duration"`
	Price              int    `json:"price"`
}

type Location struct {
	PostalCode int    `json:"postal_code"`
	Country    string `json:"country_name"`
	City       string `json:"administrative_division_level_2_name"`
}

type CourierInstantRequest struct {
	OriginLatitude       float64 `json:"origin_latitude"`
	OriginLongitude      float64 `json:"origin_longitude"`
	DestinationLatitude  float64 `json:"destination_latitude"`
	DestinationLongitude float64 `json:"destination_longitude"`
	Couriers             string  `json:"couriers"`
	Items                []Item  `json:"items"`
}

type PricingInstant struct {
	Company              string `json:"company"`
	CourierName          string `json:"courier_name"`
	CourierServiceName   string `json:"courier_service_name"`
	Price                int    `json:"price"`
	Duration             string `json:"duration"`
	ShipmentDurationUnit string `json:"shipment_duration_unit"`
}

type OrderParams struct {
	ShipperContactName      string                  `json:"shipper_contact_name"`
	ShipperContactPhone     string                  `json:"shipper_contact_phone"`
	ShipperContactEmail     string                  `json:"shipper_contact_email"`
	ShipperOrganization     string                  `json:"shipper_organization"`
	OriginContactName       string                  `json:"origin_contact_name"`
	OriginContactPhone      string                  `json:"origin_contact_phone"`
	OriginAddress           string                  `json:"origin_address"`
	OriginNote              string                  `json:"origin_note"`
	OriginCoordinate        Coordinate              `json:"origin_coordinate"`
	OriginPostalCode        int                     `json:"origin_postal_code"`
	DestinationContactName  string                  `json:"destination_contact_name"`
	DestinationContactPhone string                  `json:"destination_contact_phone"`
	DestinationContactEmail string                  `json:"destination_contact_email"`
	DestinationAddress      string                  `json:"destination_address"`
	DestinationNote         string                  `json:"destination_note"`
	DestinationCoordinate   Coordinate              `json:"destination_coordinate"`
	DestinationPostalCode   string                  `json:"destination_postal_code"`
	CourierCompany          string                  `json:"courier_company"`
	CourierType             string                  `json:"courier_type"`
	CourierInsurance        int                     `json:"courier_insurance"`
	DeliveryType            string                  `json:"delivery_type"`
	OrderNote               string                  `json:"order_note"`
	Items                   []OrderItemTestimonials `json:"items"`
}

type Coordinate struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// type OrderResponse struct {
// 	Success bool   `json:"success"`
// 	Message string `json:"message"`
// 	ID      string `json:"id"`
// 	Price   int    `json:"price"`
// 	Status  string `json:"status"`
// }
type OrderResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Object  string `json:"object"`
	ID      string `json:"id"`
	Shipper struct {
		Name         string `json:"name"`
		Email        string `json:"email"`
		Phone        string `json:"phone"`
		Organization string `json:"organization"`
	} `json:"shipper"`
	Origin struct {
		ContactName  string `json:"contact_name"`
		ContactPhone string `json:"contact_phone"`
		Coordinate   struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		} `json:"coordinate"`
		Address          string `json:"address"`
		Note             string `json:"note"`
		PostalCode       int    `json:"postal_code"`
		CollectionMethod string `json:"collection_method"`
	} `json:"origin"`
	Destination struct {
		ContactName     string `json:"contact_name"`
		ContactPhone    string `json:"contact_phone"`
		ContactEmail    string `json:"contact_email"`
		Address         string `json:"address"`
		Note            string `json:"note"`
		ProofOfDelivery struct {
			Use  bool   `json:"use"`
			Fee  int    `json:"fee"`
			Note string `json:"note"`
			Link string `json:"link"`
		} `json:"proof_of_delivery"`
		CashOnDelivery struct {
			ID            string `json:"id"`
			Amount        int    `json:"amount"`
			Fee           int    `json:"fee"`
			Note          string `json:"note"`
			Type          string `json:"type"`
			Status        string `json:"status"`
			PaymentStatus string `json:"payment_status"`
			PaymentMethod string `json:"payment_method"`
		} `json:"cash_on_delivery"`
		Coordinate struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		} `json:"coordinate"`
		PostalCode int `json:"postal_code"`
	} `json:"destination"`
	Stops   []interface{} `json:"stops"`
	Courier struct {
		TrackingID string `json:"tracking_id"`
		WaybillID  string `json:"waybill_id"`
		Company    string `json:"company"`
		Name       string `json:"name"`
		Phone      string `json:"phone"`
		Type       string `json:"type"`
		Link       string `json:"link"`
		Insurance  struct {
			Amount int    `json:"amount"`
			Fee    int    `json:"fee"`
			Note   string `json:"note"`
		} `json:"insurance"`
		RoutingCode string `json:"routing_code"`
	} `json:"courier"`
	Delivery struct {
		Datetime     string  `json:"datetime"`
		Note         string  `json:"note"`
		Type         string  `json:"type"`
		Distance     float64 `json:"distance"`
		DistanceUnit string  `json:"distance_unit"`
	} `json:"delivery"`
	ReferenceID string `json:"reference_id"`
	Items       []struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Category    string  `json:"category"`
		SKU         string  `json:"sku"`
		Value       float64 `json:"value"`
		Quantity    int     `json:"quantity"`
		Length      float64 `json:"length"`
		Width       float64 `json:"width"`
		Height      float64 `json:"height"`
		Weight      float64 `json:"weight"`
	} `json:"items"`
	Extra    []interface{}          `json:"extra"`
	Price    int                    `json:"price"`
	Metadata map[string]interface{} `json:"metadata"`
	Note     string                 `json:"note"`
	Status   string                 `json:"status"`
}

type OrderItemTestimonials struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Value       int    `json:"value"`
	Quantity    int    `json:"quantity"`
	Height      int    `json:"height"`
	Length      int    `json:"length"`
	Weight      int    `json:"weight"`
	Width       int    `json:"width"`
}

type TrackingResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	Object    string `json:"object"`
	ID        string `json:"id"`
	WaybillID string `json:"waybill_id"`
	Courier   struct {
		Company     string `json:"company"`
		Name        string `json:"name"`
		Phone       string `json:"phone"`
		DriverName  string `json:"driver_name"`
		DriverPhone string `json:"driver_phone"`
	} `json:"courier"`
	Destination struct {
		ContactName string `json:"contact_name"`
		Address     string `json:"address"`
	} `json:"destination"`
	History []struct {
		Note        string `json:"note"`
		ServiceType string `json:"service_type"`
		Status      string `json:"status"`
		UpdatedAt   string `json:"updated_at"`
	} `json:"history"`
	Link    string `json:"link"`
	OrderID string `json:"order_id"`
	Origin  struct {
		ContactName string `json:"contact_name"`
		Address     string `json:"address"`
	} `json:"origin"`
	Status string `json:"status"`
}
